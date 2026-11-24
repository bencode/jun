package app

import (
	"cmp"
	"slices"
	"time"

	"github.com/bencode/jun/internal/goals"
)

// taskFolds is which folds of Tasks are open; they last while Jun runs.
type taskFolds struct {
	Ended      map[int]bool // goal → its ended tasks show
	EndedGoals bool         // the ended goals show
}

type taskRowKind string

const (
	rowHeading    taskRowKind = "heading"     // Now or Goals
	rowGoal       taskRowKind = "goal"        // an open goal, or an ended one once shown
	rowTask       taskRowKind = "task"        // a task, under its goal or in Now
	rowEnded      taskRowKind = "ended"       // the fold of a goal's ended tasks
	rowEndedGoals taskRowKind = "ended-goals" // the fold of the ended goals
)

// taskRow is a row of Tasks.
type taskRow struct {
	Kind  taskRowKind
	Depth int        // a goal's in the tree; its tasks' and fold's one more
	Goal  goals.Goal // the goal of the row, or of its task or fold
	Task  goals.Task
	Count int  // what a fold holds
	Now   bool // a task in Now, which names its goal
}

// The headings' rows tell their section by their goal's ID, none.
const (
	headingNow   = -1
	headingGoals = -2
)

// taskRows is what Tasks lists, top to bottom: Now, the tasks at work or
// ended today; the open goals, each with its tasks to do and the fold of
// those ended; the fold of the ended goals, each with all its tasks.
func taskRows(b goals.Book, now time.Time, f taskFolds) []taskRow {
	all := goals.SelectTasks(b, goals.TaskFilter{All: true})
	of := func(goal int) []goals.Task {
		return slices.DeleteFunc(slices.Clone(all), func(t goals.Task) bool { return t.Goal != goal })
	}
	task := func(t goals.Task, depth int, current bool) taskRow {
		g, _ := b.Goal(t.Goal)
		return taskRow{Kind: rowTask, Depth: depth, Goal: g, Task: t, Now: current}
	}
	depth := func(g goals.Goal) int { return len(b.Chain(g.ID)) - 1 }
	var rows []taskRow
	current := slices.DeleteFunc(slices.Clone(all), func(t goals.Task) bool { return t.State != goals.TaskDoing && !endedOn(t, now) })
	if len(current) > 0 {
		rows = append(rows, taskRow{Kind: rowHeading, Goal: goals.Goal{ID: headingNow}})
		for _, t := range current {
			rows = append(rows, task(t, 0, true))
		}
	}
	open := goals.SelectGoals(b, goals.GoalFilter{})
	if len(open) > 0 {
		rows = append(rows, taskRow{Kind: rowHeading, Goal: goals.Goal{ID: headingGoals}})
	}
	for _, g := range open {
		d := depth(g)
		rows = append(rows, taskRow{Kind: rowGoal, Depth: d, Goal: g})
		tasks := of(g.ID)
		ended := slices.DeleteFunc(slices.Clone(tasks), func(t goals.Task) bool { return !taskEnded(t) })
		for _, t := range slices.DeleteFunc(tasks, taskEnded) {
			rows = append(rows, task(t, d+1, false))
		}
		if len(ended) == 0 {
			continue
		}
		rows = append(rows, taskRow{Kind: rowEnded, Depth: d + 1, Goal: g, Count: len(ended)})
		if f.Ended[g.ID] {
			for _, t := range ended {
				rows = append(rows, task(t, d+1, false))
			}
		}
	}
	closed := goals.SelectGoals(b, goals.GoalFilter{States: []goals.GoalState{goals.GoalDone, goals.GoalDropped}})
	if len(closed) == 0 {
		return rows
	}
	rows = append(rows, taskRow{Kind: rowEndedGoals, Count: len(closed)})
	if !f.EndedGoals {
		return rows
	}
	for _, g := range closed {
		rows = append(rows, taskRow{Kind: rowGoal, Goal: g})
		for _, t := range of(g.ID) {
			rows = append(rows, task(t, 1, false))
		}
	}
	return rows
}

func taskEnded(t goals.Task) bool { return t.State == goals.TaskDone || t.State == goals.TaskDropped }

// endedOn tells whether the task ended on now's day, where the user is.
func endedOn(t goals.Task, now time.Time) bool {
	return t.Ended != nil && t.Ended.Local().Format(time.DateOnly) == now.Local().Format(time.DateOnly)
}

// monthGroup is tasks of All Tasks that ended in one month, or that have
// not ended.
type monthGroup struct {
	Month string // "2026-11", or "" for not ended
	Tasks []goals.Task
}

// monthGroups puts the tasks not ended first, then those ended by month,
// the latest first; the tasks keep their order in each.
func monthGroups(tasks []goals.Task) []monthGroup {
	month := func(t goals.Task) string {
		if t.Ended == nil {
			return ""
		}
		return t.Ended.Local().Format("2006-01")
	}
	var months []string
	for _, t := range tasks {
		if m := month(t); !slices.Contains(months, m) {
			months = append(months, m)
		}
	}
	slices.SortFunc(months, func(x, y string) int {
		if x == "" || y == "" {
			return len(x) - len(y)
		}
		return cmp.Compare(y, x)
	})
	groups := make([]monthGroup, 0, len(months))
	for _, m := range months {
		groups = append(groups, monthGroup{Month: m, Tasks: slices.DeleteFunc(slices.Clone(tasks), func(t goals.Task) bool { return month(t) != m })})
	}
	return groups
}
