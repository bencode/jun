package app

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/bencode/jun/internal/layout"
	"github.com/bencode/jun/internal/review"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

// reviewRepo is a project on a repository with main.go and notes.md
// committed, shown at the size the tests click at.
func reviewRepo(t *testing.T) (*app, *ui.Tester, string) {
	t.Helper()
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir, "-c", "user.name=t", "-c", "user.email=t@example.com"}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	git("init", "-q", "-b", "main")
	writeText(t, filepath.Join(dir, "main.go"), "package main\n\nfunc main() {\n\tprintln(1)\n}\n")
	writeText(t, filepath.Join(dir, "notes.md"), "# Title\n\nA **bold** word here.\n\nSecond paragraph.\n")
	git("add", ".")
	git("commit", "-qm", "first")
	a := testNewApp(t)
	a.reviewRoot = t.TempDir()
	if err := a.state.AddProject("Web", dir); err != nil {
		t.Fatal(err)
	}
	tt := ui.NewTester(a.view, 1200, 700)
	frameUntil(t, tt, func() bool { return a.repoFor(dir).sources != nil })
	return a, tt, dir
}

func writeText(t *testing.T, path, text string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
}

func frameUntil(t *testing.T, tt *ui.Tester, done func() bool) {
	t.Helper()
	for range 200 {
		tt.Frame()
		if done() {
			tt.Frame()
			return
		}
	}
	t.Fatal("it did not settle")
}

func openTestFile(t *testing.T, a *app, tt *ui.Tester, name string) *layout.View {
	t.Helper()
	a.openFile("Web", fileNode{source: "main", path: name}, true)
	v := activeView(a.tabsTarget())
	frameUntil(t, tt, func() bool { st := a.fileViews[v.ID]; return st != nil && st.read && !st.loading })
	return v
}

func saveTestNote(t *testing.T, a *app, tt *ui.Tester, text string) {
	t.Helper()
	v := activeView(a.tabsTarget())
	frameUntil(t, tt, func() bool { d := a.drafts[v.ID]; return d != nil && d.checked })
	a.drafts[v.ID].text = text
	a.saveNote(a.drafts[v.ID])
	tt.Frame()
	if a.drafts[v.ID] != nil {
		t.Fatal("the note was not saved")
	}
}

func TestNotesBecomeAReviewAnAgentCanRead(t *testing.T) {
	a, tt, dir := reviewRepo(t)
	v := openTestFile(t, a, tt, "main.go")
	source := a.fileViews[v.ID].renders[a.fileViews[v.ID].shown].(*sourceRender)
	buffer := source.editor.Buffer
	source.editor.Select(buffer.Offset(editorPos(2, 0)), buffer.Offset(editorPos(3, 3)))
	a.noteHere()
	saveTestNote(t, a, tt, "  simplify this  ")

	l := a.reviewLists[dir]
	if len(l.items) != 1 {
		t.Fatalf("reviews: %+v", l.items)
	}
	r := l.items[0]
	n := r.Notes[0]
	if r.Source != "main" || r.Head == "" || r.Uncommitted() || n.Text != "simplify this" ||
		*n.Anchor.Lines != (review.Lines{First: 3, Last: 4}) || n.Anchor.Quote != "func main() {\n\tprintln(1)" || n.Anchor.Render != "source" {
		t.Fatalf("review: %+v", r)
	}

	// The file an agent is told of holds the note.
	data, err := os.ReadFile(review.Path(a.reviewRoot, r))
	if err != nil {
		t.Fatal(err)
	}
	var read review.Review
	if err := json.Unmarshal(data, &read); err != nil || read.Notes[0].Text != "simplify this" {
		t.Fatalf("the file: %v, %s", err, data)
	}

	// Reviews starts hidden, and shows the row once asked to.
	if tt.HasText(a.reviewDetail(r)) {
		t.Fatal("Reviews shows before it is asked to")
	}
	a.showReviews()
	tt.Frame()
	tt.Frame()
	if !tt.HasText(a.reviewDetail(r)) {
		t.Fatalf("Reviews shows %v", tt.Texts())
	}

	// Done, the review is read-only, and the next note starts another.
	a.openReview("Web", r.ID, true)
	tt.Frame()
	tt.Frame()
	if !tt.HasText("simplify this") {
		t.Fatalf("the review shows %v", tt.Texts())
	}
	tt.Click("Done")
	tt.Frame()
	if !l.items[0].Done() || tt.HasText("Edit") {
		t.Fatalf("done: %+v; %v", l.items[0], tt.Texts())
	}
	a.openFile("Web", fileNode{source: "main", path: "main.go"}, true)
	tt.Frame()
	a.noteHere()
	saveTestNote(t, a, tt, "another")
	if len(l.items) != 2 || l.items[0].Done() || len(l.items[0].Notes) != 1 {
		t.Fatalf("after done: %+v", l.items)
	}
}

