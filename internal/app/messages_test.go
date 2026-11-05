package app

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bencode/jun/internal/config"
	"github.com/bencode/jun/internal/message"
	"github.com/bencode/jun/internal/terminal"
	"github.com/bencode/jun/internal/tool"
	"github.com/egoist/mygo/ui"
)

func say(t *testing.T, a *app, from *terminal.Session, about terminal.ID, text string) tool.Response {
	t.Helper()
	req := tool.Request{Op: tool.OpMessage, About: uint64(about), Text: text}
	return a.uiAnswers(req, shellOf(t, from))[tool.OpMessage]()
}

func TestAnyTerminalTellsTheUser(t *testing.T) {
	a, tt, sessions := agentsApp(t)
	a.state.GlobalVisible = true
	configPath := filepath.Join(t.TempDir(), "config.json")
	a.loadMessages(configPath)
	sender, other := sessions[0], sessions[1]
	a.lastTerminal = other.ID

	if resp := say(t, a, sender, 0, "The tests pass"); resp.Error != "" {
		t.Fatal(resp.Error)
	}
	if resp := say(t, a, sender, other.ID, "It waits for you"); resp.Error != "" {
		t.Fatal(resp.Error)
	}
	if resp := say(t, a, sender, 0, " "); resp.Error == "" {
		t.Fatal("took a blank message")
	}
	if resp := say(t, a, sender, 999, "No one"); resp.Error != "no terminal 999" {
		t.Fatalf("about no terminal: %q", resp.Error)
	}
	req := tool.Request{Op: tool.OpMessage, Text: "From outside"}
	if resp := a.uiAnswers(req, nil)[tool.OpMessage](); resp.Error != "caller is not a terminal of this Jun" {
		t.Fatalf("from outside Jun: %q", resp.Error)
	}
	tt.Frame()
	if !tt.HasText("The tests pass") || !tt.HasText("It waits for you") {
		t.Fatalf("Global shows %v", tt.Texts())
	}

	a.messages = nil
	a.loadMessages(configPath)
	if len(a.messages) != 2 {
		t.Fatalf("read back %+v", a.messages)
	}
	if m := a.messages[0]; m.Terminal != uint64(sender.ID) || m.Project != sender.ProjectName {
		t.Fatalf("from the sender: %+v", m)
	}
	if m := a.messages[1]; m.Terminal != uint64(other.ID) || m.Project != other.ProjectName {
		t.Fatalf("about the other terminal: %+v", m)
	}
}

