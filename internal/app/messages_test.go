package app

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/bencode/jun/internal/config"
	"github.com/bencode/jun/internal/layout"
	"github.com/bencode/jun/internal/terminal"
	"github.com/bencode/jun/internal/tool"
	"github.com/egoist/mygo/ui"
)

func post(t *testing.T, a *app, from *terminal.Session, req tool.Request) tool.Response {
	t.Helper()
	req.Op = tool.OpMessage
	return a.uiAnswers(req, shellOf(t, from))[tool.OpMessage]()
}

func update(t *testing.T, a *app, from *terminal.Session, req tool.Request) tool.Response {
	t.Helper()
	req.Op = tool.OpUpdateMessage
	return a.uiAnswers(req, shellOf(t, from))[tool.OpUpdateMessage]()
}

func reply(t *testing.T, a *app, from *terminal.Session, id uint64, text string) {
	t.Helper()
	if resp := update(t, a, from, tool.Request{Message: id, Text: text}); resp.Error != "" {
		t.Fatal(resp.Error)
	}
}

// messageTexts are the messages' texts, the top row first.
func messageTexts(a *app) []string {
	list := a.toolMessages()
	texts := make([]string, len(list))
	for i, m := range list {
		texts[i] = m.Text
	}
	return texts
}

func say(t *testing.T, a *app, from *terminal.Session, text string) tool.Response {
	t.Helper()
	return post(t, a, from, tool.Request{Text: text})
}

func link(t *testing.T, a *app, from *terminal.Session, text string, to *tool.Target) {
	t.Helper()
	if resp := post(t, a, from, tool.Request{Text: text, Target: to}); resp.Error != "" {
		t.Fatal(resp.Error)
	}
}

