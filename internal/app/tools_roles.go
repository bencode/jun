package app

import (
	"cmp"
	"fmt"
	"log"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/bencode/jun/internal/history"
	"github.com/bencode/jun/internal/roles"
	"github.com/bencode/jun/internal/terminal"
	"github.com/bencode/jun/internal/tool"
)

// restoreRoles reads the roles kept beside the configuration. Ones that
// cannot be read are not written either: names then last while Jun runs.
func (a *app) restoreRoles(configPath string) {
	path := roles.Path(configPath)
	book, err := roles.Load(path)
	if err != nil {
		a.report(fmt.Errorf("%s: %w", a.text("Roles could not be loaded; they are not kept for this session",
			"角色无法加载；本次运行不保存角色"), err))
		return
	}
	a.roles, a.rolesPath = book, path
}

// setRoles keeps the book, when it changed: the file is replaced whole, so
// one not written leaves the roles as they were.
func (a *app) setRoles(next roles.Book) error {
	if reflect.DeepEqual(next, a.roles) {
		return nil
	}
	if a.rolesPath != "" {
		if err := roles.Save(a.rolesPath, next); err != nil {
			return err
		}
	}
	a.roles = next
	return nil
}

// ownSession is the session of the agent in s, when s is the caller's
// terminal and its variables name one: an agent naming itself gives its
// role that session.
func (a *app) ownSession(s *terminal.Session, req tool.Request, callers []int) (roles.Session, bool, error) {
	info, running := a.agents[s.ID]
	if caller, err := a.toolCaller(callers, req.Caller); err != nil || caller != s.ID || !running {
		return roles.Session{}, false, nil
	}
	kind := string(info.Kind)
	own := req.Sessions.Of(kind)
	if own.ID == "" {
		return roles.Session{}, false, nil
	}
	if !history.ValidID(own.ID) {
		return roles.Session{}, false, fmt.Errorf("%s session id %q is not one", kind, own.ID)
	}
	for _, path := range []string{own.Home, own.Path} {
		if path != "" && !filepath.IsAbs(path) {
			return roles.Session{}, false, fmt.Errorf("%s session %s: %q is not an absolute path", kind, own.ID, path)
		}
	}
	return roles.Session{Agent: kind, ID: own.ID, Home: cleanPath(own.Home), Path: cleanPath(own.Path),
		Dir: cmp.Or(s.Terminal.Dir(), s.CWD), At: time.Now().Truncate(time.Second)}, true, nil
}

func cleanPath(path string) string {
	if path == "" {
		return ""
	}
	return filepath.Clean(path)
}

// takeRole gives the session to the role of the name, or takes it out of
// its role for none: the book after, and the name as the role has it.
func takeRole(b roles.Book, own roles.Session, name string) (roles.Book, string) {
	if name == "" {
		return roles.Leave(b, own.Agent, own.ID), ""
	}
	b = roles.Take(b, name, own)
	r, _ := b.Named(name)
	return b, r.Name
}

// renameRoles is the book after the terminal's name goes from old to
// name: the role it leaves is no longer open; the one the terminal's agent
// took itself is, and when it reports to none yet, it reports to the
// terminal's lead's role.
func (a *app) renameRoles(b roles.Book, id terminal.ID, old, name string, own bool) roles.Book {
	if r, ok := b.Named(old); ok && roles.Key(old) != roles.Key(name) {
		b = roles.SetOpen(b, r.Name, false)
	}
	if !own || name == "" {
		return b
	}
	b = roles.SetOpen(b, name, true)
	if r, _ := b.Named(name); r.Lead == 0 {
		if lead, ok := a.terminalRole(b, a.lead(id)); ok {
			b = roles.SetLead(b, name, lead.Name)
		}
	}
	return b
}

// terminalRole is the role of the terminal's name, if it is one.
func (a *app) terminalRole(b roles.Book, id terminal.ID) (roles.Role, bool) {
	name := a.names[id]
	if id == 0 || name == "" {
		return roles.Role{}, false
	}
	return b.Named(name)
}

// keepLead records the terminal's lead, a terminal or none, as its role's,
// when its name is a role: the lead's role, or none when the lead's name
// is no role.
func (a *app) keepLead(id, lead terminal.ID) error {
	r, ok := a.terminalRole(a.roles, id)
	if !ok {
		return nil
	}
	l, _ := a.terminalRole(a.roles, lead)
	if err := a.setRoles(roles.SetLead(a.roles, r.Name, l.Name)); err != nil {
		return fmt.Errorf("the lead of role %q not kept: %w", r.Name, err)
	}
	return nil
}

