package goals

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

var start = time.Date(2026, 11, 24, 9, 0, 0, 0, time.UTC)

func at(hours int) time.Time { return start.Add(time.Duration(hours) * time.Hour) }

func ptr[T any](v T) *T { return &v }

// must fails the test on an error: must[Book](t)(UpdateTask(…)).
func must[T any](t *testing.T) func(T, error) T {
	return func(v T, err error) T {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
}

func addGoal(t *testing.T, b Book, parent int, title string) (Book, int) {
	t.Helper()
	b, id, err := AddGoal(b, Goal{Parent: parent, Title: title, Outcome: title + " done"}, start)
	if err != nil {
		t.Fatal(err)
	}
	return b, id
}

func addTask(t *testing.T, b Book, goal int, title string) (Book, int) {
	t.Helper()
	b, id, err := AddTask(b, Task{Goal: goal, Title: title}, start)
	if err != nil {
		t.Fatal(err)
	}
	return b, id
}

func taskIDs(tasks []Task) []int {
	ids := []int{}
	for _, t := range tasks {
		ids = append(ids, t.ID)
	}
	return ids
}

func goalIDs(goals []Goal) []int {
	ids := []int{}
	for _, g := range goals {
		ids = append(ids, g.ID)
	}
	return ids
}

// A task's times follow its state: doing first starts it, done or dropped
// ends it, going back reopens it, todo starts it over.
func TestTaskTimesFollowItsState(t *testing.T) {
	b, g := addGoal(t, Book{}, 0, "ship")
	b, id := addTask(t, b, g, "write")
	steps := []struct {
		to             TaskState
		started, ended *time.Time
	}{
		{TaskDoing, ptr(at(1)), nil},
		{TaskDone, ptr(at(1)), ptr(at(2))},
		{TaskDoing, ptr(at(1)), nil}, // reopened: its start stays
		{TaskDropped, ptr(at(1)), ptr(at(4))},
		{TaskTodo, nil, nil},
		{TaskDone, nil, ptr(at(6))}, // done without starting
	}
	for i, s := range steps {
		b = must[Book](t)(UpdateTask(b, id, TaskChange{State: ptr(s.to)}, at(i+1)))
		task, _ := b.Task(id)
		if task.State != s.to || !sameTime(task.Started, s.started) || !sameTime(task.Ended, s.ended) {
			t.Fatalf("step %d to %s: started %v ended %v", i, s.to, task.Started, task.Ended)
		}
	}
	b = must[Book](t)(UpdateGoal(b, g, GoalChange{State: ptr(GoalDone)}, at(9)))
	if goal, _ := b.Goal(g); !sameTime(goal.Ended, ptr(at(9))) {
		t.Fatalf("goal done: %v", goal.Ended)
	}
	b = must[Book](t)(UpdateGoal(b, g, GoalChange{State: ptr(GoalOpen)}, at(10)))
	if goal, _ := b.Goal(g); goal.Ended != nil {
		t.Fatalf("goal reopened: %v", goal.Ended)
	}
}

func sameTime(x, y *time.Time) bool {
	return x == nil && y == nil || x != nil && y != nil && x.Equal(*y)
}

// Rank places an item among its siblings only; one moved to other
// siblings goes last there.
func TestRankPlacesAmongSiblings(t *testing.T) {
	b, g1 := addGoal(t, Book{}, 0, "one")
	b, g2 := addGoal(t, b, 0, "two")
	b, sub := addGoal(t, b, g1, "under one")
	b, g3 := addGoal(t, b, 0, "three")
	b = must[Book](t)(UpdateGoal(b, g3, GoalChange{Rank: 1}, start))
	if got := goalIDs(SelectGoals(b, GoalFilter{})); !slices.Equal(got, []int{g3, g1, sub, g2}) {
		t.Fatalf("three first: %v", got)
	}
	b = must[Book](t)(UpdateGoal(b, g3, GoalChange{Rank: 9}, start))
	if got := goalIDs(SelectGoals(b, GoalFilter{})); !slices.Equal(got, []int{g1, sub, g2, g3}) {
		t.Fatalf("three last: %v", got)
	}

	b, t1 := addTask(t, b, g1, "a")
	b, t2 := addTask(t, b, g2, "b")
	b, t3 := addTask(t, b, g1, "c")
	b = must[Book](t)(UpdateTask(b, t3, TaskChange{Rank: 1}, start))
	b = must[Book](t)(UpdateTask(b, t1, TaskChange{Goal: ptr(g2)}, start))
	if got := taskIDs(SelectTasks(b, TaskFilter{})); !slices.Equal(got, []int{t3, t2, t1}) {
		t.Fatalf("tasks: %v", got)
	}
}

// Only what nothing is under goes; a started task is dropped instead.
func TestDeleteOnlyWhatNothingNeeds(t *testing.T) {
	b, g := addGoal(t, Book{}, 0, "top")
	b, sub := addGoal(t, b, g, "sub")
	b, id := addTask(t, b, sub, "x")
	if _, err := DeleteGoal(b, sub); err == nil {
		t.Fatal("deleted a goal with a task")
	}
	b = must[Book](t)(UpdateTask(b, id, TaskChange{State: ptr(TaskDoing)}, start))
	if _, err := DeleteTask(b, id); err == nil || !strings.Contains(err.Error(), "dropped") {
		t.Fatalf("deleted a started task: %v", err)
	}
	b = must[Book](t)(UpdateTask(b, id, TaskChange{State: ptr(TaskTodo)}, start))
	b = must[Book](t)(DeleteTask(b, id))
	b = must[Book](t)(DeleteGoal(b, sub))
	if _, err := DeleteGoal(b, sub); err == nil {
		t.Fatal("deleted a goal twice")
	}
}

// What would break the book is refused, leaving it as it was.
func TestChangesThatBreakTheBookAreRefused(t *testing.T) {
	b, g := addGoal(t, Book{}, 0, "top")
	b, sub := addGoal(t, b, g, "sub")
	b, id := addTask(t, b, sub, "x")
	refused := map[string]error{
		"cycle":        second(UpdateGoal(b, g, GoalChange{Parent: ptr(sub)}, start)),
		"own parent":   second(UpdateGoal(b, g, GoalChange{Parent: ptr(g)}, start)),
		"no parent":    third(AddGoal(b, Goal{Parent: 99, Title: "x", Outcome: "y"}, start)),
		"no outcome":   third(AddGoal(b, Goal{Title: "x"}, start)),
		"two lines":    second(UpdateTask(b, id, TaskChange{Title: ptr("a\nb")}, start)),
		"no goal":      third(AddTask(b, Task{Goal: 99, Title: "x"}, start)),
		"long text":    second(UpdateTask(b, id, TaskChange{Text: ptr(strings.Repeat("x", MaxText+1))}, start)),
		"link":         second(UpdateTask(b, id, TaskChange{Link: ptr("terminal:7")}, start)),
		"relative":     second(UpdateTask(b, id, TaskChange{Link: ptr("file:a.md")}, start)),
		"due":          second(UpdateGoal(b, g, GoalChange{Due: ptr("11-28")}, start)),
		"week 53":      second(UpdateGoal(b, g, GoalChange{Period: ptr("2025-W53")}, start)),
		"state":        second(UpdateTask(b, id, TaskChange{State: ptr(TaskState("later"))}, start)),
		"missing task": second(UpdateTask(b, 99, TaskChange{}, start)),
	}
	for name, err := range refused {
		if err == nil {
			t.Errorf("%s: not refused", name)
		}
	}
	for _, period := range []string{"2026-11", "2026-W53", "2026-W48", "2026-11-23"} {
		if _, err := UpdateGoal(b, g, GoalChange{Period: ptr(period)}, start); err != nil {
			t.Errorf("period %s: %v", period, err)
		}
	}
	if err := Check(b); err != nil || len(b.Goals) != 2 || len(b.Tasks) != 1 {
		t.Fatalf("book changed: %v %+v", err, b)
	}
}

func second[T any](_ T, err error) error        { return err }
func third[T, U any](_ T, _ U, err error) error { return err }

// Filters pick unended items by default, a goal's tree, and words in any
// case, all together.
func TestFiltersPickTogether(t *testing.T) {
	b, g := addGoal(t, Book{}, 0, "top")
	b, sub := addGoal(t, b, g, "sub")
	b, other := addGoal(t, b, 0, "other")
	b, t1 := addTask(t, b, sub, "Parser fix")
	b, t2 := addTask(t, b, other, "parser docs")
	b, t3 := addTask(t, b, g, "release")
	b = must[Book](t)(UpdateTask(b, t1, TaskChange{Owner: ptr("高斯"), Project: ptr("jun")}, start))
	b = must[Book](t)(UpdateTask(b, t3, TaskChange{State: ptr(TaskDone)}, start))
	b = must[Book](t)(UpdateGoal(b, other, GoalChange{State: ptr(GoalDropped)}, start))
	cases := []struct {
		filter TaskFilter
		want   []int
	}{
		{TaskFilter{}, []int{t1, t2}},
		{TaskFilter{All: true}, []int{t3, t1, t2}},
		{TaskFilter{States: []TaskState{TaskDone}}, []int{t3}},
		{TaskFilter{Goal: g, All: true}, []int{t3, t1}},
		{TaskFilter{Match: "PARSER"}, []int{t1, t2}},
		{TaskFilter{Match: "parser", Owner: "高斯", Project: "JUN"}, []int{t1}},
	}
	for _, c := range cases {
		if got := taskIDs(SelectTasks(b, c.filter)); !slices.Equal(got, c.want) {
			t.Errorf("%+v: %v, want %v", c.filter, got, c.want)
		}
	}
	if got := goalIDs(SelectGoals(b, GoalFilter{})); !slices.Equal(got, []int{g, sub}) {
		t.Errorf("open goals: %v", got)
	}
}

// A book saved reads back the same; an ID once given is never given
// again, even after the item is gone.
func TestBookKeepsAndNeverReusesIDs(t *testing.T) {
	path := filepath.Join(t.TempDir(), "goals.json")
	if b, err := Load(path); err != nil || b.LastGoal != 0 {
		t.Fatalf("no file: %+v %v", b, err)
	}
	b, g := addGoal(t, Book{}, 0, "top")
	b, id := addTask(t, b, g, "x")
	b = must[Book](t)(UpdateTask(b, id, TaskChange{State: ptr(TaskDoing)}, at(1)))
	b = Touch(b, id, Session{Role: "高斯", Agent: "claude", ID: "s1", Dir: "/w", At: at(1)})
	b = Touch(b, id, Session{Role: "高斯", Agent: "claude", ID: "s1", Dir: "/w", At: at(2)})
	if task, _ := b.Task(id); len(task.Sessions) != 1 {
		t.Fatalf("session touched twice: %+v", task.Sessions)
	}
	if err := Save(path, b); err != nil {
		t.Fatal(err)
	}
	read := must[Book](t)(Load(path))
	task, _ := read.Task(id)
	if read.LastGoal != 1 || read.LastTask != 1 || task.Sessions[0].ID != "s1" || !task.Started.Equal(at(1)) {
		t.Fatalf("read back: %+v", read)
	}

	b = must[Book](t)(UpdateTask(b, id, TaskChange{State: ptr(TaskTodo)}, start))
	b = must[Book](t)(DeleteTask(b, id))
	b = must[Book](t)(DeleteGoal(b, g))
	if err := Save(path, b); err != nil {
		t.Fatal(err)
	}
	read = must[Book](t)(Load(path))
	read, g2 := addGoal(t, read, 0, "next")
	read, t2 := addTask(t, read, g2, "y")
	if g2 != 2 || t2 != 2 {
		t.Fatalf("IDs given again: G%d T%d", g2, t2)
	}

	bad := `{"version": 1, "lastGoal": 2, "lastTask": 0, "goals": [` +
		`{"id":1,"parent":2,"title":"a","outcome":"b","state":"open","createdAt":"2026-11-24T09:00:00Z"},` +
		`{"id":2,"parent":1,"title":"c","outcome":"d","state":"open","createdAt":"2026-11-24T09:00:00Z"}], "tasks": []}`
	if err := os.WriteFile(path, []byte(bad), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("loaded goals in a cycle")
	}
}
