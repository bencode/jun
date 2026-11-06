package review

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/bencode/jun/internal/layout"
)

func TestLocate(t *testing.T) {
	text := "a\nb  \nc\n\nb\nc\nend"
	at := func(first, last int, quote string) Anchor {
		return Anchor{Lines: &Lines{first, last}, Quote: quote}
	}
	tests := []struct {
		name string
		a    Anchor
		want Place
	}{
		{"where it was", at(2, 3, "b\nc"), Place{Lines{2, 3}, Exact}},
		{"ending space ignored", at(2, 2, "b"), Place{Lines{2, 2}, Exact}},
		{"moved", at(1, 1, "end"), Place{Lines{7, 7}, Moved}},
		{"nearest of repeats", at(6, 6, "b"), Place{Lines{5, 5}, Moved}},
		{"nearest of repeats before", at(3, 3, "b"), Place{Lines{2, 2}, Moved}},
		{"lost stays", at(3, 4, "gone"), Place{Lines{3, 4}, Lost}},
		{"lost past the end", at(40, 45, "gone"), Place{Lines{7, 7}, Lost}},
		{"whole file", Anchor{}, Place{}},
		{"match over quote", Anchor{Lines: &Lines{1, 1}, Quote: "plain", Match: "end"}, Place{Lines{7, 7}, Moved}},
	}
	for _, tt := range tests {
		if got := Locate(text, tt.a); got != tt.want {
			t.Errorf("%s: Locate = %+v, want %+v", tt.name, got, tt.want)
		}
	}
}

func TestNotesAndFinish(t *testing.T) {
	now := time.Date(2026, 11, 5, 14, 32, 10, 0, time.UTC)
	r := New("/p", "feat", "abc", now)
	if _, err := Finish(r, now); err == nil {
		t.Fatal("finished a review without notes")
	}
	base := r
	r = AddNote(r, Anchor{Path: "a.go", Uncommitted: true}, "one")
	r = AddNote(r, Anchor{Path: "b.go"}, "two")
	if len(base.Notes) != 0 {
		t.Fatalf("AddNote changed the review it took: %+v", base)
	}
	if !r.Uncommitted() {
		t.Fatal("a note on uncommitted text does not mark the review")
	}
	r = DeleteNote(r, 1)
	r = AddNote(r, Anchor{Path: "c.go"}, "three")
	r = EditNote(r, 2, "edited")
	if r.Uncommitted() || r.Notes[0].Text != "edited" || r.Notes[1].ID != 3 {
		t.Fatalf("notes: %+v", r.Notes)
	}
	done, err := Finish(r, now)
	if err != nil || !done.Done() || r.Done() {
		t.Fatalf("Finish = %+v, %v", done, err)
	}
}

func TestStoreKeepsReviewsApartByProject(t *testing.T) {
	root := Root(filepath.Join(t.TempDir(), "config.json"))
	now := time.Date(2026, 11, 5, 14, 32, 10, 0, time.UTC)
	older := AddNote(New("/a/web", "feat/x", "abc", now), Anchor{
		Diff: &layout.DiffRef{Source: "feat/x", Base: "main"}, Path: "a.go", Side: SideNew, Lines: &Lines{1, 2}, Quote: "q",
	}, "text")
	newer := New("/a/web", "main", "", now.Add(time.Hour))
	other := New("/b/web", "main", "", now)
	for _, r := range []Review{older, newer, other} {
		if err := Save(root, r); err != nil {
			t.Fatal(err)
		}
	}
	got, err := List(root, "/a/web")
	if err != nil {
		t.Fatal(err)
	}
	if want := []Review{newer, older}; !reflect.DeepEqual(got, want) {
		t.Fatalf("List = %+v, want %+v", got, want)
	}
	if err := Delete(root, newer); err != nil {
		t.Fatal(err)
	}
	if err := Delete(root, newer); err != nil {
		t.Fatalf("deleting a review already gone: %v", err)
	}
	if got, _ := List(root, "/a/web"); len(got) != 1 {
		t.Fatalf("after Delete: %+v", got)
	}
	if got, err := List(root, "/none"); got != nil || err != nil {
		t.Fatalf("a project without reviews: %v, %v", got, err)
	}
}

func TestListTellsOfABadFileAndKeepsTheOthers(t *testing.T) {
	root := t.TempDir()
	good := New("/p", "main", "", time.Now())
	if err := Save(root, good); err != nil {
		t.Fatal(err)
	}
	bad := filepath.Join(filepath.Dir(Path(root, good)), "bad.json")
	if err := os.WriteFile(bad, []byte("{"), 0600); err != nil {
		t.Fatal(err)
	}
	got, err := List(root, "/p")
	if len(got) != 1 || err == nil || !strings.Contains(err.Error(), "bad.json") {
		t.Fatalf("List = %+v, %v", got, err)
	}
}
