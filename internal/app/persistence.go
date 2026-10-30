package app

import (
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/bencode/jun/internal/config"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

type persistence struct {
	path      string
	enabled   bool
	dirty     bool
	paused    bool
	due       time.Time
	errorOpen bool
	errorText string
	closing   bool
}

func (a *app) restoreConfig(path string) {
	a.persistence = &persistence{path: path, enabled: true}
	state, err := config.Load(path)
	if err == nil {
		a.state = state
		return
	}
	if errors.Is(err, os.ErrNotExist) {
		return
	}
	a.persistence.enabled = false
	a.report(fmt.Errorf("%s: %w", a.text("Configuration could not be loaded; saving is disabled for this session", "配置无法加载；本次运行已停用保存"), err))
}

func (a *app) layoutChanged() {
	p := a.persistence
	if p == nil || !p.enabled {
		return
	}
	p.dirty = true
	p.paused = false
	p.due = time.Time{}
}

func (a *app) autosave(c *ui.Context) {
	p := a.persistence
	if p == nil || !p.enabled || !p.dirty || p.paused || p.errorOpen || a.errorOpen {
		return
	}
	if p.due.IsZero() {
		p.due = c.Now().Add(300 * time.Millisecond)
	}
	if remaining := p.due.Sub(c.Now()); remaining > 0 {
		c.After(remaining)
		return
	}
	if err := a.saveLayout(); err != nil {
		a.saveFailed(err, false)
		c.Invalidate()
	}
}

func (a *app) saveLayout() error {
	a.reapExitedTerminals(a.focusedTerminal)
	p := a.persistence
	if p == nil || !p.enabled || !p.dirty {
		return nil
	}
	if err := config.Save(p.path, a.state); err != nil {
		return err
	}
	p.dirty = false
	p.paused = false
	p.due = time.Time{}
	p.errorOpen = false
	return nil
}

func (a *app) saveFailed(err error, closing bool) {
	log.Printf("save layout: %v", err)
	p := a.persistence
	p.errorText = err.Error()
	p.errorOpen = true
	p.paused = true
	p.closing = closing
	if a.window != nil {
		a.window.Invalidate()
	}
}

func (a *app) saveInBackground() {
	if a.persistence == nil || a.persistence.paused {
		return
	}
	if err := a.saveLayout(); err != nil {
		a.saveFailed(err, false)
	}
}

func (a *app) beforeClose(event *mygo.CloseEvent) {
	if err := a.saveLayout(); err != nil {
		event.PreventDefault()
		a.saveFailed(err, true)
	}
}

func (a *app) retrySave() {
	p := a.persistence
	closing := p.closing
	if err := a.saveLayout(); err != nil {
		a.saveFailed(err, closing)
		return
	}
	p.closing = false
	if closing && a.window != nil {
		a.window.Close()
	}
}

func (a *app) persistenceDialog(c *ui.Context) {
	p := a.persistence
	if p == nil {
		return
	}
	modal := ui.Modal(c, &p.errorOpen, func() {
		ui.Column(c).Width(460).Gap(12).Children(func() {
			ui.Text(c, a.text("Layout could not be saved", "布局无法保存")).Bold()
			ui.Text(c, p.errorText)
			ui.Row(c).FillWidth().Justify(ui.End).Gap(8).Children(func() {
				label := a.text("Continue Editing", "继续编辑")
				if p.closing {
					label = a.text("Cancel Quit", "取消退出")
				}
				ui.Button(c, label).OnClick(func() { p.errorOpen = false; p.closing = false; p.paused = true })
				if p.closing {
					ui.Button(c, a.text("Quit Without Saving", "不保存退出")).OnClick(func() {
						p.enabled = false
						p.errorOpen = false
						p.closing = false
						if a.window != nil {
							a.window.Close()
						}
					})
				}
				ui.PrimaryButton(c, a.text("Retry Save", "重试保存")).OnClick(a.retrySave)
			})
		})
	})
	if modal.Dismissed() {
		p.paused = true
		p.closing = false
	}
}

func (a *app) checkCurrentDirectory() {
	project := a.state.CurrentProject()
	if project == nil {
		return
	}
	info, err := os.Stat(project.CWD)
	if err == nil && !info.IsDir() {
		err = fmt.Errorf("not a directory: %s", project.CWD)
	}
	if err != nil {
		a.report(fmt.Errorf("%s %q (%s): %w", a.text("Project directory unavailable for", "项目目录不可用："), project.Name, project.CWD, err))
	}
}