func TestAMessageOpensOnlyWhatItIsAbout(t *testing.T) {
	a, tt, sessions := agentsApp(t)
	a.state.GlobalVisible = true
	configPath := filepath.Join(t.TempDir(), "config.json")
	var s *terminal.Session
	for _, session := range sessions {
		if !a.terminalOnScreen(session.ID) {
			s = session
		}
	}
	if s == nil {
		t.Fatal("every agent's terminal shows")
	}
	earlier := message.Message{Time: time.Now(), Terminal: uint64(s.ID), Project: "Jun", Text: "From an earlier run"}
	if err := message.Append(filepath.Join(filepath.Dir(configPath), "messages.jsonl"), earlier); err != nil {
		t.Fatal(err)
	}
	a.loadMessages(configPath)
	say(t, a, sessions[0], s.ID, "From this run")
	a.messages = append(a.messages, message.Message{Time: time.Now(), Terminal: 999, Text: "Of an ended terminal", Live: true})
	tt.Frame()
	for _, text := range []string{"From an earlier run", "Of an ended terminal"} {
		if err := tt.Click(text); err != nil {
			t.Fatal(err)
		}
		if a.peek.id != 0 {
			t.Fatalf("%q peeked terminal %d", text, a.peek.id)
		}
	}
	if err := tt.Click("From this run"); err != nil {
		t.Fatal(err)
	}
	if a.peek.id != s.ID {
		t.Fatalf("peeking terminal %d", a.peek.id)
	}

	// ⌘W closes the peek, then finds the focus in Global, which keeps its tabs.
	tabs := len(a.state.CurrentPerspective().Columns[0].Views)
	tt.Key(ui.Super, ui.KeyW)
	tt.Frame()
	tt.Key(ui.Super, ui.KeyW)
	tt.Frame()
	if a.peek.id != 0 || len(a.state.CurrentPerspective().Columns[0].Views) != tabs {
		t.Fatalf("⌘W left peek %d and closed a tab of the column", a.peek.id)
	}

	delete(a.agents, s.ID)
	tt.Frame()
	if err := tt.Click("From this run"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.peek.id != 0 || !a.terminalOnScreen(s.ID) {
		t.Fatalf("a terminal running no agent: peek %d, on screen %v", a.peek.id, a.terminalOnScreen(s.ID))
	}
}

func TestTheFileIsCutBackAsMessagesCome(t *testing.T) {
	a, _, sessions := agentsApp(t)
	configPath := filepath.Join(t.TempDir(), "config.json")
	a.loadMessages(configPath)
	callers := shellOf(t, sessions[0])
	for i := range 2*message.Keep + 1 {
		if err := a.toolMessage(tool.Request{Text: fmt.Sprint(i)}, callers); err != nil {
			t.Fatal(err)
		}
	}
	data, err := os.ReadFile(a.messagesPath)
	if err != nil || bytes.Count(data, []byte("\n")) != message.Keep {
		t.Fatalf("the file has %d lines: %v", bytes.Count(data, []byte("\n")), err)
	}
	if len(a.messages) != message.Keep || a.messages[len(a.messages)-1].Text != fmt.Sprint(2*message.Keep) {
		t.Fatalf("Jun keeps %d messages", len(a.messages))
	}
}

func TestAMessageTheFileDoesNotTakeStillShows(t *testing.T) {
	a, tt, sessions := agentsApp(t)
	a.state.GlobalVisible = true
	a.messagesPath = t.TempDir() // a directory: Append fails
	if resp := say(t, a, sessions[0], 0, "Not saved"); !strings.HasPrefix(resp.Error, "shown, but not saved") {
		t.Fatalf("the sender heard %q", resp.Error)
	}
	tt.Frame()
	if !tt.HasText("Not saved") {
		t.Fatalf("Global shows %v", tt.Texts())
	}
}

func TestACutBackThatFailsIsTriedAgain(t *testing.T) {
	a, tt, sessions := agentsApp(t)
	directory := t.TempDir()
	path := filepath.Join(directory, "messages.jsonl")
	for i := range 2 * message.Keep {
		if err := message.Append(path, message.Message{Time: time.Now(), Text: fmt.Sprint(i)}); err != nil {
			t.Fatal(err)
		}
	}
	access := func(mode os.FileMode) {
		t.Helper()
		if err := os.Chmod(directory, mode); err != nil {
			t.Fatal(err)
		}
	}
	access(0o500)
	t.Cleanup(func() { access(0o700) })
	lines := func() int {
		t.Helper()
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		return bytes.Count(data, []byte("\n"))
	}

	a.loadMessages(filepath.Join(directory, "config.json"))
	tt.Frame()
	if a.errorOpen || lines() != 2*message.Keep {
		t.Fatalf("a failed cut left %d lines, dialog open %v", lines(), a.errorOpen)
	}

	access(0o700)
	if resp := say(t, a, sessions[0], 0, "The next message"); resp.Error != "" {
		t.Fatal(resp.Error)
	}
	if lines() != message.Keep {
		t.Fatalf("the next message left %d lines", lines())
	}
}

func TestHiddenMessagesKeepComing(t *testing.T) {
	a, tt, sessions := agentsApp(t)
	a.state.GlobalVisible = true
	a.loadMessages(filepath.Join(t.TempDir(), "config.json"))
	tt.Frame()
	say(t, a, sessions[0], 0, "Before hiding")
	tt.Frame()
	menu(t, tt, a, a.state.Global, "Hide Messages")
	tt.Frame()
	if tt.HasText("Before hiding") {
		t.Fatal("hidden messages show")
	}
	if !tt.Focused(a.title(a.state.Global.Views[0])) {
		t.Fatal("the focus did not come back to Global")
	}
	path := filepath.Join(t.TempDir(), "config.json")
	if err := config.Save(path, a.state); err != nil {
		t.Fatal(err)
	}
	if s, err := config.Load(path); err != nil || !s.MessagesHidden {
		t.Fatalf("restored hidden %v: %v", s != nil && s.MessagesHidden, err)
	}

	say(t, a, sessions[0], 0, "While hidden")
	menu(t, tt, a, a.state.Global, "Show Messages")
	tt.Frame()
	if !tt.HasText("Before hiding") || !tt.HasText("While hidden") {
		t.Fatalf("Global shows %v", tt.Texts())
	}

	a.state.GlobalVisible, a.state.MessagesHidden = false, true
	a.toggleMessages()
	if !a.state.GlobalVisible || a.state.MessagesHidden {
		t.Fatal("Toggle Messages left them hidden with Global")
	}
}

func TestMessageAge(t *testing.T) {
	a := testNewApp(t)
	now := time.Date(2026, 11, 5, 12, 0, 0, 0, time.Local)
	for age, want := range map[time.Duration]string{
		59 * time.Second: "just now",
		5 * time.Minute:  "5 min ago",
		3 * time.Hour:    "3 h ago",
		48 * time.Hour:   "11-03 12:00",
	} {
		if got := a.messageAge(now, now.Add(-age)); got != want {
			t.Errorf("%v ago: %q", age, got)
		}
	}
}
