package app

import (
	"path/filepath"
	"testing"

	"github.com/bencode/jun/internal/goals"
	"github.com/bencode/jun/internal/layout"
	"github.com/bencode/jun/internal/tool"
	"github.com/egoist/mygo/ui"
)

// tasksApp has a task to do, linked to a file of the project.
func tasksApp(t *testing.T) (*app, *ui.Tester, string) {
	t.Helper()
	a, tt := globalApp(t)
	notes := writeFile(t, filepath.Join(a.state.CurrentProject().CWD, "notes.md"))
	a.goals = goals.Book{
		LastGoal: 1, LastTask: 1,
		Goals: []goals.Goal{{ID: 1, Title: "Ship the views", Outcome: "Merged", State: goals.GoalOpen}},
		Tasks: []goals.Task{{ID: 1, Goal: 1, Title: "Write the notes", Link: "file:" + notes, Project: "Jun", State: goals.TaskTodo}},
	}
	tt.Frame()
	return a, tt, notes
}

func TestAClickChoosesATaskAndReturnOpensIt(t *testing.T) {
	a, tt, notes := tasksApp(t)
	a.toggleGlobalView(a.globalOf(layout.ViewTasks).ID)
	tt.Frame()
	click(t, tt, "Write the notes")
	tt.Frame()
	if a.selectedTask != 1 || len(fileTabs(a, notes)) != 0 {
		t.Fatalf("chose T%d, file tabs %v", a.selectedTask, fileTabs(a, notes))
	}
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if len(fileTabs(a, notes)) != 1 {
		t.Fatal("↩ did not open the task's link")
	}
}

func TestAMessagesTaskShowsInTasks(t *testing.T) {
	a, tt, notes := tasksApp(t)
	link(t, a, a.terminals.All()[0], "Notes drafted", target(tool.TargetFile, 0, notes))
	a.messages[len(a.messages)-1].Task = 1
	tt.Frame()
	click(t, tt, "T1 Write the notes")
	tt.Frame()
	if !a.globalOf(layout.ViewTasks).Show || a.selectedTask != 1 || len(fileTabs(a, notes)) != 0 {
		t.Fatalf("Tasks shown %v, chose T%d, file tabs %v", a.globalOf(layout.ViewTasks).Show, a.selectedTask, fileTabs(a, notes))
	}
}

func TestAllTasksOpensOnce(t *testing.T) {
	a, tt, _ := tasksApp(t)
	count := func() int {
		n := 0
		for _, r := range a.perspectiveTabRegions() {
			for _, v := range r.Views {
				if v.Kind == layout.ViewAllTasks {
					n++
				}
			}
		}
		return n
	}
	a.openAllTasks()
	tt.Frame()
	a.openAllTasks()
	tt.Frame()
	if count() != 1 || !tt.HasText("Write the notes") {
		t.Fatalf("%d All Tasks tabs; shows %v", count(), tt.Texts())
	}
}
