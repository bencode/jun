package terminal

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

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

func (s *Store) CloseAll() error {
	var errs []error
	for _, session := range s.sessions {
		if err := session.Terminal.Close(); err != nil {
			errs = append(errs, fmt.Errorf("terminal %d: %w", session.ID, err))
		}
	}
	return errors.Join(errs...)
}
