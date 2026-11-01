package app

import (
	"fmt"
	"log"
	"os"

	"github.com/bencode/jun/internal/config"
	"github.com/bencode/jun/internal/history"
	"github.com/bencode/jun/internal/layout"
	"github.com/bencode/jun/internal/terminal"
	"github.com/bencode/jun/internal/theme"
	"github.com/egoist/mygo"
	ghostty "github.com/egoist/mygo/plugins/terminal"
	"github.com/egoist/mygo/ui"
)

type app struct {
	state       *layout.State
	persistence *persistence
	window      *mygo.Window
	theme       *ui.Theme
	projectForm struct {
		open              bool
		name, path, error string
	}
	palette struct {
		open      bool
		query     string
		highlight int
		// caretEnd puts the caret after a query the palette completed, once
		// its input shows it.
		caretEnd  bool
		themes    []string // the themes listed, in theme mode; nil otherwise
		previewed string   // the theme shown while choosing
		reveal    bool     // scrolls the highlight into view once, on opening
	}
	perspectiveForm struct {
		open        bool
		project     string
		id          layout.PerspectiveID
		name, error string
	}
	projectRename struct {
		open                 bool
		project, name, error string
	}
	confirmation    confirmation
	terminals       *terminal.Store
	bindings        map[layout.ViewID]terminal.ID
	terminalErrors  map[layout.ViewID]error
	activeRegion    layout.RegionID
	lastTabs        layout.RegionID // the column or bottom region last focused
	pendingFocus    layout.RegionID
	focusedTerminal layout.ViewID
	scroll          map[layout.ViewID]*ui.ScrollState
	errorMessage    string
	errorOpen       bool
	agents          map[terminal.ID]agentInfo
	ttys            map[terminal.ID]string
	previews        map[terminal.ID]*preview
	agentPolling    bool
	agentViews      map[layout.ViewID]*agentsViewState
	lastTerminal    terminal.ID                       // the session of the terminal last focused
	repos           map[string]*repoState             // Files' repositories, by project directory
	trees           map[layout.ViewID]*filesViewState // Files views' trees
	fileViews       map[layout.ViewID]*fileViewState  // what file views read
	previewTabs     map[layout.RegionID]layout.ViewID // each region's preview tab, which the next file opened replaces
	focusFileSearch bool                              // ⌘P asked to focus Files' search field
	history         history.Scanner                   // agents' sessions, read once each as they change
	sessionLists    map[string]*sessionList           // projects' agent sessions, by project directory
	sessionViews    map[layout.ViewID]*sessionsViewState
	transcripts     map[layout.ViewID]*transcriptState
	diffs           map[layout.ViewID]*diffViewState
}

func newApp(runtimeDir string) *app {
	return &app{state: layout.New(), theme: theme.UI(theme.Default, nil), scroll: map[layout.ViewID]*ui.ScrollState{},
		terminals: terminal.NewStore(runtimeDir), bindings: map[layout.ViewID]terminal.ID{}, terminalErrors: map[layout.ViewID]error{},
		agents: map[terminal.ID]agentInfo{}, ttys: map[terminal.ID]string{}, previews: map[terminal.ID]*preview{},
		agentViews: map[layout.ViewID]*agentsViewState{}, repos: map[string]*repoState{}, trees: map[layout.ViewID]*filesViewState{},
		fileViews: map[layout.ViewID]*fileViewState{}, previewTabs: map[layout.RegionID]layout.ViewID{},
		sessionLists: map[string]*sessionList{}, sessionViews: map[layout.ViewID]*sessionsViewState{}, transcripts: map[layout.ViewID]*transcriptState{},
		diffs: map[layout.ViewID]*diffViewState{}}
}

func Run() error {
	mygo.App.SetName("Jun")
	if !mygo.App.RequestSingleInstanceLock() {
		return nil
	}
	if err := ghostty.Load(); err != nil {
		return fmt.Errorf("load terminal library: %w", err)
	}
	runtimeDir, err := os.MkdirTemp("", "jun-terminals-")
	if err != nil {
		return fmt.Errorf("create runtime directory: %w", err)
	}
	defer func() {
		if err := os.RemoveAll(runtimeDir); err != nil {
			log.Printf("remove runtime directory: %v", err)
		}
	}()
	a := newApp(runtimeDir)
	defer a.closeTerminals()
	path, err := config.DefaultPath()
	if err != nil {
		a.persistence = &persistence{}
		a.report(fmt.Errorf("%s: %w", a.text("Configuration path unavailable; saving is disabled for this session", "配置路径不可用；本次运行已停用保存"), err))
	} else {
		a.restoreConfig(path)
	}
	a.useSavedTheme()
	a.terminals.SetTextSize(a.state.TextSize)
	mygo.App.OnSecondInstance(func(args []string, wd string) {
		if a.window == nil {
			return
		}
		if a.window.IsMinimized() {
			a.window.Restore()
		}
		a.window.Show()
		a.window.Focus()
		mygo.App.Focus()
		a.window.Update(func() { a.openArgs(args, wd) })
	})
	mygo.App.WhenReady(func() {
		mygo.App.SetMenu(a.applicationMenu())
		a.window = mygo.NewWindow(mygo.WindowOptions{
			Title: "Jun", Width: 1440, Height: 900, MinWidth: 1000, MinHeight: 640,
			Content: ui.View(a.view),
		})
		a.matchAppearance()
		stopAgents := make(chan struct{})
		a.window.OnClose(a.beforeClose)
		a.window.OnClosed(func() {
			close(stopAgents)
			a.closePreviews()
			a.closeTerminals()
		})
		a.window.OnBlur(a.saveInBackground)
		a.window.OnHide(a.saveInBackground)
		a.restoreTerminals()
		a.startAgentPoller(stopAgents)
		a.focusCurrentPerspective()
		a.window.Invalidate()
		a.checkCurrentDirectory()
		if wd, err := os.Getwd(); err != nil {
			a.report(fmt.Errorf("working directory: %w", err))
		} else {
			a.openArgs(os.Args[1:], wd)
		}
	})
	return mygo.App.Run()
}
