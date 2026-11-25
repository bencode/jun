package app

import (
	"path/filepath"
	"testing"

	"github.com/bencode/jun/internal/tool"
)

// An agent that adds or changes a task is recorded in it, in its role,
// once; the task's owner is a role and its project the caller's unless
// it says; a message about a task needs the task; all of it is there in
// the next run.
func TestTasksRecordTheAgentsThatTouchThem(t *testing.T) {
	config := filepath.Join(t.TempDir(), "config.json")
	a, _, sessions := agentsApp(t)
	a.restoreRoles(config)
	a.restoreGoals(config)
	claude, codex := sessions[0], sessions[1]
	own := tool.Sessions{Claude: tool.AgentSession{ID: "c1"}}
	if err := a.toolUpdateTerminal(tool.Request{Op: tool.OpUpdateTerminal, Terminal: uint64(claude.ID), Name: ptr("高斯"),
		Sessions: own}, shellOf(t, claude)); err != nil {
		t.Fatal(err)
	}

	goal, err := a.toolAddGoal(tool.Request{Item: &tool.Item{Title: ptr("Goals in Jun"), Outcome: ptr("Agents use them")}}, shellOf(t, claude))
	if err != nil {
		t.Fatal(err)
	}
	add := func(item tool.Item) (int, error) {
		item.Goal, item.Title = &goal, ptr("Write the model")
		return a.toolAddTask(tool.Request{Item: &item, Sessions: own}, shellOf(t, claude))
	}
	if _, err := add(tool.Item{Owner: ptr("欧拉")}); err == nil {
		t.Fatal("owner that is no role taken")
	}
	if _, err := add(tool.Item{Project: ptr("web")}); err == nil {
		t.Fatal("project that is not there taken")
	}
	task, err := add(tool.Item{Owner: ptr("高斯"), Priority: ptr("P1")})
	if err != nil {
		t.Fatal(err)
	}
	update := func(from []int, sessions tool.Sessions, item tool.Item) error {
		item.ID = task
		return a.toolUpdateTask(tool.Request{Item: &item, Sessions: sessions}, from)
	}
	// P2, the default, is kept as none.
	if err := update(shellOf(t, claude), own, tool.Item{State: ptr("doing"), Priority: ptr("P2")}); err != nil {
		t.Fatal(err)
	}
	// Codex, in a terminal of no role, works on it too; a shell with no
	// agent session is recorded as none.
	if err := update(shellOf(t, codex), tool.Sessions{Codex: tool.AgentSession{ID: "x1"}}, tool.Item{Rank: 1}); err != nil {
		t.Fatal(err)
	}
	if err := update(shellOf(t, codex), tool.Sessions{}, tool.Item{Rank: 1}); err != nil {
		t.Fatal(err)
	}
	if _, err := a.toolTasks(tool.Request{Filter: &tool.Filter{Goal: 99}}); err == nil {
		t.Fatal("tasks of no goal listed")
	}
	if _, err := a.toolMessage(tool.Request{Text: "Asks", Task: 99}, shellOf(t, claude)); err == nil {
		t.Fatal("message about no task taken")
	}
	if _, err := a.toolMessage(tool.Request{Text: "Asks", Task: task}, shellOf(t, claude)); err != nil {
		t.Fatal(err)
	}

	check := func(a *app) {
		t.Helper()
		goals, tasks, err := a.toolTaskDetail(tool.Request{Item: &tool.Item{ID: task}})
		if err != nil {
			t.Fatal(err)
		}
		got := tasks[0]
		if len(goals) != 1 || got.Owner != "高斯" || got.Project != "Jun" || got.State != "doing" || got.Priority != "" || got.Started == nil ||
			len(got.Sessions) != 2 || got.Sessions[0].Role != "高斯" || got.Sessions[0].ID != "c1" ||
			got.Sessions[1].Role != "" || got.Sessions[1].Agent != "codex" || got.Sessions[0].Resume == "" {
			t.Fatalf("task: %+v", got)
		}
	}
	check(a)
	if m := a.toolMessages(); m[0].Task != task {
		t.Fatalf("message's task: %+v", m[0])
	}
	next, _, _ := agentsApp(t)
	next.restoreRoles(config)
	next.restoreGoals(config)
	check(next)
}

func ptr[T any](v T) *T { return &v }
