package app

import (
	"fmt"

	"github.com/bencode/jun/internal/config"
	"github.com/bencode/jun/internal/layout"
	"github.com/bencode/jun/internal/terminal"
	"github.com/egoist/mygo"
	ghostty "github.com/egoist/mygo/plugins/terminal"
	"github.com/egoist/mygo/ui"
)

type app struct {
	state       *layout.State
	persistence *persistence
	window      *mygo.Window
	theme       *ui.Theme
	chinese     bool
	projectForm struct {
		open              bool
		name, path, error string
	}
	perspectiveForm struct {
		open        bool
		project     string
		id          layout.PerspectiveID
		name, error string
	}
	terminals      *terminal.Store
	bindings       map[layout.ViewID]terminal.ID
	terminalErrors map[layout.ViewID]error
	activeRegion   layout.RegionID
	pendingFocus   layout.RegionID
	scroll         map[layout.ViewID]*ui.ScrollState
	errorMessage   string
	errorOpen      bool
}

func newApp() *app {
	return &app{state: layout.New(), theme: ui.DarkTheme(), scroll: map[layout.ViewID]*ui.ScrollState{},
		terminals: terminal.NewStore(), bindings: map[layout.ViewID]terminal.ID{}, terminalErrors: map[layout.ViewID]error{}}
}

func Run() error {
	a := newApp()
	mygo.App.SetName("Jun")
	if !mygo.App.RequestSingleInstanceLock() {
		return nil
	}
	if err := ghostty.Load(); err != nil {
		return fmt.Errorf("load terminal library: %w", err)
	}
	defer a.closeTerminals()
	path, err := config.DefaultPath()
	if err != nil {
		a.persistence = &persistence{}
		a.report(fmt.Errorf("%s: %w", a.text("Configuration path unavailable; saving is disabled for this session", "配置路径不可用；本次运行已停用保存"), err))
	} else {
		a.restoreConfig(path)
	}
	mygo.App.OnSecondInstance(func(_ []string, _ string) {
		if a.window == nil {
			return
		}
		if a.window.IsMinimized() {
			a.window.Restore()
		}
		a.window.Show()
		a.window.Focus()
		mygo.App.Focus()
	})
	mygo.App.WhenReady(func() {
		mygo.App.SetMenu(a.applicationMenu())
		a.window = mygo.NewWindow(mygo.WindowOptions{
			Title: "Jun", Width: 1440, Height: 900, MinWidth: 1000, MinHeight: 640,
			Content: ui.View(a.view),
		})
		a.window.OnClose(a.beforeClose)
		a.window.OnClosed(a.closeTerminals)
		a.window.OnBlur(a.saveInBackground)
		a.window.OnHide(a.saveInBackground)
		a.restoreTerminals()
		a.focusCurrentPerspective()
		a.window.Invalidate()
		a.checkCurrentDirectory()
	})
	return mygo.App.Run()
}
