package app

import (
	"errors"
	"strings"
	"testing"

	"github.com/bencode/jun/internal/terminal"
	"github.com/egoist/mygo/ui"
)

func TestQuitDetailAndEnterCancels(t *testing.T) {
	a, tt := testApp(t)
	sessions := a.terminals.All()
	if len(sessions) < 2 {
		t.Fatalf("test app has %d terminals", len(sessions))
	}
	busy := map[terminal.ID]string{sessions[1].ID: "sleep", sessions[0].ID: "codex"}
	want := strings.Join([]string{
		"Programs still run in 2 terminals; quitting ends them:",
		"codex — Jun · " + a.sessionTitle(sessions[0]),
		"sleep — Jun · " + a.sessionTitle(sessions[1]),
	}, "\n")
	if got := a.quitDetail(busy, nil); got != want {
		t.Fatalf("detail:\n%s\nwant:\n%s", got, want)
	}
	if got := a.quitDetail(nil, errors.New("ps failed")); !strings.Contains(got, "ps failed") {
		t.Fatalf("error detail %q", got)
	}
	quit := false
	a.confirm("Quit Jun?", a.quitDetail(busy, nil), "Quit", func() { quit = true })
	tt.Frame()
	tt.Key(0, ui.KeyEnter)
	if quit || a.confirmation.open {
		t.Fatalf("Enter quit %v, dialog open %v", quit, a.confirmation.open)
	}
}