// releaseRoles drops the name of each terminal whose agent is gone, as next
// polls them, when the name is a role: a role is its agent's, not the
// shell's, and the session resumed elsewhere takes it back. Its agent
// ended in a shell that still runs, the role is no longer open; one whose
// terminal ended, as Jun quit, stays open, for what was at work. Other
// names stay with their terminal.
func (a *app) releaseRoles(next map[terminal.ID]agentInfo) {
	book := a.roles
	for id, name := range a.names {
		_, was := a.agents[id]
		_, still := next[id]
		r, role := book.Named(name)
		if !was || still || !role {
			continue
		}
		delete(a.names, id)
		if s := a.terminals.Get(id); s != nil && !s.Exited() {
			book = roles.SetOpen(book, r.Name, false)
		}
	}
	if err := a.setRoles(book); err != nil {
		a.report(fmt.Errorf("%s: %w", a.text("Roles not kept", "角色未保存"), err))
	}
}

// toolRoles answers roles: all of them by name; one with its latest
// sessions; a team; the open ones no terminal has, leads first; or the
// caller's.
func (a *app) toolRoles(req tool.Request) ([]tool.Role, error) {
	held := map[string]terminal.ID{}
	for id, name := range a.names {
		if s := a.terminals.Get(id); s != nil && !s.Exited() {
			held[roles.Key(name)] = id
		}
	}
	summary := func(r roles.Role) tool.Role { return a.toolRole(r, held, 0) }
	limit := cmp.Or(req.Limit, tool.DefaultRoleSessions)
	all := slices.SortedFunc(slices.Values(a.roles.Roles), func(x, y roles.Role) int {
		return strings.Compare(roles.Key(x.Name), roles.Key(y.Name))
	})
	if req.Mine {
		for _, agent := range []string{"claude", "codex", "pi"} {
			if own := req.Sessions.Of(agent); own.ID != "" {
				if r, ok := a.roles.Of(agent, own.ID); ok {
					return []tool.Role{a.toolRole(r, held, limit)}, nil
				}
			}
		}
		return nil, nil
	}
	if req.Name != nil || req.Team != nil {
		name := *cmp.Or(req.Name, req.Team)
		r, ok := a.roles.Named(name)
		if !ok {
			return nil, fmt.Errorf("no role %q", name)
		}
		if req.Name != nil {
			return []tool.Role{a.toolRole(r, held, limit)}, nil
		}
		team := []tool.Role{summary(r)}
		for _, member := range all {
			if member.Lead == r.ID {
				team = append(team, summary(member))
			}
		}
		return team, nil
	}
	if req.Away {
		all = leadsFirst(slices.DeleteFunc(all, func(r roles.Role) bool { return !r.Open || held[roles.Key(r.Name)] != 0 }))
	}
	list := make([]tool.Role, 0, len(all))
	for _, r := range all {
		list = append(list, summary(r))
	}
	return list, nil
}

// leadsFirst orders the roles for each to come after its lead, when its
// lead is one of them, keeping their order otherwise.
func leadsFirst(rs []roles.Role) []roles.Role {
	var ordered []roles.Role
	for len(rs) > 0 {
		left := map[int]bool{}
		for _, r := range rs {
			left[r.ID] = true
		}
		var later []roles.Role
		for _, r := range rs {
			if left[r.Lead] {
				later = append(later, r)
			} else {
				ordered = append(ordered, r)
			}
		}
		rs = later
	}
	return ordered
}

// toolRole is the role as roles gives it, with its latest sessions, the
// latest first, up to the limit: none for no limit.
func (a *app) toolRole(r roles.Role, held map[string]terminal.ID, limit int) tool.Role {
	memory := a.roles.Memory(r.ID)
	lead, _ := a.roles.Role(r.Lead)
	last := a.toolRoleSession(memory[len(memory)-1])
	role := tool.Role{Name: r.Name, Lead: lead.Name, Open: r.Open, Terminal: uint64(held[roles.Key(r.Name)]),
		Count: len(memory), Last: &last}
	for _, s := range slices.Backward(memory) {
		if len(role.Sessions) == limit {
			break
		}
		role.Sessions = append(role.Sessions, a.toolRoleSession(s))
	}
	return role
}

func (a *app) toolRoleSession(s roles.Session) tool.RoleSession {
	resume, err := history.ResumeCommand(history.Session{Agent: s.Agent, ID: s.ID, Path: s.Path, Home: s.Home}, a.resumeCommands())
	if err != nil {
		log.Printf("Jun: resume of the %s session %s: %v", s.Agent, s.ID, err)
	}
	return tool.RoleSession{Agent: s.Agent, ID: s.ID, Home: s.Home, Dir: s.Dir, At: s.At, Resume: resume}
}
