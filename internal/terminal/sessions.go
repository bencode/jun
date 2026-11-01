package terminal

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/bencode/jun/internal/theme"
	ghostty "github.com/egoist/mygo/plugins/terminal"
	"github.com/egoist/mygo/ui"
)

type ID uint64

type Session struct {
	ID          ID
	ProjectName string
	CWD         string
	Terminal    *ghostty.Terminal
	report      string
	shellPID    int
}

type Store struct {
	nextID     ID
	sessions   []*Session
	runtimeDir string
	theme      *ghostty.Theme
	textSize   float32 // 0 for Ghostty's own
}

// NewStore keeps sessions whose shells report their process IDs in
// runtimeDir.
func NewStore(runtimeDir string) *Store {
	return &Store{runtimeDir: runtimeDir, theme: theme.JunTerminal()}
}

// SetTheme gives the sessions, and those created later, the theme.
func (s *Store) SetTheme(t *ghostty.Theme) {
	s.theme = t
	for _, session := range s.sessions {
		session.Terminal.SetTheme(t, nil)
	}
}

// SetTextSize gives the sessions, and those created later, text of size
// DIPs.
func (s *Store) SetTextSize(size float32) {
	s.textSize = size
	for _, session := range s.sessions {
		session.Terminal.SetFont(ghostty.Font{Size: size})
	}
}

// Theme returns the sessions' theme.
func (s *Store) Theme() *ghostty.Theme { return s.theme }

// shellCommand starts the user's login shell, as the terminal does by
// default, after the shell's process writes its ID to report: exec keeps
// the process, so the ID, the terminal and the exit code are the shell's.
// A shell that cannot run is left to the terminal's default, which fails
// to start it, so that the failure shows rather than closing as an exit.
func shellCommand(report string) []string {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/zsh"
	}
	if _, err := exec.LookPath(shell); err != nil {
		return nil
	}
	return []string{"/bin/zsh", "-fc", `print -r -- $$ > $1; exec -a "-${2:t}" $2`, "jun-shell", report, shell}
}

// ShellPID returns the process ID of the session's shell once it reported it.
func (s *Session) ShellPID() (int, bool) {
	if s.shellPID == 0 {
		b, err := os.ReadFile(s.report)
		if err != nil {
			return 0, false
		}
		pid, err := strconv.Atoi(strings.TrimSpace(string(b)))
		if err != nil {
			return 0, false
		}
		s.shellPID = pid
	}
	return s.shellPID, true
}

func (s *Store) Create(projectName, cwd string, invalidate func()) (*Session, error) {
	if cwd == "" || !filepath.IsAbs(cwd) {
		return nil, fmt.Errorf("terminal directory must be absolute: %q", cwd)
	}
	info, err := os.Stat(cwd)
	if err != nil {
		return nil, fmt.Errorf("terminal directory: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("not a directory: %s", cwd)
	}
	notify := func() {
		if invalidate != nil {
			invalidate()
		}
	}
	report := filepath.Join(s.runtimeDir, strconv.FormatUint(uint64(s.nextID+1), 10))
	term, err := ghostty.New(ghostty.Options{
		Command: shellCommand(report), Dir: cwd, Theme: s.theme, Font: ghostty.Font{Size: s.textSize}, Padding: ghostty.Padding{X: 12, Y: 8}, Scrollback: 10 << 20,
		// ⌘K is Jun's command palette; Control+L clears the screen. ⇧⌘D
		// reviews changes, where Ghostty would split. ⌘+, ⌘− and ⌘0 change
		// Jun's text size, of all terminals and file tabs.
		Keybinds: []ghostty.Keybind{
			{Mods: ui.Super, Key: ui.KeyK, Action: ghostty.ActionNone},
			{Mods: ui.Super | ui.Shift, Key: ui.KeyD, Action: ghostty.ActionNone},
			{Mods: ui.Super, Key: ui.KeyEqual, Action: ghostty.ActionNone},
			{Mods: ui.Super | ui.Shift, Key: ui.KeyEqual, Action: ghostty.ActionNone},
			{Mods: ui.Super, Key: ui.KeyMinus, Action: ghostty.ActionNone},
			{Mods: ui.Super, Key: ui.Key0, Action: ghostty.ActionNone},
		},
		OnTitle: func(string) { notify() }, OnExit: func(int) { notify() },
	})
	if err != nil {
		return nil, err
	}
	s.nextID++
	session := &Session{ID: s.nextID, ProjectName: projectName, CWD: cwd, Terminal: term, report: report}
	s.sessions = append(s.sessions, session)
	return session, nil
}

func (s *Store) Get(id ID) *Session {
	for _, session := range s.sessions {
		if session.ID == id {
			return session
		}
	}
	return nil
}

func (s *Store) All() []*Session { return slices.Clone(s.sessions) }

func (s *Store) ForProject(projectName string) []*Session {
	var result []*Session
	for _, session := range s.sessions {
		if session.ProjectName == projectName {
			result = append(result, session)
		}
	}
	return result
}

// Exited reports whether the shell ran and has ended, whatever its exit code:
// a shell's code is only the status of its last command. A shell that could
// not start (-1) does not count, so its error stays visible.
func (s *Session) Exited() bool {
	select {
	case <-s.Terminal.Done():
		return s.Terminal.ExitCode() != -1
	default:
		return false
	}
}

func (s *Store) ReapExited() ([]ID, error) {
	var closed []ID
	var errs []error
	s.sessions = slices.DeleteFunc(s.sessions, func(session *Session) bool {
		if !session.Exited() {
			return false
		}
		if err := session.Terminal.Close(); err != nil {
			errs = append(errs, fmt.Errorf("terminal %d: %w", session.ID, err))
		}
		closed = append(closed, session.ID)
		return true
	})
	return closed, errors.Join(errs...)
}

// Rename moves the sessions of a project to its new name.
func (s *Store) Rename(old, name string) {
	for _, session := range s.sessions {
		if session.ProjectName == old {
			session.ProjectName = name
		}
	}
}

// Close ends the sessions, and whatever runs in their shells, and forgets
// them.
func (s *Store) Close(ids ...ID) error {
	var errs []error
	s.sessions = slices.DeleteFunc(s.sessions, func(session *Session) bool {
		if !slices.Contains(ids, session.ID) {
			return false
		}
		if err := session.Terminal.Close(); err != nil {
			errs = append(errs, fmt.Errorf("terminal %d: %w", session.ID, err))
		}
		return true
	})
	return errors.Join(errs...)
}

func (s *Store) CloseAll() error {
	var errs []error
	for _, session := range s.sessions {
		if err := session.Terminal.Close(); err != nil {
			errs = append(errs, fmt.Errorf("terminal %d: %w", session.ID, err))
		}
	}
	return errors.Join(errs...)
}
