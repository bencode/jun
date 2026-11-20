// Package roles keeps what agents are called across sessions: a name is a
// role, an entity, and the sessions that took it are its memory.
package roles

import (
	"strings"
	"time"
)

// Session is one of a role's activities: an agent's session, as the agent
// knows it.
type Session struct {
	Agent string    `json:"agent"` // claude, codex or pi
	ID    string    `json:"id"`
	Dir   string    `json:"dir"`
	At    time.Time `json:"at"` // when it took the role
}

// same reports whether s and other are the one session.
func (s Session) same(other Session) bool { return s.Agent == other.Agent && s.ID == other.ID }

// Role is a name and its memory, the sessions in the order they took it.
type Role struct {
	Name     string
	Sessions []Session
}

// Book is the roles with a memory, by key.
type Book map[string]Role

// Key is what tells names apart: their case does not.
func Key(name string) string { return strings.ToLower(name) }

// Change is a line a role's file gets: a session added or dropped.
type Change struct {
	Name string   `json:"-"`
	Add  *Session `json:"add,omitempty"`
	Drop *Session `json:"drop,omitempty"`
}

// Of is the role whose memory has the session, if any.
func (b Book) Of(s Session) (Role, bool) {
	for _, r := range b {
		for _, mine := range r.Sessions {
			if mine.same(s) {
				return r, true
			}
		}
	}
	return Role{}, false
}

// Take gives the session to the named role, out of the one it was in: the
// book after, and the changes its files get. A session already the role's
// changes nothing.
func Take(b Book, name string, s Session) (Book, []Change) {
	if old, ok := b.Of(s); ok && Key(old.Name) == Key(name) {
		return b, nil
	}
	b, changes := Leave(b, s)
	next := clone(b)
	r := next[Key(name)]
	if r.Name == "" {
		r.Name = name
	}
	r.Sessions = append(append([]Session{}, r.Sessions...), s)
	next[Key(name)] = r
	return next, append(changes, Change{Name: r.Name, Add: &s})
}

// Leave takes the session out of its role, which goes once it has none.
func Leave(b Book, s Session) (Book, []Change) {
	old, ok := b.Of(s)
	if !ok {
		return b, nil
	}
	next := clone(b)
	var kept []Session
	for _, mine := range old.Sessions {
		if !mine.same(s) {
			kept = append(kept, mine)
		}
	}
	if len(kept) == 0 {
		delete(next, Key(old.Name))
	} else {
		next[Key(old.Name)] = Role{Name: old.Name, Sessions: kept}
	}
	return next, []Change{{Name: old.Name, Drop: &Session{Agent: s.Agent, ID: s.ID}}}
}

// apply is the book after the change, as a file read back replays it.
func apply(b Book, c Change) Book {
	if c.Add != nil {
		b, _ = Take(b, c.Name, *c.Add)
	}
	if c.Drop != nil {
		if r, ok := b.Of(*c.Drop); ok && Key(r.Name) == Key(c.Name) {
			b, _ = Leave(b, *c.Drop)
		}
	}
	return b
}

func clone(b Book) Book {
	next := make(Book, len(b))
	for k, r := range b {
		next[k] = r
	}
	return next
}
