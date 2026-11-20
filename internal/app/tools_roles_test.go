package app

import (
	"path/filepath"
	"testing"

	"github.com/bencode/jun/internal/terminal"
	"github.com/bencode/jun/internal/tool"
)

// An agent naming itself gives the role its session, kept across Jun's
// runs; naming another's terminal gives it none.
func TestRolesKeepTheSessionsOfAgentsThatNameThemselves(t *testing.T) {
	config := filepath.Join(t.TempDir(), "config.json")
	run := func() (*app, []*terminal.Session) {
		a, _, sessions := agentsApp(t)
		a.restoreRoles(config)
		return a, sessions
	}
	name := func(a *app, s *terminal.Session, callers []int, name, session string) error {
		t.Helper()
		return a.toolUpdateTerminal(tool.Request{Op: tool.OpUpdateTerminal, Terminal: uint64(s.ID), Name: &name,
			Sessions: tool.Sessions{Claude: session, Codex: "codex-" + session}}, callers)
	}
	listed := func(a *app, role string, limit int) tool.Role {
		t.Helper()
		list, err := a.toolRoles(tool.Request{Op: tool.OpRoles, Name: &role, Limit: limit})
		if err != nil {
			t.Fatal(err)
		}
		return list[0]
	}

	a, sessions := run()
	claude, codex := sessions[0], sessions[1]
	if err := name(a, claude, shellOf(t, claude), "狄仁杰", "s1"); err != nil {
		t.Fatal(err)
	}
	if r := listed(a, "狄仁杰", 0); r.Terminal != uint64(claude.ID) || r.Count != 1 || r.Sessions[0].Agent != "claude" || r.Sessions[0].ID != "s1" {
		t.Fatalf("named itself: %+v", r)
	}
	// Each agent's own variable: Codex's, in its terminal.
	if err := name(a, codex, shellOf(t, codex), "欧拉", "s2"); err != nil {
		t.Fatal(err)
	}
	if r := listed(a, "欧拉", 0); r.Last.Agent != "codex" || r.Last.ID != "codex-s2" {
		t.Fatalf("Codex named itself: %+v", r)
	}
	// Named by another, a terminal's name lasts while it runs, with no role.
	if err := name(a, sessions[2], shellOf(t, claude), "高斯", "s3"); err != nil {
		t.Fatal(err)
	}
	if list, _ := a.toolRoles(tool.Request{Op: tool.OpRoles}); len(list) != 2 || list[0].Name != "欧拉" || list[1].Name != "狄仁杰" {
		t.Fatalf("roles %+v", list)
	}
	if err := name(a, claude, shellOf(t, claude), "x", "bad id"); err == nil {
		t.Fatal("a session id that is none taken")
	}

	// Jun again: the role stays, with no terminal; a session finds its own,
	// and a new session taking it adds to its memory, the latest first.
	a, sessions = run()
	if mine, _ := a.toolRoles(tool.Request{Op: tool.OpRoles, Mine: true, Sessions: tool.Sessions{Claude: "s1"}}); len(mine) != 1 || mine[0].Name != "狄仁杰" {
		t.Fatalf("its role: %+v", mine)
	}
	if mine, _ := a.toolRoles(tool.Request{Op: tool.OpRoles, Mine: true, Sessions: tool.Sessions{Claude: "s9"}}); len(mine) != 0 {
		t.Fatalf("a session of no role: %+v", mine)
	}
	claude = sessions[0]
	if r := listed(a, "狄仁杰", 0); r.Terminal != 0 || r.Count != 1 {
		t.Fatalf("after a restart: %+v", r)
	}
	if err := name(a, claude, shellOf(t, claude), "狄仁杰", "s4"); err != nil {
		t.Fatal(err)
	}
	if r := listed(a, "狄仁杰", 1); r.Count != 2 || len(r.Sessions) != 1 || r.Sessions[0].ID != "s4" {
		t.Fatalf("taken again: %+v", r)
	}
	// Renamed, the session moves; dropped, it leaves, and an empty role goes.
	if err := name(a, claude, shellOf(t, claude), "Huygens", "s4"); err != nil {
		t.Fatal(err)
	}
	if err := name(a, claude, shellOf(t, claude), "", "s4"); err != nil {
		t.Fatal(err)
	}
	huygens := "huygens"
	if _, err := a.toolRoles(tool.Request{Op: tool.OpRoles, Name: &huygens}); err == nil {
		t.Fatal("a role with no session listed")
	}
	if r := listed(a, "狄仁杰", 0); r.Count != 1 || r.Last.ID != "s1" {
		t.Fatalf("the old role: %+v", r)
	}
}