func TestANoteOnUncommittedTextIsMarked(t *testing.T) {
	a, tt, dir := reviewRepo(t)
	writeText(t, filepath.Join(dir, "main.go"), "package main\n\nfunc main() {\n\tprintln(2)\n}\n")
	v := openTestFile(t, a, tt, "main.go")
	if text := a.fileViews[v.ID].text; !strings.Contains(text, "println(2)") {
		t.Fatalf("the file tab reads %q", text)
	}
	a.noteHere()
	saveTestNote(t, a, tt, "from an agent")
	if r := a.reviewLists[dir].items[0]; !r.Uncommitted() || r.Head == "" {
		t.Fatalf("review: %+v", r)
	}
}

func TestJumpFindsANoteThatMovedAndTellsOfOneGone(t *testing.T) {
	a, tt, dir := reviewRepo(t)
	v := openTestFile(t, a, tt, "main.go")
	source := a.fileViews[v.ID].renders[a.fileViews[v.ID].shown].(*sourceRender)
	buffer := source.editor.Buffer
	source.editor.Select(buffer.Offset(editorPos(3, 0)), buffer.Offset(editorPos(3, 3)))
	a.noteHere()
	saveTestNote(t, a, tt, "look")
	project := a.project("Web")
	r := a.reviewLists[dir].items[0]

	// Lines were added above the note.
	source.update([]byte("package main\n\n// added\n// added\n\nfunc main() {\n\tprintln(1)\n}\n"))
	a.fileViews[v.ID].text = source.editor.Buffer.String()
	a.jumpNote(project, r, r.Notes[0])
	frameUntil(t, tt, func() bool { return len(a.jumps) == 0 })
	if got, want := a.reviewNotices[noteKey(r.ID, 1)], "Moved to line 7."; got != want {
		t.Fatalf("notice %q, want %q", got, want)
	}
	if line := source.editor.CursorPos().Line; line != 6 {
		t.Fatalf("the caret is on line %d", line)
	}

	source.update([]byte("package main\n"))
	a.fileViews[v.ID].text = source.editor.Buffer.String()
	a.jumpNote(project, r, r.Notes[0])
	frameUntil(t, tt, func() bool { return len(a.jumps) == 0 })
	if got, want := a.reviewNotices[noteKey(r.ID, 1)], "The text has changed."; got != want {
		t.Fatalf("notice %q, want %q", got, want)
	}
}

func TestDiffNoteReopensItsComparison(t *testing.T) {
	a, tt, dir := reviewRepo(t)
	cmd := exec.Command("git", "-C", dir, "checkout", "-qb", "feat")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	writeText(t, filepath.Join(dir, "main.go"), "package main\n\nfunc main() {\n\tprintln(9)\n}\n")
	a.refreshRepo(a.repoFor(dir), dir)
	frameUntil(t, tt, func() bool { _, ok := a.repoFor(dir).source("feat"); return ok })
	a.openDiff("Web", "feat")
	v := activeView(a.tabsTarget())
	frameUntil(t, tt, func() bool {
		st := a.diffs[v.ID]
		return st != nil && st.done && !st.file.loading && st.file.state != nil
	})
	st := a.diffs[v.ID]
	if st.base != "main" {
		t.Fatalf("base %q", st.base)
	}
	lines := diffLines(3)
	a.noteInDiff(v, &lines)
	saveTestNote(t, a, tt, "why 9")
	r := a.reviewLists[dir].items[0]
	n := r.Notes[0]
	if n.Anchor.Diff == nil || n.Anchor.Diff.Base != "main" || n.Anchor.Quote != "\tprintln(9)" || n.Anchor.Side != review.SideNew || !n.Anchor.Uncommitted {
		t.Fatalf("note: %+v", n)
	}

	// The tab moved on to other parameters; the note brings them back.
	v.Diff.Uncommitted = true
	v.Diff.Base = ""
	tt.Frame()
	a.jumpNote(a.project("Web"), r, n)
	frameUntil(t, tt, func() bool { return len(a.jumps) == 0 })
	shown := activeView(a.tabsTarget())
	if shown.Diff.Base != "main" || shown.Diff.Uncommitted {
		t.Fatalf("the Diff compares %+v", shown.Diff)
	}
	if got, ok := a.diffs[shown.ID].file.state.SelectedLines(); !ok || got.Start != 3 {
		t.Fatalf("selected %+v, %v", got, ok)
	}
}

