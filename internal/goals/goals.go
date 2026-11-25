// Package goals keeps what the user means to get done: goals, a tree of
// what and why, and the tasks they break into, each followed by one role.
// It knows nothing of time beyond the moments it is given: a period or a
// due date is the user's planning, for them to read.
package goals

import (
	"fmt"
	"slices"
	"time"
)

const (
	MaxTitle   = 200  // the bytes of a title, one line
	MaxOutcome = 500  // the bytes of an outcome, one line
	MaxText    = 4096 // the bytes of a text; longer goes in a file it links to
)

type GoalState string

const (
	GoalOpen    GoalState = "open"
	GoalDone    GoalState = "done"
	GoalDropped GoalState = "dropped"
)

type TaskState string

const (
	TaskTodo    TaskState = "todo"
	TaskDoing   TaskState = "doing"
	TaskDone    TaskState = "done"
	TaskDropped TaskState = "dropped"
)

// Priority says how much a task matters beside the others, whatever
// their goals: P0 most. P2, the default, is kept as none.
type Priority string

const (
	P0 Priority = "P0" // urgent
	P1 Priority = "P1" // important
	P2 Priority = ""   // as the rest
)

var (
	GoalStates = []GoalState{GoalOpen, GoalDone, GoalDropped}
	TaskStates = []TaskState{TaskTodo, TaskDoing, TaskDone, TaskDropped}
	Priorities = []Priority{P0, P1, P2}
)

// Goal is something the user means to get done. Its parent is the goal
// it serves, 0 for none; Outcome tells when it is done.
type Goal struct {
	ID      int        `json:"id"`
	Parent  int        `json:"parent,omitempty"`
	Title   string     `json:"title"`
	Outcome string     `json:"outcome"`
	Text    string     `json:"text,omitempty"`
	Link    string     `json:"link,omitempty"`   // file:<absolute path> or project:<name>
	Due     string     `json:"due,omitempty"`    // 2006-01-02
	Period  string     `json:"period,omitempty"` // 2026-11, 2026-W48 or 2026-11-23
	State   GoalState  `json:"state"`
	Created time.Time  `json:"createdAt"`
	Ended   *time.Time `json:"endedAt,omitempty"` // when it became done or dropped
}

// Task is a piece of a goal, followed by its owner, a role's name.
type Task struct {
	ID       int        `json:"id"`
	Goal     int        `json:"goal"`
	Title    string     `json:"title"`
	Outcome  string     `json:"outcome,omitempty"`
	Text     string     `json:"text,omitempty"`
	Link     string     `json:"link,omitempty"`
	Project  string     `json:"project,omitempty"`
	Owner    string     `json:"owner,omitempty"`
	State    TaskState  `json:"state"`
	Priority Priority   `json:"priority,omitempty"`
	Created  time.Time  `json:"createdAt"`
	Started  *time.Time `json:"startedAt,omitempty"` // when it first became doing
	Ended    *time.Time `json:"endedAt,omitempty"`   // when it became done or dropped
	Sessions []Session  `json:"sessions,omitempty"`  // in the order they first touched it
}

// Session is an agent's session that touched a task, in the role it was
// in then, by name: what to read or resume to follow the task back.
type Session struct {
	Role  string    `json:"role,omitempty"`
	Agent string    `json:"agent"` // claude, codex or pi
	ID    string    `json:"id"`
	Home  string    `json:"home,omitempty"`
	Path  string    `json:"path,omitempty"`
	Dir   string    `json:"dir"`
	At    time.Time `json:"at"`
}

// Book is the goals and tasks. Goals of one parent come in the order of
// their importance, tasks of one goal in the order to do them. The last
// IDs given never go back, so an ID is one item's for good.
type Book struct {
	LastGoal, LastTask int
	Goals              []Goal
	Tasks              []Task
}

// GoalChange is what an update sets: nil leaves a field, "" clears an
// optional one. Rank, from 1, places the goal among its siblings.
type GoalChange struct {
	Title, Outcome, Text, Link, Due, Period *string
	Parent                                  *int
	State                                   *GoalState
	Rank                                    int
}

