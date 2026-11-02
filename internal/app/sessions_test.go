package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/bencode/jun/internal/history"
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
	sessions := p.Side.Views[0]
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
	if got, more := matchSessions(a.sessionLists[dir].sessions, "LARGER", 1); len(got) != 1 || got[0].ID != "c1" || more {
		t.Fatalf("searching found %+v", got)
	}
	if got, more := matchSessions(a.sessionLists[dir].sessions, "", 1); len(got) != 1 || got[0].ID != "c2" || !more {
		t.Fatalf("the list kept %+v, more %v, past its limit", got, more)
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

func TestSessionsListsTheSettingsClaudeConfigFolders(t *testing.T) {
	dir := gitRepo(t)
	def, other := t.TempDir(), t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", def)
	t.Setenv("CODEX_HOME", t.TempDir())
	now := time.Now()
	claudeSession(t, def, dir, "c1", "From the default", now.Add(-time.Hour), "Hi")
	claudeSession(t, other, dir, "c2", "From another account", now, "Hello")
	a := testNewApp(t)
	if err := a.state.AddProject("Jun", dir); err != nil {
		t.Fatal(err)
	}
	p := a.state.CurrentPerspective()
	sessions := p.Side.Views[slices.IndexFunc(p.Side.Views, func(v *layout.View) bool { return v.Kind == layout.ViewSessions })]
	if err := a.state.SelectView(p.Side.ID, sessions.ID); err != nil {
		t.Fatal(err)
	}
	tt := ui.NewTester(a.view, 1200, 700)
	tt.Frame()
	tt.Frame()
	if !tt.HasText("From the default") || tt.HasText("From another account") {
		t.Fatalf("listed %v before adding the folder", tt.Texts())
	}
	a.setPref([]string{"claude", "configDirs"}, []string{def, other})
	tt.Frame()
	tt.Frame()
	if !tt.HasText("From the default") || !tt.HasText("From another account") {
		t.Fatalf("listed %v from %v", tt.Texts(), a.prefs.Claude.ConfigDirs)
	}
}

func TestSessionDetailShowsWhatIsKnown(t *testing.T) {
	updated := time.Date(2026, 11, 1, 15, 4, 0, 0, time.Local)
	session := func(cwd, branch string, ran time.Duration) history.Session {
		s := history.Session{CWD: cwd, Branch: branch, Updated: updated}
		if ran > 0 {
			s.Start = updated.Add(-ran)
		}
		return s
	}
	for _, c := range []struct {
		s    history.Session
		want string
	}{
		{session("/work/jun", "main", 25*time.Minute), "main · 15:04 · 25m"},
		{session("/work/jun/priv/worktrees/x", "feat/x", 2*time.Hour), "priv/worktrees/x · feat/x · 15:04 · 2h"},
		{session("/work/june", "", 0), "15:04"},
		{session("/work/jun", "", 30*time.Second), "15:04 · <1m"},
		{session("/work/jun", "", 70*time.Minute), "15:04 · 1h 10m"},
		{history.Session{CWD: "/work/jun", Start: updated.Add(time.Hour), Updated: updated}, "15:04"},
	} {
		if got := sessionDetail(c.s, "/work/jun"); got != c.want {
			t.Errorf("detail of %+v is %q, want %q", c.s, got, c.want)
		}
	}
}