func TestEditingANoteOfAReviewGoneOrDoneChangesNothing(t *testing.T) {
	a, tt, dir := reviewRepo(t)
	v := openTestFile(t, a, tt, "main.go")
	a.noteHere()
	saveTestNote(t, a, tt, "first")
	l := a.reviewLists[dir]
	r := l.items[0]
	a.editNote(v, r, r.Notes[0])
	a.drafts[v.ID].text = "edited"
	a.finishReview(r)
	a.saveNote(a.drafts[v.ID])
	if got := l.items[0]; got.Notes[0].Text != "first" || a.drafts[v.ID] != nil {
		t.Fatalf("a done review changed: %+v", got)
	}
	a.editNote(v, l.items[0], l.items[0].Notes[0])
	a.drafts[v.ID].text = "edited"
	a.deleteReview(l.items[0])
	a.saveNote(a.drafts[v.ID])
	if len(l.items) != 0 || a.drafts[v.ID] != nil {
		t.Fatalf("a deleted review came back: %+v", l.items)
	}
}

func TestJumpOpensTheFileOnceTheComparisonShowsNoChanges(t *testing.T) {
	a, tt, dir := reviewRepo(t)
	writeText(t, filepath.Join(dir, "main.go"), "package main\n\nfunc main() {\n\tprintln(9)\n}\n")
	a.refreshRepo(a.repoFor(dir), dir)
	frameUntil(t, tt, func() bool { return a.repoFor(dir).sources != nil })
	if _, err := a.state.OpenDiff(a.tabsTarget().ID, "Web", layout.DiffRef{Source: "main", Uncommitted: true}); err != nil {
		t.Fatal(err)
	}
	v := activeView(a.tabsTarget())
	frameUntil(t, tt, func() bool {
		st := a.diffs[v.ID]
		return st != nil && st.done && !st.file.loading && st.file.state != nil
	})
	lines := diffLines(3)
	a.noteInDiff(v, &lines)
	saveTestNote(t, a, tt, "why 9")
	r := a.reviewLists[dir].items[0]

	// The agent committed it: nothing is uncommitted any more.
	for _, args := range [][]string{{"add", "."}, {"-c", "user.name=t", "-c", "user.email=t@example.com", "commit", "-qm", "second"}} {
		if out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("%v\n%s", err, out)
		}
	}
	a.diffs[v.ID].version = -1
	frameUntil(t, tt, func() bool { st := a.diffs[v.ID]; return !st.loading && len(st.rows) == 0 })
	a.jumpNote(a.project("Web"), r, r.Notes[0])
	frameUntil(t, tt, func() bool { return len(a.jumps) == 0 })
	shown := activeView(a.tabsTarget())
	if shown.File == nil || shown.File.Path != "main.go" {
		t.Fatalf("the jump shows %+v", shown)
	}
	if notice := a.reviewNotices[noteKey(r.ID, 1)]; notice != "" {
		t.Fatalf("notice %q", notice)
	}
}

func TestTheKeysWriteANoteAndADraftStaysWithItsView(t *testing.T) {
	a, tt, dir := reviewRepo(t)
	v := openTestFile(t, a, tt, "main.go")
	tt.Key(noteKeys.mods, noteKeys.key)
	tt.Frame()
	tt.Frame()
	tt.Type("clearer")
	tt.Frame()
	tt.Key(ui.Super, ui.KeyEnter)
	tt.Frame()
	if a.drafts[v.ID] != nil || a.reviewLists[dir].items[0].Notes[0].Text != "clearer" {
		t.Fatalf("⌘↩: %+v", a.drafts[v.ID])
	}

	tt.Key(noteKeys.mods, noteKeys.key)
	tt.Frame()
	tt.Frame()
	tt.Type("half")
	tt.Frame()
	tt.Key(noteKeys.mods, noteKeys.key)
	tt.Frame()
	if d := a.drafts[v.ID]; d == nil || d.text != "half" {
		t.Fatalf("a second start took the draft: %+v", d)
	}

	// Another view's note leaves it as it is.
	r := a.reviewLists[dir].items[0]
	a.openReview("Web", r.ID, true)
	tt.Frame()
	tt.Frame()
	tt.Click("Edit")
	tt.Frame()
	if d := a.drafts[activeView(a.tabsTarget()).ID]; d == nil || d.text != "clearer" {
		t.Fatalf("the edit shows %+v", d)
	}
	if d := a.drafts[v.ID]; d == nil || d.text != "half" {
		t.Fatalf("editing took the draft: %+v", d)
	}

	// Back on its tab, the keys that start a note lead to the draft.
	a.state.SelectView(a.tabsTarget().ID, v.ID)
	tt.Frame()
	tt.Frame()
	tt.Key(noteKeys.mods, noteKeys.key)
	tt.Frame()
	tt.Frame()
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if a.drafts[v.ID] != nil {
		t.Fatal("Esc left the input open")
	}
}

