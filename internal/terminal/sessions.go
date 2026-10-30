package terminal

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	ghostty "github.com/egoist/mygo/plugins/terminal"
)

type ID uint64

type Session struct {
	ID          ID
	ProjectName string
	CWD         string
	Terminal    *ghostty.Terminal
}

type Store struct {
	nextID   ID
	sessions []*Session
}

func NewStore() *Store { return &Store{} }

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
	term, err := ghostty.New(ghostty.Options{
		Dir: cwd, Theme: ghostty.DarkTheme(), Scrollback: 10 << 20,
		OnTitle: func(string) { notify() }, OnExit: func(int) { notify() },
	})
	if err != nil {
		return nil, err
	}
	s.nextID++
	session := &Session{ID: s.nextID, ProjectName: projectName, CWD: cwd, Terminal: term}
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

func (s *Store) CloseAll() error {
	var errs []error
	for _, session := range s.sessions {
		if err := session.Terminal.Close(); err != nil {
			errs = append(errs, fmt.Errorf("terminal %d: %w", session.ID, err))
		}
	}
	return errors.Join(errs...)
}
