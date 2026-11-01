package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bencode/jun/internal/layout"
	"github.com/egoist/mygo/ui"
)

// claudeSession writes a Claude Code session of dir, asking prompts.
func claudeSession(t *testing.T, config, dir, id, title string, mod time.Time, prompts ...string) {
	t.Helper()
	var b strings.Builder
	for _, p := range prompts {
		line, err := json.Marshal(map[string]any{"type": "user", "cwd": dir, "gitBranch": "main", "message": map[string]any{"role": "user", "content": p}})
		if err != nil {
			t.Fatal(err)
		}
		b.Write(append(line, '\n'))
	}
	b.WriteString(`{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","name":"Bash"},{"type":"text","text":"Done, see **zoom**."}]}}` + "\n")
	b.WriteString(`{"type":"ai-title","aiTitle":"` + title + `"}` + "\n")
	folder := filepath.Join(config, "projects", strings.Map(func(r rune) rune {
		if r < 128 && (r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9') {
			return r
		}
		return '-'
	}, dir))
	path := filepath.Join(folder, id+".jsonl")
	if err := os.MkdirAll(folder, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, mod, mod); err != nil {
		t.Fatal(err)
	}
}

func TestSessionsListSearchShowAndResume(t *testing.T) {
	dir := gitRepo(t)
	config := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", config)
	t.Setenv("CODEX_HOME", t.TempDir())
	now := time.Now()
	claudeSession(t, config, dir, "c1", "Zoom the text", now.Add(-time.Hour), "Make text larger")
	claudeSession(t, config, dir, "c2", "Sessions plan", now, "Plan the sessions list")
	claudeSession(t, config, t.TempDir(), "c3", "Elsewhere", now, "Not this project")
	a := testNewApp(t)
	if err := a.state.AddProject("Jun", dir); err != nil {
		t.Fatal(err)
	}
	p := a.state.CurrentPerspective()
	sessions := p.Side.Views[1]
	if err := a.state.SelectView(p.Side.ID, sessions.ID); err != nil {
		t.Fatal(err)
	}
	tt := ui.NewTester(a.view, 1200, 700)
	tt.Frame()
	tt.Frame()
	if !tt.HasText("Sessions plan") || !tt.HasText("Zoom the text") || tt.HasText("Elsewhere") || !tt.HasText("Today") {
		t.Fatalf("listed %v", tt.Texts())
	}

	column := p.Columns[0]
	tt.Click("Sessions plan")
	tt.Frame()
	shown := activeView(column)
	if shown == nil || shown.Transcript == nil || shown.Transcript.ID != "c2" || a.previewTabs[column.ID] != shown.ID {
		t.Fatalf("a click showed %+v", shown)
	}
	if !tt.HasText("Plan the sessions list") || !tt.HasText("· 1 tool step") || !tt.HasText("c2") {
		t.Fatalf("the transcript shows %v", tt.Texts())
	}

	// The preview tab moves on to an older session found by a search, and
	// shows it.
	a.sessionViews[sessions.ID].query = "LARGER"
	tt.Frame()
	if got := matchSessions(a.sessionLists[dir].sessions, "LARGER"); len(got) != 1 || got[0].ID != "c1" {
		t.Fatalf("searching found %+v", got)
	}
	tt.Click("Zoom the text")
	tt.Frame()
	if moved := activeView(column); moved.ID != shown.ID || moved.Transcript.ID != "c1" || !tt.HasText("Make text larger") || tt.HasText("Plan the sessions list") {
		t.Fatalf("the preview tab shows %+v: %v", moved.Transcript, tt.Texts())
	}

	tt.Click("Resume")
	tt.Frame()
	terminal := activeView(column)
	if terminal.Kind != layout.ViewTerminal || a.sessionFor(terminal.ID) == nil || a.sessionFor(terminal.ID).CWD != dir {
		t.Fatalf("Resume opened %+v", terminal)
	}
}
