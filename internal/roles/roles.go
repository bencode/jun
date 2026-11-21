// Package roles keeps what agents are called across sessions: a name is a
// role, an entity, and the sessions that took it are its memory. A role
// may report to another, its lead.
package roles

import (
	"fmt"
	"slices"
	"strings"
	"time"
)

// Role is a name that lasts. Its lead is another role's ID, 0 for none;
// it is open from when an agent takes the name until the agent ends.
type Role struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Lead int    `json:"lead"`
	Open bool   `json:"open"`
}

// Session is one of a role's activities: an agent's session, as the agent
// knows it, in the agent's directory Home (empty for its default) and, for
// Pi, its file Path.
type Session struct {
	Role  int       `json:"role"`
	Agent string    `json:"agent"` // claude, codex or pi
	ID    string    `json:"id"`
	Home  string    `json:"home,omitempty"`
	Path  string    `json:"path,omitempty"`
	Dir   string    `json:"dir"`
	At    time.Time `json:"at"` // when it took the role
}

// Book is the roles and their sessions, each in the order it came. Every
// role has a session; Check tells what else holds.
type Book struct {
	Roles    []Role    `json:"roles"`
	Sessions []Session `json:"sessions"`
}

// Key is what tells names apart: their case does not.
func Key(name string) string { return strings.ToLower(name) }

// Named is the role of the name, whatever its case.
func (b Book) Named(name string) (Role, bool) {
	i := slices.IndexFunc(b.Roles, func(r Role) bool { return Key(r.Name) == Key(name) })
	if i < 0 {
		return Role{}, false
	}
	return b.Roles[i], true
}

// Role is the role of the ID.
func (b Book) Role(id int) (Role, bool) {
	i := slices.IndexFunc(b.Roles, func(r Role) bool { return r.ID == id })
	if i < 0 {
		return Role{}, false
	}
	return b.Roles[i], true
}

// Of is the role whose memory has the agent's session.
func (b Book) Of(agent, id string) (Role, bool) {
	i := slices.IndexFunc(b.Sessions, func(s Session) bool { return s.Agent == agent && s.ID == id })
	if i < 0 {
		return Role{}, false
	}
	return b.Role(b.Sessions[i].Role)
}

// Memory is the role's sessions, in the order they took it.
func (b Book) Memory(role int) []Session {
	var mine []Session
	for _, s := range b.Sessions {
		if s.Role == role {
			mine = append(mine, s)
		}
	}
	return mine
}

// Take gives the session to the role of the name, a new one if there is
// none, out of the one it was in. A session already the role's changes
// nothing.
func Take(b Book, name string, s Session) Book {
	r, ok := b.Named(name)
	if old, was := b.Of(s.Agent, s.ID); was && ok && old.ID == r.ID {
		return b
	}
	if !ok {
		r = Role{ID: nextID(b), Name: name}
		b.Roles = append(slices.Clone(b.Roles), r)
	}
	s.Role = r.ID
	b = Leave(b, s.Agent, s.ID)
	b.Sessions = append(slices.Clone(b.Sessions), s)
	return b
}

// Leave takes the agent's session out of its role. A role left with none
// goes, and so does every lead it was.
func Leave(b Book, agent, id string) Book {
	r, ok := b.Of(agent, id)
	if !ok {
		return b
	}
	var sessions []Session
	for _, s := range b.Sessions {
		if s.Agent != agent || s.ID != id {
			sessions = append(sessions, s)
		}
	}
	b.Sessions = sessions
	if len(b.Memory(r.ID)) > 0 {
		return b
	}
	var kept []Role
	for _, other := range b.Roles {
		if other.ID == r.ID {
			continue
		}
		if other.Lead == r.ID {
			other.Lead = 0
		}
		kept = append(kept, other)
	}
	b.Roles = kept
	return b
}

// SetLead makes the role of the lead's name the named role's lead, or
// drops its lead for none. The latest lead wins: one the role led, itself
// or through others, stops reporting to it. A lead that is no role, or
// the role itself, changes nothing.
func SetLead(b Book, name, lead string) Book {
	r, ok := b.Named(name)
	if !ok {
		return b
	}
	id := 0
	if lead != "" {
		l, ok := b.Named(lead)
		if !ok || l.ID == r.ID {
			return b
		}
		id = l.ID
	}
	if r.Lead == id {
		return b
	}
	led := id
	for seen := 0; led != 0 && seen <= len(b.Roles); seen++ {
		above, _ := b.Role(led)
		if above.Lead == r.ID {
			break
		}
		led = above.Lead
	}
	next := make([]Role, len(b.Roles))
	for i, other := range b.Roles {
		if other.ID == r.ID {
			other.Lead = id
		} else if other.ID == led {
			other.Lead = 0
		}
		next[i] = other
	}
	b.Roles = next
	return b
}

// SetOpen marks the named role open or not.
func SetOpen(b Book, name string, open bool) Book {
	return update(b, name, func(r Role) Role { r.Open = open; return r })
}

// update is the book with the named role changed; the same book when it
// is no role or stays as it was.
func update(b Book, name string, change func(Role) Role) Book {
	r, ok := b.Named(name)
	if !ok || change(r) == r {
		return b
	}
	b.Roles = slices.Clone(b.Roles)
	b.Roles[slices.Index(b.Roles, r)] = change(r)
	return b
}

// leads reports whether the role of the ID leads the other, or any it
// leads does: a role is its own lead here.
func leads(b Book, id, other int) bool {
	for seen := 0; other != 0 && seen <= len(b.Roles); seen++ {
		if other == id {
			return true
		}
		r, _ := b.Role(other)
		other = r.Lead
	}
	return false
}

func nextID(b Book) int {
	id := 0
	for _, r := range b.Roles {
		id = max(id, r.ID)
	}
	return id + 1
}

// Check tells what is wrong with the book, if anything: names and IDs are
// each one role's, a session is in one role that is there, a lead is a
// role that is there and not led by the role, and every role has a
// session.
func Check(b Book) error {
	names, ids := map[string]bool{}, map[int]bool{}
	for _, r := range b.Roles {
		if r.ID < 1 || ids[r.ID] {
			return fmt.Errorf("role %q: ID %d is not its own", r.Name, r.ID)
		}
		if r.Name == "" || names[Key(r.Name)] {
			return fmt.Errorf("role %d: name %q is not its own", r.ID, r.Name)
		}
		ids[r.ID], names[Key(r.Name)] = true, true
	}
	held := map[int]bool{}
	seen := map[[2]string]bool{}
	for _, s := range b.Sessions {
		key := [2]string{s.Agent, s.ID}
		if s.Agent == "" || s.ID == "" || seen[key] {
			return fmt.Errorf("%s session %q is not one of its own", s.Agent, s.ID)
		}
		if !ids[s.Role] {
			return fmt.Errorf("%s session %q: role %d is not there", s.Agent, s.ID, s.Role)
		}
		seen[key], held[s.Role] = true, true
	}
	for _, r := range b.Roles {
		if !held[r.ID] {
			return fmt.Errorf("role %q has no session", r.Name)
		}
		if r.Lead != 0 && (!ids[r.Lead] || leads(b, r.ID, r.Lead)) {
			return fmt.Errorf("role %q: lead %d is no role it can report to", r.Name, r.Lead)
		}
	}
	return nil
}
