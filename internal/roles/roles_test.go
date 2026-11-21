package roles

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func session(id string) Session {
	return Session{Agent: "claude", ID: id, Dir: "/w", At: time.Date(2026, 11, 20, 10, 0, 0, 0, time.UTC)}
}

func ids(b Book, name string) []string {
	r, _ := b.Named(name)
	var out []string
	for _, s := range b.Memory(r.ID) {
		out = append(out, s.ID)
	}
	return out
}

func lead(b Book, name string) string {
	r, _ := b.Named(name)
	l, _ := b.Role(r.Lead)
	return l.Name
}

// Sessions take roles, move between them and leave them; roles get leads
// and are opened and closed; each book saved reads back the same.
func TestRolesKeepTheirSessionsAndLeads(t *testing.T) {
	path := filepath.Join(t.TempDir(), "roles.json")
	book := Book{}
	step := func(next Book) {
		t.Helper()
		if err := Save(path, next); err != nil {
			t.Fatal(err)
		}
		loaded, err := Load(path)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(loaded, next) {
			t.Fatalf("loaded %+v, want %+v", loaded, next)
		}
		book = next
	}

	step(Take(book, "狄仁杰", session("a")))
	step(Take(book, "狄仁杰", session("b")))
	step(Take(book, "福尔摩斯", session("c")))
	// Taken again, in any case, nothing changes.
	if again := Take(book, "狄仁杰", session("a")); !reflect.DeepEqual(again, book) {
		t.Fatalf("taken again: %+v", again)
	}
	if got := ids(book, "狄仁杰"); !reflect.DeepEqual(got, []string{"a", "b"}) {
		t.Fatalf("memory %v", got)
	}
	step(SetOpen(SetLead(book, "狄仁杰", "福尔摩斯"), "狄仁杰", true))
	if r, _ := book.Named("狄仁杰"); lead(book, "狄仁杰") != "福尔摩斯" || !r.Open {
		t.Fatalf("lead and open: %+v", book.Roles)
	}
	// The latest lead wins: the one it led stops reporting to it.
	if next := SetLead(book, "福尔摩斯", "狄仁杰"); lead(next, "福尔摩斯") != "狄仁杰" || lead(next, "狄仁杰") != "" {
		t.Fatalf("turned round: %+v", next.Roles)
	}
	// A lead that is the role or no role changes nothing, nor does a value
	// set again.
	for _, next := range []Book{SetLead(book, "狄仁杰", "狄仁杰"),
		SetLead(book, "狄仁杰", "nobody"), SetLead(book, "狄仁杰", "福尔摩斯"), SetOpen(book, "狄仁杰", true), SetOpen(book, "nobody", true)} {
		if !reflect.DeepEqual(next, book) {
			t.Fatalf("changed: %+v", next)
		}
	}
	// Renamed, a session moves, and the role it leaves keeps its lead.
	step(Take(book, "Euler", session("b")))
	if r, ok := book.Of("claude", "b"); !ok || r.Name != "Euler" || lead(book, "狄仁杰") != "福尔摩斯" {
		t.Fatalf("moved: %+v", book)
	}
	// A role left with no session goes, as a lead too.
	step(Leave(book, "claude", "c"))
	if _, ok := book.Named("福尔摩斯"); ok || lead(book, "狄仁杰") != "" {
		t.Fatalf("lead gone: %+v", book)
	}
	step(SetLead(book, "狄仁杰", ""))
	step(Leave(Leave(book, "claude", "a"), "claude", "b"))
	if len(book.Roles) != 0 || len(book.Sessions) != 0 {
		t.Fatalf("book %+v", book)
	}
}

// A file that breaks what a book holds is an error, not memory lost unseen.
func TestRolesRefuseABrokenFile(t *testing.T) {
	dir := t.TempDir()
	if book, err := Load(filepath.Join(dir, "none.json")); err != nil || len(book.Roles) != 0 {
		t.Fatalf("no file: %+v, %v", book, err)
	}
	s := `{"role":1,"agent":"claude","id":"a","dir":"/w","at":"2026-11-20T10:00:00Z"}`
	for name, doc := range map[string]string{
		"version":       `{"version":2,"roles":[],"sessions":[]}`,
		"unknown field": `{"version":1,"roles":[],"sessions":[],"x":1}`,
		"same name":     `{"version":1,"roles":[{"id":1,"name":"A"},{"id":2,"name":"a"}],"sessions":[` + s + `]}`,
		"same id":       `{"version":1,"roles":[{"id":1,"name":"A"},{"id":1,"name":"B"}],"sessions":[` + s + `]}`,
		"same session":  `{"version":1,"roles":[{"id":1,"name":"A"}],"sessions":[` + s + `,` + s + `]}`,
		"no role":       `{"version":1,"roles":[],"sessions":[` + s + `]}`,
		"no session":    `{"version":1,"roles":[{"id":1,"name":"A"},{"id":2,"name":"B"}],"sessions":[` + s + `]}`,
		"lead missing":  `{"version":1,"roles":[{"id":1,"name":"A","lead":3}],"sessions":[` + s + `]}`,
		"lead cycle": `{"version":1,"roles":[{"id":1,"name":"A","lead":2},{"id":2,"name":"B","lead":1}],"sessions":[` + s + `,` +
			strings.Replace(strings.Replace(s, `"a"`, `"b"`, 1), `"role":1`, `"role":2`, 1) + `]}`,
		"cut short": `{"version":1,"roles":[`,
	} {
		path := filepath.Join(dir, strings.ReplaceAll(name, " ", "-")+".json")
		if err := os.WriteFile(path, []byte(doc), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(path); err == nil {
			t.Errorf("%s: loaded", name)
		}
	}
}
