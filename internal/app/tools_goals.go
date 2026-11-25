package app

import (
	"fmt"
	"log"
	"slices"
	"strings"
	"time"

	"github.com/bencode/jun/internal/goals"
	"github.com/bencode/jun/internal/history"
	"github.com/bencode/jun/internal/terminal"
	"github.com/bencode/jun/internal/tool"
)

// restoreGoals reads the goals kept beside the configuration. Ones that
// cannot be read are not written either: changes then last while Jun runs.
func (a *app) restoreGoals(configPath string) {
	path := goals.Path(configPath)
	book, err := goals.Load(path)
	if err != nil {
		a.report(fmt.Errorf("%s: %w", a.text("Goals could not be loaded; they are not kept for this session",
			"目标无法加载；本次运行不保存目标"), err))
		return
	}
	a.goals, a.goalsPath = book, path
}

// setGoals keeps the book: the file is replaced whole, so one not written
// leaves the goals as they were.
func (a *app) setGoals(next goals.Book) error {
	if a.goalsPath != "" {
		if err := goals.Save(a.goalsPath, next); err != nil {
			return err
		}
	}
	a.goals = next
	return nil
}

func (a *app) toolGoals(req tool.Request) []tool.Goal {
	f := goals.GoalFilter{States: states[goals.GoalState](req.Filter.States), All: req.Filter.All, Match: req.Filter.Match}
	return mapped(goals.SelectGoals(a.goals, f), toolGoal)
}

// toolGoalDetail is the goal with the goals above it, and its own tasks.
func (a *app) toolGoalDetail(req tool.Request) ([]tool.Goal, []tool.Task, error) {
	if _, ok := a.goals.Goal(req.Item.ID); !ok {
		return nil, nil, fmt.Errorf("no goal G%d", req.Item.ID)
	}
	own := slices.DeleteFunc(slices.Clone(a.goals.Tasks), func(t goals.Task) bool { return t.Goal != req.Item.ID })
	return mapped(a.goals.Chain(req.Item.ID), toolGoal), mapped(own, func(t goals.Task) tool.Task { return a.toolTask(t, false) }), nil
}

func (a *app) toolTasks(req tool.Request) ([]tool.Task, error) {
	f := req.Filter
	if _, ok := a.goals.Goal(f.Goal); f.Goal != 0 && !ok {
		return nil, fmt.Errorf("no goal G%d", f.Goal)
	}
	picked := goals.SelectTasks(a.goals, goals.TaskFilter{States: states[goals.TaskState](f.States), All: f.All,
		Owner: f.Owner, Project: f.Project, Match: f.Match, Goal: f.Goal})
	return mapped(picked, func(t goals.Task) tool.Task { return a.toolTask(t, false) }), nil
}

// toolTaskDetail is the task, whole, with its goal and those above it.
func (a *app) toolTaskDetail(req tool.Request) ([]tool.Goal, []tool.Task, error) {
	t, ok := a.goals.Task(req.Item.ID)
	if !ok {
		return nil, nil, fmt.Errorf("no task T%d", req.Item.ID)
	}
	return mapped(a.goals.Chain(t.Goal), toolGoal), []tool.Task{a.toolTask(t, true)}, nil
}

func (a *app) toolAddGoal(req tool.Request, callers []int) (int, error) {
	if _, err := a.toolCaller(callers, req.Caller); err != nil {
		return 0, err
	}
	it := req.Item
	link, err := a.itemLink(it.Link)
	if err != nil {
		return 0, err
	}
	g := goals.Goal{Parent: value(it.Parent), Title: value(it.Title), Outcome: value(it.Outcome), Text: value(it.Text),
		Link: value(link), Due: value(it.Due), Period: value(it.Period)}
	next, id, err := goals.AddGoal(a.goals, g, toolNow())
	if err != nil {
		return 0, err
	}
	return id, a.setGoals(next)
}

