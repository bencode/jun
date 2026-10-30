package terminal

import (
	"fmt"
	"os"
	"path/filepath"
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
	store := NewStore()
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
	store := NewStore()
	for _, cwd := range []string{"", "relative", filepath.Join(t.TempDir(), "missing")} {
		if _, err := store.Create("A", cwd, nil); err == nil {
			t.Fatalf("accepted directory %q", cwd)
		}
	}
	if len(store.ForProject("A")) != 0 {
		t.Fatal("failed creation registered a session")
	}
}

func TestReapOnlySuccessfulExits(t *testing.T) {
	shell := filepath.Join(t.TempDir(), "shell")
	if err := os.WriteFile(shell, []byte("#!/bin/sh\nexec /bin/sh -i\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SHELL", shell)
	t.Setenv("ENV", "")
	store := NewStore()
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
	sessions[0].Terminal.Send([]byte("exit 0\n"))
	sessions[1].Terminal.Send([]byte("printf '\\nFAILED_OUTPUT\\n'; exit 9\n"))
	sessions[2].Terminal.Send([]byte("/bin/sh -c 'exit 0'; printf '\\nSHELL_ALIVE\\n'\n"))
	for _, session := range sessions[:2] {
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
	closed, err := store.ReapSuccessful()
	if err != nil || len(closed) != 1 || closed[0] != sessions[0].ID || store.Get(sessions[0].ID) != nil {
		t.Fatalf("successful session not reaped: %v, %v", closed, err)
	}
	if store.Get(sessions[1].ID) == nil || sessions[1].Terminal.ExitCode() != 9 ||
		!strings.Contains(sessions[1].Terminal.Text(), "\nFAILED_OUTPUT\n") || store.Get(sessions[2].ID) == nil {
		t.Fatal("reaping discarded a failure or a running shell")
	}
	if closed, err := store.ReapSuccessful(); len(closed) != 0 || err != nil {
		t.Fatalf("repeated reaping changed sessions: %v, %v", closed, err)
	}
}