// TaskChange is what an update sets, as GoalChange.
type TaskChange struct {
	Title, Outcome, Text, Link, Project, Owner *string
	Goal                                       *int
	State                                      *TaskState
	Priority                                   *Priority
	Rank                                       int
}

// GoalFilter picks goals: those not ended, unless States or All says.
type GoalFilter struct {
	States []GoalState
	All    bool
	Match  string // in the title, outcome or text, whatever its case
}

// TaskFilter picks tasks, as GoalFilter; Goal takes that goal's and those
// of the goals under it. The conditions all hold.
type TaskFilter struct {
	States                []TaskState
	All                   bool
	Owner, Project, Match string
	Goal                  int
}

func (b Book) Goal(id int) (Goal, bool) {
	i := slices.IndexFunc(b.Goals, func(g Goal) bool { return g.ID == id })
	if i < 0 {
		return Goal{}, false
	}
	return b.Goals[i], true
}

func (b Book) Task(id int) (Task, bool) {
	i := slices.IndexFunc(b.Tasks, func(t Task) bool { return t.ID == id })
	if i < 0 {
		return Task{}, false
	}
	return b.Tasks[i], true
}

// Chain is the goal and those above it, the top one first.
func (b Book) Chain(goal int) []Goal {
	var chain []Goal
	for g, ok := b.Goal(goal); ok && len(chain) <= len(b.Goals); g, ok = b.Goal(g.Parent) {
		chain = append(chain, g)
	}
	slices.Reverse(chain)
	return chain
}

// AddGoal adds an open goal, the last of its siblings.
func AddGoal(b Book, g Goal, now time.Time) (Book, int, error) {
	g.ID, g.State, g.Created, g.Ended = b.LastGoal+1, GoalOpen, now, nil
	next := b
	next.LastGoal, next.Goals = g.ID, append(slices.Clone(b.Goals), g)
	return checked(b, next, g.ID)
}

// AddTask adds a task to do, the last of its goal's.
func AddTask(b Book, t Task, now time.Time) (Book, int, error) {
	t.ID, t.State, t.Created, t.Started, t.Ended, t.Sessions = b.LastTask+1, TaskTodo, now, nil, nil, nil
	next := b
	next.LastTask, next.Tasks = t.ID, append(slices.Clone(b.Tasks), t)
	return checked(b, next, t.ID)
}

// UpdateGoal changes the goal. One moved to another parent goes last
// among its new siblings, unless Rank places it.
func UpdateGoal(b Book, id int, c GoalChange, now time.Time) (Book, error) {
	i := slices.IndexFunc(b.Goals, func(g Goal) bool { return g.ID == id })
	if i < 0 {
		return b, fmt.Errorf("no goal G%d", id)
	}
	g := b.Goals[i]
	for _, f := range []struct{ to, from *string }{{&g.Title, c.Title}, {&g.Outcome, c.Outcome}, {&g.Text, c.Text},
		{&g.Link, c.Link}, {&g.Due, c.Due}, {&g.Period, c.Period}} {
		set(f.to, f.from)
	}
	moved := c.Parent != nil && *c.Parent != g.Parent
	if c.Parent != nil {
		g.Parent = *c.Parent
	}
	if c.State != nil && *c.State != g.State {
		g.State, g.Ended = *c.State, endedAt(*c.State != GoalOpen, now)
	}
	next := b
	next.Goals = place(b.Goals, i, g, moved, c.Rank, func(o Goal) bool { return o.Parent == g.Parent })
	if err := Check(next); err != nil {
		return b, err
	}
	return next, nil
}

