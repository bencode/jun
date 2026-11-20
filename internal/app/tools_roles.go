package app

import (
	"cmp"
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/bencode/jun/internal/history"
	"github.com/bencode/jun/internal/roles"
	"github.com/bencode/jun/internal/terminal"
	"github.com/bencode/jun/internal/tool"
)

// restoreRoles reads the roles kept beside the configuration. Ones that
// cannot be read are not written either: names then last while Jun runs.
func (a *app) restoreRoles(configPath string) {
	dir := roles.Dir(configPath)
	book, err := roles.Load(dir)
	if err != nil {
		a.report(fmt.Errorf("%s: %w", a.text("Roles could not be loaded; they are not kept for this session",
			"角色无法加载；本次运行不保存角色"), err))
		return
	}
	a.roles, a.rolesDir = book, dir
}

// ownSession is the session of the agent in s, when s is the caller's
// terminal and its variables name one: an agent naming itself gives its
// role that session.
func (a *app) ownSession(s *terminal.Session, req tool.Request, callers []int) (roles.Session, bool, error) {
	info, running := a.agents[s.ID]
	if caller, err := a.toolCaller(callers, req.Caller); err != nil || caller != s.ID || !running {
		return roles.Session{}, false, nil
	}
	id := req.Sessions.Of(string(info.Kind))
	if id == "" {
		return roles.Session{}, false, nil
	}
	if !history.ValidID(id) {
		return roles.Session{}, false, fmt.Errorf("%s session id %q is not one", info.Kind, id)
	}
	return roles.Session{Agent: string(info.Kind), ID: id, Dir: cmp.Or(s.Terminal.Dir(), s.CWD), At: time.Now().Truncate(time.Second)}, true, nil
}

// nameRole gives the session to the role of the name, or takes it out of
// its role for none: the name as the role has it.
func (a *app) nameRole(own roles.Session, name string) (string, error) {
	book, changes := roles.Leave(a.roles, own)
	if name != "" {
		book, changes = roles.Take(a.roles, name, own)
		name = book[roles.Key(name)].Name
	}
	if a.rolesDir != "" {
		if err := roles.Write(a.rolesDir, changes); err != nil {
			a.reloadRoles()
			return "", fmt.Errorf("role not kept: %w", err)
		}
	}
	a.roles = book
	return name, nil
}

// reloadRoles reads the roles again after a write failed part way, for
// what Jun knows to be what the files say.
func (a *app) reloadRoles() {
	book, err := roles.Load(a.rolesDir)
	if err != nil {
		a.report(fmt.Errorf("%s: %w", a.text("Roles could not be read again; they are not kept for this session",
			"角色无法重新读取；本次运行不保存角色"), err))
		a.rolesDir = ""
		return
	}
	a.roles = book
}

// toolRoles lists the roles by name, or the one named, or the caller's,
// with its latest sessions.
func (a *app) toolRoles(req tool.Request) ([]tool.Role, error) {
	held := map[string]terminal.ID{}
	for id, name := range a.names {
		if s := a.terminals.Get(id); s != nil && !s.Exited() {
			held[roles.Key(name)] = id
		}
	}
	if req.Mine {
		for _, agent := range []string{"claude", "codex", "pi"} {
			if id := req.Sessions.Of(agent); id != "" {
				if r, ok := a.roles.Of(roles.Session{Agent: agent, ID: id}); ok {
					return []tool.Role{toolRoleWith(r, held, req.Limit)}, nil
				}
			}
		}
		return nil, nil
	}
	if req.Name != nil {
		r, ok := a.roles[roles.Key(*req.Name)]
		if !ok {
			return nil, fmt.Errorf("no role %q", *req.Name)
		}
		return []tool.Role{toolRoleWith(r, held, req.Limit)}, nil
	}
	list := make([]tool.Role, 0, len(a.roles))
	for _, key := range slices.Sorted(maps.Keys(a.roles)) {
		list = append(list, toolRole(a.roles[key], held))
	}
	return list, nil
}

func toolRole(r roles.Role, held map[string]terminal.ID) tool.Role {
	last := toolRoleSession(r.Sessions[len(r.Sessions)-1])
	return tool.Role{Name: r.Name, Terminal: uint64(held[roles.Key(r.Name)]), Count: len(r.Sessions), Last: &last}
}

// toolRoleWith is the role with its latest sessions, the latest first.
func toolRoleWith(r roles.Role, held map[string]terminal.ID, limit int) tool.Role {
	role := toolRole(r, held)
	for _, s := range slices.Backward(r.Sessions) {
		if len(role.Sessions) == cmp.Or(limit, tool.DefaultRoleSessions) {
			break
		}
		role.Sessions = append(role.Sessions, toolRoleSession(s))
	}
	return role
}

func toolRoleSession(s roles.Session) tool.RoleSession {
	return tool.RoleSession{Agent: s.Agent, ID: s.ID, Dir: s.Dir, At: s.At}
}