func (a *app) toolUpdateGoal(req tool.Request, callers []int) error {
	if _, err := a.toolCaller(callers, req.Caller); err != nil {
		return err
	}
	it := req.Item
	link, err := a.itemLink(it.Link)
	if err != nil {
		return err
	}
	c := goals.GoalChange{Title: it.Title, Outcome: it.Outcome, Text: it.Text, Link: link, Due: it.Due, Period: it.Period,
		Parent: it.Parent, State: state[goals.GoalState](it.State), Rank: it.Rank}
	next, err := goals.UpdateGoal(a.goals, it.ID, c, toolNow())
	if err != nil {
		return err
	}
	return a.setGoals(next)
}

func (a *app) toolDeleteGoal(req tool.Request, callers []int) error {
	if _, err := a.toolCaller(callers, req.Caller); err != nil {
		return err
	}
	next, err := goals.DeleteGoal(a.goals, req.Item.ID)
	if err != nil {
		return err
	}
	return a.setGoals(next)
}

// toolAddTask adds the task, in the caller's project unless it says, and
// records the caller's agent session as one that touched it.
func (a *app) toolAddTask(req tool.Request, callers []int) (int, error) {
	caller, err := a.toolCaller(callers, req.Caller)
	if err != nil {
		return 0, err
	}
	it := req.Item
	if it.Project == nil {
		own := a.terminals.Get(caller).ProjectName
		it.Project = &own
	}
	fields, err := a.taskFields(it)
	if err != nil {
		return 0, err
	}
	t := goals.Task{Goal: value(it.Goal), Title: value(it.Title), Outcome: value(it.Outcome), Text: value(it.Text),
		Link: value(fields.Link), Project: value(fields.Project), Owner: value(fields.Owner), Priority: value(priority(it.Priority))}
	next, id, err := goals.AddTask(a.goals, t, toolNow())
	if err != nil {
		return 0, err
	}
	if next, err = a.touchTask(next, id, req, callers, caller); err != nil {
		return 0, err
	}
	return id, a.setGoals(next)
}

func (a *app) toolUpdateTask(req tool.Request, callers []int) error {
	caller, err := a.toolCaller(callers, req.Caller)
	if err != nil {
		return err
	}
	it := req.Item
	fields, err := a.taskFields(it)
	if err != nil {
		return err
	}
	c := goals.TaskChange{Title: it.Title, Outcome: it.Outcome, Text: it.Text, Link: fields.Link, Project: fields.Project,
		Owner: fields.Owner, Goal: it.Goal, State: state[goals.TaskState](it.State), Priority: priority(it.Priority), Rank: it.Rank}
	next, err := goals.UpdateTask(a.goals, it.ID, c, toolNow())
	if err != nil {
		return err
	}
	if next, err = a.touchTask(next, it.ID, req, callers, caller); err != nil {
		return err
	}
	return a.setGoals(next)
}

func (a *app) toolDeleteTask(req tool.Request, callers []int) error {
	if _, err := a.toolCaller(callers, req.Caller); err != nil {
		return err
	}
	next, err := goals.DeleteTask(a.goals, req.Item.ID)
	if err != nil {
		return err
	}
	return a.setGoals(next)
}

// taskFields is a task's link, project and owner as Jun keeps them: the
// names as their projects and roles have them. Nil stays nil, "" clears.
func (a *app) taskFields(it *tool.Item) (tool.Item, error) {
	link, err := a.itemLink(it.Link)
	if err != nil {
		return tool.Item{}, err
	}
	fields := tool.Item{Link: link, Project: it.Project, Owner: it.Owner}
	if value(it.Project) != "" {
		p, err := namedProject(a, tool.Target{Kind: tool.TargetProject, Name: *it.Project}, nil)
		if err != nil {
			return tool.Item{}, err
		}
		fields.Project = &p.Name
	}
	if value(it.Owner) != "" {
		r, ok := a.roles.Named(*it.Owner)
		if !ok {
			return tool.Item{}, fmt.Errorf("no role %q: an owner is a role's name, as roles lists them", *it.Owner)
		}
		fields.Owner = &r.Name
	}
	return fields, nil
}

