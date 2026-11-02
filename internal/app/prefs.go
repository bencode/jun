package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"slices"

	"github.com/bencode/jun/internal/settings"
	"github.com/egoist/mygo"
)

// startPrefs reads the settings file, moves into it the settings earlier
// versions kept in the layout, and applies them.
func (a *app) startPrefs() {
	path, err := settings.Path()
	if err != nil {
		a.report(fmt.Errorf("%s: %w", a.text("Settings file unavailable", "设置文件不可用"), err))
	}
	a.settingsPath = path
	if a.readPrefs() {
		a.migratePrefs()
	} else {
		a.report(fmt.Errorf("%s: %w", a.text("The settings file has an error; using defaults until it is fixed", "设置文件有错误，修正前使用默认设置"), a.prefsErr))
	}
	a.applyPrefs(nil)
}

// readPrefs reads the settings file, keeping the settings as they were
// when it has an error, which the settings tab shows.
func (a *app) readPrefs() bool {
	s, err := settings.Load(a.settingsPath)
	a.prefsErr = err
	if err != nil {
		log.Printf("Jun: settings: %v", err)
		return false
	}
	a.prefs = s
	return true
}

// reloadPrefs reads the settings file again, applies what changed, and
// has the settings tab follow the file.
func (a *app) reloadPrefs() {
	old := a.prefs
	if a.readPrefs() {
		a.applyPrefs(&old)
	}
	a.settingsFileChanged()
	if a.window != nil {
		a.window.Invalidate()
	}
}

// applyPrefs applies the settings that differ from old, or all of them
// when old is nil.
func (a *app) applyPrefs(old *settings.Settings) {
	p := a.prefs
	if old == nil || old.Theme != p.Theme {
		a.useSavedTheme()
	}
	if old == nil || old.Font.Size != p.Font.Size {
		a.terminals.SetTextSize(p.Font.Size)
	}
	if old != nil && old.Language != p.Language {
		mygo.App.SetMenu(a.applicationMenu())
	}
	if old != nil && !slices.Equal(old.Claude.ConfigDirs, p.Claude.ConfigDirs) {
		clear(a.sessionLists) // read again from the new directories
	}
}

// setPref saves a setting Jun's own controls change, and applies it.
func (a *app) setPref(key []string, value any) {
	if err := settings.Set(a.settingsPath, key, value); err != nil {
		a.report(err)
		return
	}
	a.reloadPrefs()
	if a.prefsErr != nil {
		a.report(fmt.Errorf("%s: %w", a.text("Saved, but the settings file has an error, so nothing applies until it is fixed", "已保存，但设置文件有错误，修正前不会生效"), a.prefsErr))
	}
}

// migratePrefs moves into the settings file the settings earlier versions
// kept in the layout, unless the file has its own, and drops them from
// the layout.
func (a *app) migratePrefs() {
	st := a.state
	if !st.Chinese && st.Theme == "" && st.TextSize == 0 {
		return
	}
	moves := []struct {
		key   []string
		value any
		move  bool
	}{
		{[]string{"language"}, "zh", st.Chinese && a.prefs.Language == ""},
		{[]string{"theme"}, st.Theme, st.Theme != "" && a.prefs.Theme == ""},
		{[]string{"font", "size"}, st.TextSize, st.TextSize != 0 && a.prefs.Font.Size == 0},
	}
	for _, m := range moves {
		if !m.move {
			continue
		}
		if err := settings.Set(a.settingsPath, m.key, m.value); err != nil {
			a.report(fmt.Errorf("%s: %w", a.text("Moving settings into the settings file", "将设置移入设置文件"), err))
			return
		}
	}
	st.Chinese, st.Theme, st.TextSize = false, "", 0
	a.layoutChanged()
	a.readPrefs()
}

// watchPrefs reads the settings file again as it changes, while there is
// a window: watching needs the app running.
func (a *app) watchPrefs() {
	if a.window == nil || a.settingsPath == "" {
		return
	}
	dir, name := filepath.Dir(a.settingsPath), filepath.Base(a.settingsPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		log.Printf("Jun: watch settings: %v", err)
		return
	}
	w, err := mygo.WatchFiles(context.Background(), dir, mygo.FileWatchOptions{
		Ignore: func(p string, isDir bool) bool { return !isDir && p != name },
	})
	if err != nil {
		log.Printf("Jun: watch settings: %v", err)
		return
	}
	window := a.window
	go func() {
		for {
			if _, err := w.Next(context.Background()); err != nil {
				if !errors.Is(err, io.EOF) {
					log.Printf("Jun: watch settings: %v", err)
				}
				return
			}
			window.Update(a.reloadPrefs)
		}
	}()
}