// brain adds a project with a terminal, beside testApp's.
func brain(t *testing.T, a *app) (string, *terminal.Session) {
	t.Helper()
	dir := t.TempDir()
	if err := a.state.AddProject("Brain", dir); err != nil {
		t.Fatal(err)
	}
	a.state.ActiveProject = "Jun"
	s, err := a.terminals.Create("Brain", dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	return dir, s
}

func writeFile(t *testing.T, path string) string {
	t.Helper()
	if err := os.WriteFile(path, []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestAnyTerminalTellsTheUser(t *testing.T) {
	a, tt, sessions := agentsApp(t)
	a.state.GlobalVisible = true
	req := tool.Request{Title: "It asks", Kind: tool.MessageAsk, Text: "Whether to open a PR"}
	if resp := post(t, a, sessions[0], req); resp.Error != "" {
		t.Fatal(resp.Error)
	}
	if resp := post(t, a, sessions[0], tool.Request{Title: "Tests pass"}); resp.Error != "" {
		t.Fatal(resp.Error)
	}
	if resp := say(t, a, sessions[0], " "); resp.Error == "" {
		t.Fatal("took a blank message")
	}
	outside := tool.Request{Op: tool.OpMessage, Text: "From outside"}
	if resp := a.uiAnswers(outside, nil)[tool.OpMessage](); resp.Error != "caller is not a terminal of this Jun" {
		t.Fatalf("from outside Jun: %q", resp.Error)
	}
	tt.Frame()
	for _, text := range []string{"It asks", "Whether to open a PR", "Tests pass"} {
		if !tt.HasText(text) {
			t.Fatalf("no %q in %v", text, tt.Texts())
		}
	}
	if tt.HasText(fmt.Sprintf("#%d Jun", sessions[0].ID)) {
		t.Fatal("plain news has a tag")
	}

	// Plain news leads nowhere, by mouse or key.
	revealed := a.state.ActiveProject + fmt.Sprint(a.lastTerminal)
	if err := tt.Click("Whether to open a PR"); err != nil {
		t.Fatal(err)
	}
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if a.peek.id != 0 || a.errorOpen || a.state.ActiveProject+fmt.Sprint(a.lastTerminal) != revealed {
		t.Fatalf("plain news went somewhere: peek %d, error %q", a.peek.id, a.errorMessage)
	}
}

// A loop in one project reports on all of them: what it links to tells the
// project, not where it runs.
func TestALinkTellsAMessagesProject(t *testing.T) {
	a, tt, sessions := agentsApp(t)
	a.state.GlobalVisible = true
	dir, other := brain(t, a)
	notes := writeFile(t, filepath.Join(dir, "notes.md"))
	outside := writeFile(t, filepath.Join(t.TempDir(), "elsewhere.md"))
	for _, c := range []struct {
		to      *tool.Target
		project string
		label   string
	}{
		{target(tool.TargetTerminal, other.ID, ""), "Brain", fmt.Sprintf("#%d Brain", other.ID)},
		{target(tool.TargetFile, 0, notes), "Brain", "Brain · notes.md"},
		{target(tool.TargetFile, 0, outside), "Jun", "Jun · elsewhere.md"},
		{target(tool.TargetProject, 0, "brain"), "Brain", "Brain"},
	} {
		link(t, a, sessions[0], "Report", c.to)
		m := a.messages[len(a.messages)-1]
		tt.Frame()
		if m.Project.Name != c.project || !tt.HasText(c.label) {
			t.Fatalf("%+v in %v", m, tt.Texts())
		}
	}
}

func TestAMessageFollowsItsProject(t *testing.T) {
	a, tt, sessions := agentsApp(t)
	a.state.GlobalVisible = true
	brain(t, a)
	link(t, a, sessions[0], "Go there", target(tool.TargetProject, 0, "Brain"))
	if err := a.state.RenameProject("Brain", "Mind"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !tt.HasText("Mind") {
		t.Fatalf("no new name in %v", tt.Texts())
	}
	if err := tt.Click("Go there"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.errorOpen || a.state.ActiveProject != "Mind" {
		t.Fatalf("in %s, error %q", a.state.ActiveProject, a.errorMessage)
	}
}

func TestTheLatestMessageShows(t *testing.T) {
	a, tt, sessions := agentsApp(t)
	a.state.GlobalVisible = true
	for i := range 60 {
		say(t, a, sessions[0], fmt.Sprintf("Message %d", i))
	}
	tt.Frame()
	tt.Frame()
	say(t, a, sessions[0], "The latest")
	tt.Frame()
	tt.Frame()
	if !tt.HasText("The latest") {
		t.Fatal("the list moved away from the latest message")
	}
}

func TestALinkToNothingShowsNoMessage(t *testing.T) {
	a, _, sessions := agentsApp(t)
	for _, to := range []*tool.Target{
		target(tool.TargetTerminal, 999, ""), target(tool.TargetDiff, 999, ""), target(tool.TargetDiff, sessions[1].ID, ""),
		target(tool.TargetFile, 0, filepath.Join(t.TempDir(), "gone.md")), target(tool.TargetFile, 0, t.TempDir()),
		target(tool.TargetProject, 0, "Nothing"), target(tool.TargetPerspective, 0, "Work"),
	} {
		if resp := post(t, a, sessions[0], tool.Request{Text: "Report", Target: to}); resp.Error == "" {
			t.Errorf("took %+v", to)
		}
	}
	if len(a.messages) != 0 {
		t.Fatalf("kept %+v", a.messages)
	}
}

func TestAMessageTakesTheUserToItsTerminal(t *testing.T) {
	a, tt, sessions := agentsApp(t)
	a.state.GlobalVisible = true
	var s *terminal.Session
	for _, session := range sessions {
		if !a.terminalOnScreen(session.ID) {
			s = session
		}
	}
	if s == nil {
		t.Fatal("every agent's terminal shows")
	}
	link(t, a, sessions[0], "It waits", target(tool.TargetTerminal, s.ID, ""))
	a.messages = append(a.messages, message{ID: 99, Time: time.Now(), Text: "Of an ended terminal",
		Project: a.state.CurrentProject(), Link: target(tool.TargetTerminal, 999, "")})
	tt.Frame()
	if err := tt.Click("Of an ended terminal"); err != nil {
		t.Fatal(err)
	}
	if a.peek.id != 0 || a.errorOpen {
		t.Fatalf("an ended terminal's message: peek %d, error %q", a.peek.id, a.errorMessage)
	}
	if err := tt.Click("It waits"); err != nil {
		t.Fatal(err)
	}
	if a.peek.id != s.ID {
		t.Fatalf("peeking terminal %d", a.peek.id)
	}
	tt.Frame()
	if err := tt.Click("It waits"); err != nil { // a double click, which the row's button keeps from the list
		t.Fatal(err)
	}
	tt.Frame()
	if a.peek.id != 0 {
		t.Fatalf("clicked again, it still peeks terminal %d", a.peek.id)
	}

	tt.Frame()
	if err := tt.ClickWith(ui.Super, "It waits"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.peek.id != 0 || !a.terminalOnScreen(s.ID) {
		t.Fatalf("⌘-click: peek %d, on screen %v", a.peek.id, a.terminalOnScreen(s.ID))
	}

	// ⌘W closes no tab of the column from Global.
	a.focusRegion(a.state.Global.ID)
	tt.Frame()
	tabs := len(a.state.CurrentPerspective().Columns[0].Views)
	tt.Key(ui.Super, ui.KeyW)
	tt.Frame()
	if len(a.state.CurrentPerspective().Columns[0].Views) != tabs {
		t.Fatal("⌘W closed a tab of the column")
	}

	delete(a.agents, s.ID)
	a.revealSession(sessions[0].ID)
	tt.Frame()
	if err := tt.Click("It waits"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.peek.id != 0 || !a.terminalOnScreen(s.ID) {
		t.Fatalf("a terminal running no agent: peek %d, on screen %v", a.peek.id, a.terminalOnScreen(s.ID))
	}
}

func fileTabs(a *app, path string) []*layout.View {
	var views []*layout.View
	for _, p := range a.state.Projects {
		for _, perspective := range p.Perspectives {
			for _, r := range perspective.Columns {
				for _, v := range r.Views {
					if v.File != nil && v.File.Path == filepath.Base(path) {
						views = append(views, v)
					}
				}
			}
		}
	}
	return views
}

func TestAMessageKeepsAResourceToOneTab(t *testing.T) {
	a, tt, sessions := agentsApp(t)
	a.state.GlobalVisible = true
	dir, _ := brain(t, a)
	notes := writeFile(t, filepath.Join(dir, "notes.md"))
	link(t, a, sessions[0], "The report", target(tool.TargetFile, 0, notes))
	tt.Frame()

	if err := tt.Click("The report"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if views := fileTabs(a, notes); a.state.ActiveProject != "Brain" || len(views) != 1 || views[0].ProjectName != "Brain" {
		t.Fatalf("in %s: %+v", a.state.ActiveProject, views)
	}

	a.selectProject("Jun")
	tt.Frame()
	if err := tt.Click("The report"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if views := fileTabs(a, notes); a.state.ActiveProject != "Brain" || len(views) != 1 {
		t.Fatalf("back in %s: %+v", a.state.ActiveProject, views)
	}

	if err := os.Remove(notes); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("The report"); err != nil {
		t.Fatal(err)
	}
	if !a.errorOpen {
		t.Fatal("a file gone since says nothing")
	}
}

func TestAMessageKeepsADiffToOneTab(t *testing.T) {
	a := testNewApp(t)
	if err := a.state.AddProject("Jun", gitRepo(t)); err != nil {
		t.Fatal(err)
	}
	a.state.GlobalVisible = true
	tt := ui.NewTester(a.view, 1440, 900)
	tt.Frame()
	project := a.state.CurrentProject()
	s, err := a.terminals.Create("Jun", project.CWD, nil)
	if err != nil {
		t.Fatal(err)
	}
	diff := target(tool.TargetDiff, s.ID, "")
	onUI := func(fn func()) bool { fn(); return true }
	a.awaitRepo(nil, onUI, diff, func() bool { return false })
	link(t, a, s, "Look at the changes", diff)
	tt.Frame()
	if !tt.HasText(fmt.Sprintf("#%d Jun · Diff", s.ID)) {
		t.Fatalf("no tag in %v", tt.Texts())
	}
	diffs := func() int {
		count := 0
		for _, r := range a.state.CurrentPerspective().Columns {
			for _, v := range r.Views {
				if v.Diff != nil {
					count++
				}
			}
		}
		return count
	}
	for range 2 {
		if err := tt.Click("Look at the changes"); err != nil {
			t.Fatal(err)
		}
		tt.Frame()
		if a.errorOpen || diffs() != 1 {
			t.Fatalf("%d Diffs, error %q", diffs(), a.errorMessage)
		}
	}
}

func TestKeysGoThroughMessages(t *testing.T) {
	a, tt, sessions := agentsApp(t)
	a.state.GlobalVisible = true
	hidden := sessions[slices.IndexFunc(sessions, func(s *terminal.Session) bool { return !a.terminalOnScreen(s.ID) })]
	link(t, a, sessions[0], "Older", target(tool.TargetTerminal, hidden.ID, ""))
	say(t, a, sessions[1], "Newer")
	tt.Frame()
	a.focusRegion(a.state.Global.ID)
	tt.Frame()
	tt.Key(ui.Super, ui.Key2)
	tt.Frame()
	if !tt.Focused("Messages") {
		t.Fatal("⌘2 did not reach Messages")
	}
	tt.Key(0, ui.KeyTab)
	tt.Frame()
	tt.Key(0, ui.KeyDown)
	tt.Frame()
	tt.Key(0, ui.KeyDown)
	tt.Frame()
	if a.messageSelected != 1 || a.peek.id != 0 {
		t.Fatalf("row %d, peek %d", a.messageSelected, a.peek.id)
	}
	say(t, a, sessions[2], "Newest")
	tt.Frame()
	if a.messageSelected != 2 {
		t.Fatalf("the choice left its message: row %d", a.messageSelected)
	}
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if a.peek.id != hidden.ID {
		t.Fatalf("peeking %d", a.peek.id)
	}
}

func TestALongMessageFolds(t *testing.T) {
	a, tt, sessions := agentsApp(t)
	a.state.GlobalVisible = true
	hidden := sessions[slices.IndexFunc(sessions, func(s *terminal.Session) bool { return !a.terminalOnScreen(s.ID) })]
	say(t, a, sessions[0], "Short")
	tt.Frame()
	tt.Frame()
	if tt.HasText("Show more") {
		t.Fatal("a short text folds")
	}
	link(t, a, sessions[0], strings.Repeat("A long report of what happened. ", 40), target(tool.TargetTerminal, hidden.ID, ""))
	tt.Frame()
	tt.Frame()
	if err := tt.Click("Show more"); err != nil {
		t.Fatalf("%v in %v", err, tt.Texts())
	}
	tt.Frame()
	if !tt.HasText("Show less") || a.peek.id != 0 {
		t.Fatalf("opening it peeked %d, or did not open: %v", a.peek.id, tt.Texts())
	}

	// The click left the keyboard on the list, with that row chosen.
	tt.Key(0, ui.KeyLeft)
	tt.Frame()
	if !tt.HasText("Show more") {
		t.Fatal("← did not fold it")
	}
	tt.Key(0, ui.KeyRight)
	tt.Frame()
	if !tt.HasText("Show less") {
		t.Fatal("→ did not open it")
	}
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if a.peek.id != hidden.ID {
		t.Fatalf("↩ peeked %d", a.peek.id)
	}
}

func TestFoldingKeepsTheRowInView(t *testing.T) {
	a, tt, sessions := agentsApp(t)
	a.state.GlobalVisible = true
	say(t, a, sessions[0], strings.Repeat("A long report of what happened. ", 120))
	for i := range 40 {
		say(t, a, sessions[0], fmt.Sprintf("Message %d", i))
	}
	tt.Frame()
	tt.Frame()
	long := len(a.messages) - 1 // the row of the oldest
	a.setMessageWhole(a.messages[0].ID, true)
	a.messagesList.ScrollIntoView(long)
	tt.Frame()
	tt.Frame()
	a.setMessageWhole(a.messages[0].ID, false)
	tt.Frame()
	tt.Frame()
	if first, last := a.messagesList.Visible(); long < first || long > last {
		t.Fatalf("row %d is out of %d to %d", long, first, last)
	}
}

func TestRowsStayWhileScrolledAway(t *testing.T) {
	a, tt, sessions := agentsApp(t)
	a.state.GlobalVisible = true
	for i := range 60 {
		say(t, a, sessions[0], fmt.Sprintf("Message %d", i))
	}
	tt.Frame()
	tt.Frame()
	a.messagesList.ScrollIntoView(45)
	tt.Frame()
	tt.Frame()
	before, ok := tt.Find("Message 20")
	if !ok {
		t.Fatalf("not in %v", tt.Texts())
	}
	say(t, a, sessions[0], "One more")
	tt.Frame()
	tt.Frame()
	if after, ok := tt.Find("Message 20"); !ok || after.Y != before.Y {
		t.Fatalf("it moved from %v to %v", before, after)
	}
}

func TestTheOldMessagesFileGoes(t *testing.T) {
	config := filepath.Join(t.TempDir(), "config.json")
	old := writeFile(t, filepath.Join(filepath.Dir(config), "messages.jsonl"))
	removeOldMessages(config)
	if _, err := os.Stat(old); !os.IsNotExist(err) {
		t.Fatalf("still there: %v", err)
	}
	removeOldMessages(config) // nothing to remove is fine
}

func TestHiddenMessagesKeepComing(t *testing.T) {
	a, tt, sessions := agentsApp(t)
	a.state.GlobalVisible = true
	tt.Frame()
	say(t, a, sessions[0], "Before hiding")
	tt.Frame()
	menu(t, tt, a, a.state.Global, "Messages")
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
	if s, err := config.Load(path); err != nil || !s.Global.Views[1].Hidden {
		t.Fatalf("restored %v: %v", s, err)
	}

	say(t, a, sessions[0], "While hidden")
	menu(t, tt, a, a.state.Global, "Messages")
	tt.Frame()
	if !tt.HasText("Before hiding") || !tt.HasText("While hidden") {
		t.Fatalf("Global shows %v", tt.Texts())
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

func TestAReplyBringsItsMessageUp(t *testing.T) {
	a, tt, sessions := agentsApp(t)
	a.state.GlobalVisible = true
	asked := say(t, a, sessions[0], "Oldest").Created
	say(t, a, sessions[0], "Middle")
	say(t, a, sessions[0], "Newest")
	tt.Frame()
	a.focusRegion(a.state.Global.ID)
	tt.Frame()
	tt.Key(ui.Super, ui.Key2)
	tt.Frame()
	tt.Key(0, ui.KeyTab)
	tt.Frame()
	for range 3 {
		tt.Key(0, ui.KeyDown)
		tt.Frame()
	}
	if a.messageSelected != 2 {
		t.Fatalf("row %d", a.messageSelected)
	}
	reply(t, a, sessions[1], asked, "You answered")
	tt.Frame()
	if got := messageTexts(a); !slices.Equal(got, []string{"Oldest", "Newest", "Middle"}) {
		t.Fatalf("rows %v", got)
	}
	if !tt.HasText("You answered") {
		t.Fatalf("no reply in %v", tt.Texts())
	}
	if a.messageSelected != 0 {
		t.Fatalf("the choice left its message: row %d", a.messageSelected)
	}
}

func TestANewKindLeavesAMessageInPlace(t *testing.T) {
	a, tt, sessions := agentsApp(t)
	a.state.GlobalVisible = true
	asked := post(t, a, sessions[0], tool.Request{Text: "Oldest", Kind: tool.MessageAsk}).Created
	say(t, a, sessions[0], "Newest")
	if resp := update(t, a, sessions[0], tool.Request{Message: asked, Kind: tool.MessageDone}); resp.Error != "" {
		t.Fatal(resp.Error)
	}
	tt.Frame()
	list := a.toolMessages()
	if !slices.Equal(messageTexts(a), []string{"Newest", "Oldest"}) || list[1].Kind != tool.MessageDone || len(list[1].Replies) != 0 {
		t.Fatalf("messages %+v", list)
	}
}

func TestARefusedUpdateChangesNothing(t *testing.T) {
	a, _, sessions := agentsApp(t)
	asked := post(t, a, sessions[0], tool.Request{Text: "It asks", Kind: tool.MessageAsk}).Created
	say(t, a, sessions[0], "Newest")
	if resp := update(t, a, sessions[0], tool.Request{Message: asked + 10, Kind: tool.MessageDone, Text: "x"}); resp.Error != fmt.Sprintf("no message %d", asked+10) {
		t.Fatalf("no such message: %q", resp.Error)
	}
	if resp := update(t, a, sessions[0], tool.Request{Message: asked, Kind: "urgent", Text: "x"}); resp.Error == "" {
		t.Fatal("took an unknown kind")
	}
	if resp := update(t, a, sessions[0], tool.Request{Message: asked, Text: " "}); resp.Error == "" {
		t.Fatal("took a blank reply")
	}
	outside := tool.Request{Op: tool.OpUpdateMessage, Message: asked, Kind: tool.MessageDone}
	if resp := a.uiAnswers(outside, nil)[tool.OpUpdateMessage](); resp.Error != "caller is not a terminal of this Jun" {
		t.Fatalf("from outside Jun: %q", resp.Error)
	}
	list := a.toolMessages()
	if !slices.Equal(messageTexts(a), []string{"Newest", "It asks"}) || list[1].Kind != tool.MessageAsk || len(list[1].Replies) != 0 {
		t.Fatalf("messages %+v", list)
	}
}

func TestAMessageKeepsItsLatestReplies(t *testing.T) {
	a, _, sessions := agentsApp(t)
	id := say(t, a, sessions[0], "Busy").Created
	for i := range tool.MaxReplies + 2 {
		reply(t, a, sessions[0], id, fmt.Sprintf("Reply %d", i))
	}
	replies := a.toolMessages()[0].Replies
	if len(replies) != tool.MaxReplies || replies[0].Text != "Reply 2" {
		t.Fatalf("%d replies from %q", len(replies), replies[0].Text)
	}
}

func TestMessagesListAsTheyShow(t *testing.T) {
	a, _, sessions := agentsApp(t)
	first := post(t, a, sessions[0], tool.Request{Title: "It asks", Kind: tool.MessageAsk, Target: target(tool.TargetTerminal, sessions[1].ID, "")}).Created
	say(t, a, sessions[0], "Plain news")
	reply(t, a, sessions[0], first, "You answered")
	resp := a.uiAnswers(tool.Request{Op: tool.OpMessages}, nil)[tool.OpMessages]()
	if len(resp.Messages) != 2 {
		t.Fatalf("messages %+v", resp.Messages)
	}
	top, news := resp.Messages[0], resp.Messages[1]
	if top.ID != first || top.Kind != tool.MessageAsk || top.Project != "Jun" || top.Replies[0].Text != "You answered" {
		t.Fatalf("top %+v", top)
	}
	if news.Link != "" || news.Project != "" {
		t.Fatalf("plain news %+v", news)
	}
	if want := fmt.Sprintf("terminal:%d", sessions[1].ID); top.Link != want {
		t.Fatalf("link %q, not %q", top.Link, want)
	}
}

func TestRepliesFold(t *testing.T) {
	a, tt, sessions := agentsApp(t)
	a.state.GlobalVisible = true
	id := say(t, a, sessions[0], "Short").Created
	for i := range repliesShown {
		reply(t, a, sessions[0], id, fmt.Sprintf("Reply %d", i))
	}
	tt.Frame()
	tt.Frame()
	if tt.HasText("Show more") || !tt.HasText("Reply 0") {
		t.Fatalf("replies that fit: %v", tt.Texts())
	}
	reply(t, a, sessions[0], id, "Reply 3")
	tt.Frame()
	tt.Frame()
	if tt.HasText("Reply 0") || !tt.HasText("Reply 3") || !tt.HasText("1 earlier") {
		t.Fatalf("folded: %v", tt.Texts())
	}
	if err := tt.Click("Show more"); err != nil {
		t.Fatalf("%v in %v", err, tt.Texts())
	}
	tt.Frame()
	if !tt.HasText("Reply 0") || tt.HasText("1 earlier") {
		t.Fatalf("opened: %v", tt.Texts())
	}
}

func TestMessagesFoldAtTheTextSize(t *testing.T) {
	a, tt, sessions := agentsApp(t)
	a.state.GlobalVisible = true
	tt.Frame()
	tt.Frame()
	// A title keeps the text's width, whatever the size of the row's icon.
	post(t, a, sessions[0], tool.Request{Title: "Report", Text: strings.Repeat("word ", int(a.messagesWidth/12))})
	tt.Frame()
	tt.Frame()
	if tt.HasText("Show more") {
		t.Fatal("it folds at the default size")
	}
	a.prefs.Font.Size = 40
	tt.Frame()
	tt.Frame()
	if !tt.HasText("Show more") {
		t.Fatalf("it does not fold at 40: %v", tt.Texts())
	}
}