func TestMarkdownTakesNotesInItsSource(t *testing.T) {
	a, tt, dir := reviewRepo(t)
	v := openTestFile(t, a, tt, "notes.md")
	st := a.fileViews[v.ID]
	if _, reading := st.renders[st.shown].(*markdownRender); !reading {
		t.Fatalf("notes.md shows as %T", st.renders[st.shown])
	}
	tt.Frame()
	tt.Key(noteKeys.mods, noteKeys.key)
	tt.Frame()
	if _, found := tt.Find("Add Note"); found || a.drafts[v.ID] != nil {
		t.Fatalf("Reading takes a note: %v", a.drafts[v.ID])
	}

	st.shown = slices.IndexFunc(st.kinds, func(k int) bool { return renderKinds[k].name == sourceKind.name })
	tt.Frame()
	tt.Frame()
	source := st.renders[st.shown].(*sourceRender)
	buffer := source.editor.Buffer
	source.editor.Select(buffer.Offset(editorPos(2, 0)), buffer.Offset(editorPos(2, 4)))
	a.noteHere()
	saveTestNote(t, a, tt, "clearer")
	r := a.reviewLists[dir].items[0]
	if n := r.Notes[0].Anchor; n.Render != "source" || *n.Lines != (review.Lines{First: 3, Last: 3}) {
		t.Fatalf("anchor: %+v", n)
	}

	// Back in Reading, a jump goes to the source to show its place.
	st.shown = 0
	tt.Frame()
	a.jumpNote(a.project("Web"), r, r.Notes[0])
	frameUntil(t, tt, func() bool { return len(a.jumps) == 0 })
	if _, inSource := st.renders[st.shown].(*sourceRender); !inSource || source.editor.CursorPos().Line != 2 {
		t.Fatalf("the jump shows %T, line %d", st.renders[st.shown], source.editor.CursorPos().Line)
	}
}

func TestAJumpToAFileADiffCannotShowEnds(t *testing.T) {
	a, tt, dir := reviewRepo(t)
	writeText(t, filepath.Join(dir, "main.go"), "package main\n\nfunc main() {\n\tprintln(9)\n}\n")
	writeText(t, filepath.Join(dir, "other.txt"), "x\n")
	a.refreshRepo(a.repoFor(dir), dir)
	frameUntil(t, tt, func() bool { return a.repoFor(dir).sources != nil })
	if _, err := a.state.OpenDiff(a.tabsTarget().ID, "Web", layout.DiffRef{Source: "main", Uncommitted: true}); err != nil {
		t.Fatal(err)
	}
	v := activeView(a.tabsTarget())
	frameUntil(t, tt, func() bool {
		st := a.diffs[v.ID]
		return st != nil && st.done && !st.file.loading && st.file.state != nil
	})
	st := a.diffs[v.ID]
	lines := diffLines(3)
	a.noteInDiff(v, &lines)
	saveTestNote(t, a, tt, "why 9")
	r := a.reviewLists[dir].items[0]

	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte{0, 1, 2, 0, 3}, 0600); err != nil {
		t.Fatal(err)
	}
	st.version = -1
	st.file = diffFile{}
	frameUntil(t, tt, func() bool { return !st.loading })
	a.jumpNote(a.project("Web"), r, r.Notes[0])
	frameUntil(t, tt, func() bool { return len(a.jumps) == 0 })
	if notice := a.reviewNotices[noteKey(r.ID, 1)]; notice != st.file.note || notice == "" {
		t.Fatalf("notice %q, file %q", notice, st.file.note)
	}

	other := slices.IndexFunc(st.rows, func(row diffRow) bool { return row.dir == "" && row.change.Path == "other.txt" })
	st.selected = other
	tt.Frame()
	tt.Frame()
	if st.selected != other {
		t.Fatal("the Diff went back to the note's file")
	}
}

func editorPos(line, col int) editor.Pos { return editor.Pos{Line: line, Col: col} }

func diffLines(line int) editor.DiffLineRange {
	return editor.DiffLineRange{Side: editor.DiffNew, Start: line, End: line}
}