// UpdateTask changes the task, its times following its state: doing first
// starts it, done or dropped ends it, todo starts it over.
func UpdateTask(b Book, id int, c TaskChange, now time.Time) (Book, error) {
	i := slices.IndexFunc(b.Tasks, func(t Task) bool { return t.ID == id })
	if i < 0 {
		return b, fmt.Errorf("no task T%d", id)
	}
	t := b.Tasks[i]
	for _, f := range []struct{ to, from *string }{{&t.Title, c.Title}, {&t.Outcome, c.Outcome}, {&t.Text, c.Text},
		{&t.Link, c.Link}, {&t.Project, c.Project}, {&t.Owner, c.Owner}} {
		set(f.to, f.from)
	}
	moved := c.Goal != nil && *c.Goal != t.Goal
	if c.Goal != nil {
		t.Goal = *c.Goal
	}
	if c.Priority != nil {
		t.Priority = *c.Priority
	}
	if c.State != nil && *c.State != t.State {
		t.State, t.Ended = *c.State, endedAt(*c.State == TaskDone || *c.State == TaskDropped, now)
		if t.State == TaskTodo {
			t.Started = nil
		} else if t.State == TaskDoing && t.Started == nil {
			t.Started = &now
		}
	}
	next := b
	next.Tasks = place(b.Tasks, i, t, moved, c.Rank, func(o Task) bool { return o.Goal == t.Goal })
	if err := Check(next); err != nil {
		return b, err
	}
	return next, nil
}

// Touch records the session as one that touched the task, once.
func Touch(b Book, task int, s Session) Book {
	i := slices.IndexFunc(b.Tasks, func(t Task) bool { return t.ID == task })
	if i < 0 || slices.ContainsFunc(b.Tasks[i].Sessions, func(o Session) bool { return o.Agent == s.Agent && o.ID == s.ID }) {
		return b
	}
	b.Tasks = slices.Clone(b.Tasks)
	b.Tasks[i].Sessions = append(slices.Clone(b.Tasks[i].Sessions), s)
	return b
}

// DeleteGoal takes out a goal nothing is under: any other is dropped
// instead, for its history to stay.
func DeleteGoal(b Book, id int) (Book, error) {
	if _, ok := b.Goal(id); !ok {
		return b, fmt.Errorf("no goal G%d", id)
	}
	if slices.ContainsFunc(b.Goals, func(g Goal) bool { return g.Parent == id }) ||
		slices.ContainsFunc(b.Tasks, func(t Task) bool { return t.Goal == id }) {
		return b, fmt.Errorf("goal G%d has goals or tasks under it: set its state to dropped instead", id)
	}
	b.Goals = slices.DeleteFunc(slices.Clone(b.Goals), func(g Goal) bool { return g.ID == id })
	return b, nil
}

// DeleteTask takes out a task not started: any other is dropped instead.
func DeleteTask(b Book, id int) (Book, error) {
	t, ok := b.Task(id)
	if !ok {
		return b, fmt.Errorf("no task T%d", id)
	}
	if t.State != TaskTodo {
		return b, fmt.Errorf("task T%d is %s: set its state to dropped instead", id, t.State)
	}
	b.Tasks = slices.DeleteFunc(slices.Clone(b.Tasks), func(t Task) bool { return t.ID == id })
	return b, nil
}

// checked is the next book and the new item's ID, or the book as it was.
func checked(b, next Book, id int) (Book, int, error) {
	if err := Check(next); err != nil {
		return b, 0, err
	}
	return next, id, nil
}

func set(to, from *string) {
	if from != nil {
		*to = *from
	}
}

func endedAt(ended bool, now time.Time) *time.Time {
	if !ended {
		return nil
	}
	return &now
}

// place puts the item, changed, back in the list: where it was, or last
// when it moved to other siblings, or at the rank among its siblings.
func place[T any](items []T, i int, item T, moved bool, rank int, sibling func(T) bool) []T {
	items = slices.Clone(items)
	if !moved && rank == 0 {
		items[i] = item
		return items
	}
	items = slices.Delete(items, i, i+1)
	at, seen := len(items), 0
	for j, other := range items {
		if rank > 0 && sibling(other) {
			if seen++; seen == rank {
				at = j
				break
			}
		}
	}
	return slices.Insert(items, at, item)
}
