package app

import (
	"github.com/bencode/jun/internal/layout"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

type app struct {
	state       *layout.State
	window      *mygo.Window
	theme       *ui.Theme
	chinese     bool
	projectForm struct {
		open              bool
		name, path, error string
	}
	pendingFocus layout.RegionID
	scroll       map[layout.ViewID]*ui.ScrollState
	errorMessage string
	errorOpen    bool
}

func newApp() *app {
	return &app{state: layout.New(), theme: ui.DarkTheme(), scroll: map[layout.ViewID]*ui.ScrollState{}}
}

func Run() error {
	a := newApp()
	mygo.App.SetName("Jun")
	mygo.App.WhenReady(func() {
		mygo.App.SetMenu(a.applicationMenu())
		a.window = mygo.NewWindow(mygo.WindowOptions{
			Title: "Jun", Width: 1440, Height: 900, MinWidth: 1000, MinHeight: 640,
			Content: ui.View(a.view),
		})
	})
	return mygo.App.Run()
}
