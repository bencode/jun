package app

import (
	"maps"
	"path/filepath"
	"slices"
	"testing"

	"github.com/bencode/jun/internal/agent"
	"github.com/bencode/jun/internal/settings"
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
			Sessions: tool.Sessions{Claude: tool.AgentSession{ID: session}, Codex: tool.AgentSession{ID: "codex-" + session}}}, callers)
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
	if mine, _ := a.toolRoles(tool.Request{Op: tool.OpRoles, Mine: true, Sessions: tool.Sessions{Claude: tool.AgentSession{ID: "s1"}}}); len(mine) != 1 || mine[0].Name != "狄仁杰" {
		t.Fatalf("its role: %+v", mine)
	}
	if mine, _ := a.toolRoles(tool.Request{Op: tool.OpRoles, Mine: true, Sessions: tool.Sessions{Claude: tool.AgentSession{ID: "s9"}}}); len(mine) != 0 {
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
	// Its agent gone, a terminal gives its role up, for the session resumed
	// elsewhere; a name that is no role stays.
	other := sessions[1]
	if err := name(a, sessions[2], nil, "高斯", ""); err != nil {
		t.Fatal(err)
	}
	gone := maps.Clone(a.agents)
	delete(gone, claude.ID)
	delete(gone, sessions[2].ID)
	gone[other.ID] = agentInfo{agent.Claude, agent.Idle} // where the session is resumed
	a.releaseRoles(gone)
	a.agents = gone
	if a.names[claude.ID] != "" || a.names[sessions[2].ID] != "高斯" {
		t.Fatalf("names after the agents went: %v", a.names)
	}
	if err := name(a, other, shellOf(t, other), "狄仁杰", "s4"); err != nil || a.names[other.ID] != "狄仁杰" || listed(a, "狄仁杰", 0).Count != 2 {
		t.Fatalf("the role taken back elsewhere: %v", err)
	}
	claude = other
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

// Roles keep whom they report to, as their agents join and leave, and
// whether they are open: taken by an agent that has not ended. Each tells
// the command that resumes its sessions.
func TestRolesKeepTheirLeadsAndWhetherOpen(t *testing.T) {
	config := filepath.Join(t.TempDir(), "config.json")
	home := filepath.Join(t.TempDir(), "rec")
	run := func() (*app, []*terminal.Session) {
		a, _, sessions := agentsApp(t)
		a.prefs.Claude.ConfigDirs = []settings.ConfigDir{{Path: home, Command: "reclaude"}}
		a.restoreRoles(config)
		return a, sessions
	}
	sessions := tool.Sessions{Claude: tool.AgentSession{ID: "s1", Home: home}, Codex: tool.AgentSession{ID: "s2"},
		Pi: tool.AgentSession{ID: "s3", Path: "/p/s3.jsonl"}}
	name := func(a *app, s *terminal.Session, name string) {
		t.Helper()
		if err := a.toolUpdateTerminal(tool.Request{Op: tool.OpUpdateTerminal, Terminal: uint64(s.ID), Name: &name,
			Sessions: sessions}, shellOf(t, s)); err != nil {
			t.Fatal(err)
		}
	}
	roleOf := func(a *app, req tool.Request) []tool.Role {
		t.Helper()
		req.Op = tool.OpRoles
		list, err := a.toolRoles(req)
		if err != nil {
			t.Fatal(err)
		}
		return list
	}
	named := func(list []tool.Role) []string {
		var out []string
		for _, r := range list {
			out = append(out, r.Name+">"+r.Lead)
		}
		return out
	}
	str := func(s string) *string { return &s }
	join := func(a *app, s, lead *terminal.Session) {
		t.Helper()
		if err := a.toolJoin(tool.Request{Op: tool.OpJoin, Terminal: uint64(lead.ID)}, shellOf(t, s)); err != nil {
			t.Fatal(err)
		}
	}

	a, ts := run()
	holmes, di, song := ts[0], ts[1], ts[2]
	name(a, holmes, "福尔摩斯")
	name(a, di, "狄仁杰")
	join(a, di, holmes)
	// Started by its lead, as run does, an agent naming itself reports to
	// the lead's role.
	a.leads[song.ID] = holmes.ID
	name(a, song, "宋慈")
	if got := named(roleOf(a, tool.Request{Team: str("福尔摩斯")})); !slices.Equal(got, []string{"福尔摩斯>", "宋慈>福尔摩斯", "狄仁杰>福尔摩斯"}) {
		t.Fatalf("team %v", got)
	}
	all := roleOf(a, tool.Request{})
	if !all[0].Open || !all[1].Open || !all[2].Open {
		t.Fatalf("open %+v", all)
	}
	// The resume commands: the home's command, the agent's own, Pi's file.
	if got := []string{all[2].Last.Resume, all[1].Last.Resume, all[0].Last.Resume}; !slices.Equal(got,
		[]string{"reclaude --resume s1", "codex resume s2", "pi --session '/p/s3.jsonl'"}) {
		t.Fatalf("resume %v", got)
	}
	// Leaving drops the lead; joining a terminal that is no role does too.
	if err := a.toolLeave(tool.Request{Op: tool.OpLeave}, shellOf(t, di)); err != nil {
		t.Fatal(err)
	}
	if r := roleOf(a, tool.Request{Name: str("狄仁杰")}); r[0].Lead != "" {
		t.Fatalf("left: %+v", r)
	}
	join(a, di, holmes)
	// Its agent ended in a shell that runs on, a role is closed; one whose
	// terminal ended stays open.
	gone := maps.Clone(a.agents)
	delete(gone, di.ID)
	delete(gone, song.ID)
	song.Send("exit")
	eventually(t, "the shell to end", song.Exited)
	a.releaseRoles(gone)
	a.agents = gone
	if got := roleOf(a, tool.Request{Name: str("狄仁杰")}); got[0].Open {
		t.Fatalf("ended: %+v", got)
	}

	// Jun again: what was open and is no terminal's comes back leads
	// first, and a session resumed by another, which leads it as it runs,
	// keeps the lead it had.
	a, ts = run()
	if got := named(roleOf(a, tool.Request{Away: true})); !slices.Equal(got, []string{"福尔摩斯>", "宋慈>福尔摩斯"}) {
		t.Fatalf("to bring back %v", got)
	}
	holmes, song = ts[0], ts[2]
	name(a, holmes, "福尔摩斯")
	join(a, ts[1], holmes)
	name(a, ts[1], "狄仁杰") // its lead is the coordinator's terminal too
	a.leads[song.ID] = ts[1].ID
	name(a, song, "宋慈")
	if got := named(roleOf(a, tool.Request{})); !slices.Equal(got, []string{"宋慈>福尔摩斯", "狄仁杰>福尔摩斯", "福尔摩斯>"}) {
		t.Fatalf("brought back %v", got)
	}
	if got := roleOf(a, tool.Request{Away: true}); len(got) != 0 {
		t.Fatalf("still away %+v", got)
	}
}
