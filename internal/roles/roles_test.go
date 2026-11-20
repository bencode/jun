package roles

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func session(id string) Session {
	return Session{Agent: "claude", ID: id, Dir: "/w", At: time.Date(2026, 11, 20, 10, 0, 0, 0, time.UTC)}
}

func ids(r Role) []string {
	var out []string
	for _, s := range r.Sessions {
		out = append(out, s.ID)
	}
	return out
}

// Sessions take roles, move between them, and leave them; a role with none
// goes, and the files replay to the same book.
func TestRolesKeepTheirSessions(t *testing.T) {
	dir := t.TempDir()
	book := Book{}
	step := func(next Book, changes []Change) {
		t.Helper()
		if err := Write(dir, changes); err != nil {
			t.Fatal(err)
		}
		book = next
		loaded, err := Load(dir)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(loaded, book) {
			t.Fatalf("loaded %v, want %v", loaded, book)
		}
	}

	step(Take(book, "狄仁杰", session("a")))
	step(Take(book, "狄仁杰", session("b")))
	// Again, or in another case, it changes nothing.
	if _, changes := Take(book, "狄仁杰", session("a")); changes != nil {
		t.Fatalf("taken again: %v", changes)
	}
	if got := ids(book[Key("狄仁杰")]); !reflect.DeepEqual(got, []string{"a", "b"}) {
		t.Fatalf("memory %v", got)
	}
	// Renamed, the session moves; the old role keeps the rest.
	step(Take(book, "Euler", session("b")))
	if got := ids(book[Key("狄仁杰")]); !reflect.DeepEqual(got, []string{"a"}) {
		t.Fatalf("old role %v", got)
	}
	if r, ok := book.Of(session("b")); !ok || r.Name != "Euler" {
		t.Fatalf("b is %v", r)
	}
	// Its last session gone, a role goes, and its name with it.
	step(Take(book, "euler", session("a")))
	if _, ok := book[Key("狄仁杰")]; ok || len(book) != 1 || book[Key("Euler")].Name != "Euler" {
		t.Fatalf("book %v", book)
	}
	step(Leave(book, session("a")))
	step(Leave(book, session("b")))
	if len(book) != 0 {
		t.Fatalf("book %v", book)
	}
	// A role taken again after it went keeps its file.
	step(Take(book, "狄仁杰", session("c")))
}

// A line cut short, Jun stopping as it wrote, is left out, and the next
// change starts a line of its own.
func TestRolesReadPastALineCutShort(t *testing.T) {
	dir := t.TempDir()
	book, changes := Take(Book{}, "高斯", session("a"))
	if err := Write(dir, changes); err != nil {
		t.Fatal(err)
	}
	path := file(dir, "高斯")
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(`{"add":{"agent":"cla`); err != nil {
		t.Fatal(err)
	}
	f.Close()
	if loaded, err := Load(dir); err != nil || !reflect.DeepEqual(loaded, book) {
		t.Fatalf("loaded %v, %v", loaded, err)
	}
	book, changes = Take(book, "高斯", session("b"))
	if err := Write(dir, changes); err != nil {
		t.Fatal(err)
	}
	if loaded, err := Load(dir); err != nil || !reflect.DeepEqual(loaded, book) {
		t.Fatalf("loaded %v, %v", loaded, err)
	}
	// A broken line elsewhere is an error, not memory lost unseen.
	if err := os.WriteFile(filepath.Join(dir, "x.jsonl"), []byte("{\"name\":\"x\"}\nnope\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(dir); err == nil {
		t.Fatal("a broken file loaded")
	}
}

func TestNoRolesBeforeTheFirst(t *testing.T) {
	book, err := Load(filepath.Join(t.TempDir(), "roles"))
	if err != nil || len(book) != 0 {
		t.Fatalf("%v, %v", book, err)
	}
}
