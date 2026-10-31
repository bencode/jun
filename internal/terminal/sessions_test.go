package terminal

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

func waitFor(t *testing.T, ready func() bool) {
	t.Helper()
	deadline := time.Now().Add(8 * time.Second)
	for !ready() {
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for terminal")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestSessionsRunWithoutViewsAndKeepIndependentDirectories(t *testing.T) {
	shell := filepath.Join(t.TempDir(), "shell")
	if err := os.WriteFile(shell, []byte("#!/bin/sh\nexec /bin/sh -i\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SHELL", shell)
	t.Setenv("ENV", "")
	store := NewStore(t.TempDir())
	t.Cleanup(func() {
		if err := store.CloseAll(); err != nil {
			t.Error(err)
		}
	})
	first, err := store.Create("A", t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.Create("B", t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, session := range []*Session{first, second} {
		t.Cleanup(func() {
			if t.Failed() {
				t.Logf("session %d: %s", session.ID, session.Terminal.Text())
			}
		})
		session.Terminal.Send([]byte(fmt.Sprintf("test . -ef %q && printf '\\nCWD_OK\\n'; printf '\\nBACKGROUND_READY\\n'\n", session.CWD)))
		waitFor(t, func() bool {
			return strings.Contains(session.Terminal.Text(), "\nCWD_OK\n") && strings.Contains(session.Terminal.Text(), "\nBACKGROUND_READY\n")
		})
		if !strings.Contains(session.Terminal.Text(), "\nBACKGROUND_READY\n") {
			t.Fatal("background output missing")
		}
	}
	if first.ID == second.ID || store.Get(first.ID) != first || len(store.ForProject("A")) != 1 {
		t.Fatal("session identity or ownership lost")
	}
	first.Terminal.Send([]byte("exit 7\n"))
	waitFor(t, func() bool {
		select {
		case <-first.Terminal.Done():
			return true
		default:
			return false
		}
	})
	if first.Terminal.ExitCode() != 7 || !strings.Contains(first.Terminal.Text(), "BACKGROUND_READY") {
		t.Fatal("exit discarded output or status")
	}
	if err := store.CloseAll(); err != nil {
		t.Fatal(err)
	}
	waitFor(t, func() bool {
		select {
		case <-second.Terminal.Done():
			return true
		default:
			return false
		}
	})
}

func TestInvalidDirectoryDoesNotCreateSession(t *testing.T) {
	store := NewStore(t.TempDir())
	for _, cwd := range []string{"", "relative", filepath.Join(t.TempDir(), "missing")} {
		if _, err := store.Create("A", cwd, nil); err == nil {
			t.Fatalf("accepted directory %q", cwd)
		}
	}
	if len(store.ForProject("A")) != 0 {
		t.Fatal("failed creation registered a session")
	}
}

func TestReapExitedShellsRegardlessOfCode(t *testing.T) {
	shell := filepath.Join(t.TempDir(), "shell")
	if err := os.WriteFile(shell, []byte("#!/bin/sh\nexec /bin/sh -i\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SHELL", shell)
	t.Setenv("ENV", "")
	store := NewStore(t.TempDir())
	t.Cleanup(func() {
		if err := store.CloseAll(); err != nil {
			t.Error(err)
		}
	})
	var sessions []*Session
	for range 3 {
		session, err := store.Create("Jun", t.TempDir(), nil)
		if err != nil {
			t.Fatal(err)
		}
		sessions = append(sessions, session)
	}
	t.Setenv("SHELL", filepath.Join(t.TempDir(), "missing"))
	broken, err := store.Create("Jun", t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	sessions[0].Terminal.Send([]byte("exit 0\n"))
	sessions[1].Terminal.Send([]byte("false; exit\n"))
	sessions[2].Terminal.Send([]byte("/bin/sh -c 'exit 0'; printf '\\nSHELL_ALIVE\\n'\n"))
	for _, session := range []*Session{sessions[0], sessions[1], broken} {
		waitFor(t, func() bool {
			select {
			case <-session.Terminal.Done():
				return true
			default:
				return false
			}
		})
	}
	waitFor(t, func() bool { return strings.Contains(sessions[2].Terminal.Text(), "\nSHELL_ALIVE\n") })
	closed, err := store.ReapExited()
	if err != nil || !slices.Equal(closed, []ID{sessions[0].ID, sessions[1].ID}) {
		t.Fatalf("exited sessions not reaped: %v, %v", closed, err)
	}
	if store.Get(sessions[2].ID) == nil || store.Get(broken.ID) == nil || broken.Terminal.ExitCode() != -1 {
		t.Fatal("reaping discarded a running shell or a start failure")
	}
	if closed, err := store.ReapExited(); len(closed) != 0 || err != nil {
		t.Fatalf("repeated reaping changed sessions: %v, %v", closed, err)
	}
}

func TestShellReportsItsProcess(t *testing.T) {
	shell := filepath.Join(t.TempDir(), "shell")
	if err := os.WriteFile(shell, []byte("#!/bin/sh\nexec /bin/sh -i\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SHELL", shell)
	t.Setenv("ENV", "")
	store := NewStore(t.TempDir())
	t.Cleanup(func() {
		if err := store.CloseAll(); err != nil {
			t.Error(err)
		}
	})
	session, err := store.Create("Jun", t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	session.Terminal.Send([]byte("printf '\\nPID=%s ARGV0=%s\\n' $$ \"$0\"\n"))
	waitFor(t, func() bool { return strings.Contains(session.Terminal.Text(), "\nPID=") })
	pid, ok := session.ShellPID()
	if !ok || !strings.Contains(session.Terminal.Text(), fmt.Sprintf("\nPID=%d ARGV0=", pid)) {
		t.Fatalf("reported %d (%v), shell printed: %s", pid, ok, session.Terminal.Text())
	}
}