// itemLink checks that the link leads somewhere: a file there, or a
// project, by its name.
func (a *app) itemLink(link *string) (*string, error) {
	if value(link) == "" {
		return link, nil
	}
	if path, ok := strings.CutPrefix(*link, "file:"); ok {
		return link, a.checkFile(path)
	}
	name, ok := strings.CutPrefix(*link, "project:")
	if !ok {
		return nil, fmt.Errorf("link %q is neither file:<path> nor project:<name>", *link)
	}
	p, err := namedProject(a, tool.Target{Kind: tool.TargetProject, Name: name}, nil)
	if err != nil {
		return nil, err
	}
	canonical := tool.Target{Kind: tool.TargetProject, Name: p.Name}.String()
	return &canonical, nil
}

// touchTask records the caller's agent session, in its role, as one that
// touched the task; a caller running no agent session is none.
func (a *app) touchTask(b goals.Book, id int, req tool.Request, callers []int, caller terminal.ID) (goals.Book, error) {
	own, ok, err := a.ownSession(a.terminals.Get(caller), req, callers)
	if err != nil || !ok {
		return b, err
	}
	role, _ := a.roles.Of(own.Agent, own.ID)
	return goals.Touch(b, id, goals.Session{Role: role.Name, Agent: own.Agent, ID: own.ID, Home: own.Home, Path: own.Path,
		Dir: own.Dir, At: own.At}), nil
}

func toolGoal(g goals.Goal) tool.Goal {
	return tool.Goal{ID: g.ID, Parent: g.Parent, Title: g.Title, Outcome: g.Outcome, Text: g.Text, Link: g.Link, Due: g.Due,
		Period: g.Period, State: string(g.State), Created: g.Created, Ended: g.Ended}
}

// toolTask is the task as task gives it: whole, with its text and its
// sessions and how to resume them, or as a list has it.
func (a *app) toolTask(t goals.Task, whole bool) tool.Task {
	item := tool.Task{ID: t.ID, Goal: t.Goal, Title: t.Title, Outcome: t.Outcome, Link: t.Link, Project: t.Project, Owner: t.Owner,
		State: string(t.State), Priority: string(t.Priority), Created: t.Created, Started: t.Started, Ended: t.Ended}
	if !whole {
		return item
	}
	item.Text = t.Text
	for _, s := range t.Sessions {
		resume, err := history.ResumeCommand(history.Session{Agent: s.Agent, ID: s.ID, Path: s.Path, Home: s.Home}, a.resumeCommands())
		if err != nil {
			log.Printf("Jun: resume of the %s session %s: %v", s.Agent, s.ID, err)
		}
		item.Sessions = append(item.Sessions, tool.TaskSession{Role: s.Role, Agent: s.Agent, ID: s.ID, Home: s.Home, Dir: s.Dir,
			At: s.At, Resume: resume})
	}
	return item
}

// toolNow is when a change is made, to the second, as roles keep it.
func toolNow() time.Time { return time.Now().Truncate(time.Second) }

func value[T any](p *T) T {
	var zero T
	if p == nil {
		return zero
	}
	return *p
}

func state[S ~string](s *string) *S {
	if s == nil {
		return nil
	}
	v := S(*s)
	return &v
}

// priority is the priority an option names, P2 kept as none.
func priority(name *string) *goals.Priority {
	if name == nil {
		return nil
	}
	p := goals.Priority(*name)
	if *name == "P2" {
		p = goals.P2
	}
	return &p
}

func states[S ~string](list []string) []S { return mapped(list, func(s string) S { return S(s) }) }

func mapped[T, U any](items []T, f func(T) U) []U {
	out := make([]U, 0, len(items))
	for _, item := range items {
		out = append(out, f(item))
	}
	return out
}
