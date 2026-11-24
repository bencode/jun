package app

import (
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/bencode/jun/internal/goals"
)

// rowsText is the rows as they list: kind, depth, goal and task.
func rowsText(rows []taskRow) []string {
	texts := make([]string, len(rows))
	for i, r := range rows {
		texts[i] = fmt.Sprintf("%s %d G%d T%d", r.Kind, r.Depth, r.Goal.ID, r.Task.ID)
		if r.Count > 0 {
			texts[i] += fmt.Sprintf(" ×%d", r.Count)
		}
		if r.Now {
			texts[i] += " now"
		}
	}
	return texts
}

func TestTaskRows(t *testing.T) {
	now := time.Date(2026, 11, 24, 15, 0, 0, 0, time.Local)
	today, lastMonth := now.Add(-time.Hour), now.AddDate(0, -1, 0)
	b := goals.Book{
		Goals: []goals.Goal{
			{ID: 1, Title: "Month", State: goals.GoalOpen},
			{ID: 3, Title: "Shipped", State: goals.GoalDone, Ended: &lastMonth},
			{ID: 2, Parent: 1, Title: "Week", State: goals.GoalOpen},
		},
		Tasks: []goals.Task{
			{ID: 1, Goal: 1, State: goals.TaskDoing},
			{ID: 2, Goal: 2, State: goals.TaskTodo},
			{ID: 3, Goal: 2, State: goals.TaskDone, Ended: &today},
			{ID: 4, Goal: 2, State: goals.TaskDropped, Ended: &lastMonth},
			{ID: 5, Goal: 3, State: goals.TaskDone, Ended: &lastMonth},
		},
	}
	folded := []string{
		"heading 0 G-1 T0",
		"task 0 G1 T1 now",
		"task 0 G2 T3 now",
		"heading 0 G-2 T0",
		"goal 0 G1 T0",
		"task 1 G1 T1",
		"goal 1 G2 T0",
		"task 2 G2 T2",
		"ended 2 G2 T0 ×2",
		"ended-goals 0 G0 T0 ×1",
	}
	if got := rowsText(taskRows(b, now, taskFolds{})); !slices.Equal(got, folded) {
		t.Fatalf("folded:\n%q\nwant\n%q", got, folded)
	}
	open := slices.Concat(folded[:9], []string{"task 2 G2 T3", "task 2 G2 T4"}, folded[9:], []string{"goal 0 G3 T0", "task 1 G3 T5"})
	if got := rowsText(taskRows(b, now, taskFolds{Ended: map[int]bool{2: true}, EndedGoals: true})); !slices.Equal(got, open) {
		t.Fatalf("open:\n%q\nwant\n%q", got, open)
	}
}

func TestMonthGroups(t *testing.T) {
	at := func(month time.Month) *time.Time {
		when := time.Date(2026, month, 10, 12, 0, 0, 0, time.Local)
		return &when
	}
	tasks := []goals.Task{{ID: 1, Ended: at(10)}, {ID: 2}, {ID: 3, Ended: at(11)}, {ID: 4, Ended: at(10)}, {ID: 5}}
	var got []string
	for _, g := range monthGroups(tasks) {
		ids := make([]int, len(g.Tasks))
		for i, task := range g.Tasks {
			ids[i] = task.ID
		}
		got = append(got, fmt.Sprintf("%q %v", g.Month, ids))
	}
	want := []string{`"" [2 5]`, `"2026-11" [3]`, `"2026-10" [1 4]`}
	if !slices.Equal(got, want) {
		t.Fatalf("groups %v, want %v", got, want)
	}
}
