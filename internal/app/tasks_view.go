package app

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/bencode/jun/internal/goals"
	"github.com/bencode/jun/internal/layout"
	"github.com/bencode/jun/internal/terminal"
	"github.com/bencode/jun/internal/theme"
	"github.com/bencode/jun/internal/tool"
	"github.com/egoist/mygo/ui"
)

// tasksView lists the goals and their tasks, Now first. Agents change
// them with jun tool; the view only reads them.
func (a *app) tasksView(c *ui.Context) {
	c.After(time.Minute) // for Now to drop what ended yesterday
	rows := taskRows(a.goals, time.Now(), a.taskFolds)
	ui.Column(c.Key("tasks")).Fill().Children(func() {
		a.tasksBar(c)
		if len(rows) == 0 {
			ui.Column(c).Grow(1).FillWidth().AlignItems(ui.Center).Justify(ui.Center).Padding(12).Children(func() {
				ui.Text(c, a.text("Agents add goals and tasks here, with jun tool goal and task.", "agent 用 jun tool goal 和 task 建目标和任务。")).
					TextColor(a.theme.TextMuted)
			})
			return
		}
		a.taskList(c, rows)
	})
}

// tasksBar counts the tasks at work and to do, and opens All Tasks.
func (a *app) tasksBar(c *ui.Context) {
	count := func(s goals.TaskState) int {
		return len(goals.SelectTasks(a.goals, goals.TaskFilter{States: []goals.TaskState{s}}))
	}
	ui.Row(c).FillWidth().Height(32).Shrink(0).Padding(0, 6, 0, 12).Gap(8).AlignItems(ui.Center).Children(func() {
		ui.Text(c, fmt.Sprintf(a.text("%d doing · %d to do", "进行中 %d · 待办 %d"), count(goals.TaskDoing), count(goals.TaskTodo))).
			FontSize(theme.Caption).TextColor(a.theme.TextMuted).SingleLine().Ellipsis("…").Grow(1).MinWidth(0)
		ui.Button(c, a.kindName(layout.ViewAllTasks)).Height(24).FontSize(theme.Caption).Shrink(0).OnClick(a.openAllTasks)
	})
}

// taskList keeps the keyboard on the list, as messageList does: ↩ opens
// the task chosen, → and ← open and close folds.
func (a *app) taskList(c *ui.Context, rows []taskRow) {
	st := &a.tasksList
	if st.Selected == nil {
		a.taskRowSel = -1
		st.Selected = &a.taskRowSel
	}
	st.Key = func(i int) any { return taskRowKey(rows[i]) }
	st.Label = func(i int) string { return a.taskRowLabel(rows[i]) }
	clicked := false
	list := ui.List(c, st, len(rows), func(i int) {
		if a.taskRowView(c, rows[i], i == a.taskRowSel) {
			a.taskRowSel, clicked = i, true
		}
	}).Gap(2).Grow(1).FillWidth().Padding(0, 6, 6, 6)
	if clicked {
		list.Focus()
		c.Invalidate()
	}
	if a.taskRowSel < 0 || a.taskRowSel >= len(rows) {
		a.selectedTask = 0
		return
	}
	r := rows[a.taskRowSel]
	a.selectedTask = 0
	if r.Kind == rowTask {
		a.selectedTask = r.Task.ID
	}
	if clicked && r.Kind != rowTask {
		a.toggleFold(r, !a.foldOpen(r))
	}
	if list.Submitted() && r.Kind == rowTask {
		a.openTask(r.Task)
		c.Invalidate()
	}
	for key, open := range map[ui.Key]bool{ui.KeyRight: true, ui.KeyLeft: false} {
		if list.Shortcut(0, key) {
			a.toggleFold(r, open)
			c.Invalidate()
		}
	}
}

// taskRowKey tells the rows apart, a task by where it shows: in Now and
// under its goal it is two rows.
func taskRowKey(r taskRow) string {
	return fmt.Sprintf("%s %d %d %v", r.Kind, r.Goal.ID, r.Task.ID, r.Now)
}

// foldOpen tells whether the fold the row opens is open: a goal's is that
// of its ended tasks.
func (a *app) foldOpen(r taskRow) bool {
	if r.Kind == rowEndedGoals {
		return a.taskFolds.EndedGoals
	}
	return a.taskFolds.Ended[r.Goal.ID]
}

func (a *app) toggleFold(r taskRow, open bool) {
	if r.Kind == rowEndedGoals {
		a.taskFolds.EndedGoals = open
		return
	}
	if r.Kind != rowEnded && r.Kind != rowGoal {
		return
	}
	if a.taskFolds.Ended == nil {
		a.taskFolds.Ended = map[int]bool{}
	}
	a.taskFolds.Ended[r.Goal.ID] = open
}

// taskRowView draws the row and reports a click on it.
func (a *app) taskRowView(c *ui.Context, r taskRow, chosen bool) bool {
	muted := a.messageMuted(chosen)
	row := ui.Column(c).FillWidth().Padding(4, 6, 4, 6+float32(r.Depth)*memberIndent).Radius(4).Gap(2)
	row.Children(func() {
		if r.Kind == rowHeading {
			ui.Text(c, a.taskHeading(r)).FontSize(theme.Caption).TextColor(muted)
			return
		}
		if r.Kind == rowEnded || r.Kind == rowEndedGoals {
			mark := map[bool]string{false: " ▸", true: " ▾"}[a.foldOpen(r)]
			label := a.text("Ended %d", "已结束 %d")
			if r.Kind == rowEndedGoals {
				label = a.text("Ended goals %d", "已结束的目标 %d")
			}
			ui.Text(c, fmt.Sprintf(label, r.Count)+mark).FontSize(theme.Caption).TextColor(muted)
			return
		}
		ui.Row(c).FillWidth().Gap(messageGap).AlignItems(ui.Center).Children(func() {
			if r.Kind == rowGoal {
				a.goalRowContent(c, r.Goal, chosen)
				return
			}
			a.taskRowContent(c, r.Task, chosen)
		})
		if r.Now {
			ui.Text(c, r.Goal.Title).FontSize(theme.Caption).TextColor(muted).SingleLine().Ellipsis("…").Padding(0, 0, 0, a.theme.Rem(1)+messageGap)
		}
		if when := a.goalWhen(r.Goal); r.Kind == rowGoal && when != "" {
			ui.Text(c, when).FontSize(theme.Caption).TextColor(muted).SingleLine().Ellipsis("…")
		}
	})
	return row.Clicked()
}

func (a *app) taskHeading(r taskRow) string {
	if r.Goal.ID == headingNow {
		return a.text("Now", "当前")
	}
	return a.text("Goals", "目标")
}

func (a *app) goalRowContent(c *ui.Context, g goals.Goal, chosen bool) {
	muted := a.messageMuted(chosen)
	ui.Text(c, fmt.Sprintf("G%d", g.ID)).FontSize(theme.Caption).TextColor(muted).MinWidth(numberWidth(c, "G", a.goals.LastGoal)).Shrink(0)
	title := ui.Text(c, g.Title).FontWeight(600).SingleLine().Ellipsis("…").Grow(1).MinWidth(0)
	if g.State != goals.GoalOpen {
		title.TextColor(muted)
	}
}

// goalWhen is the goal's period and due date, under its title for the
// title to keep the width.
func (a *app) goalWhen(g goals.Goal) string {
	due := ""
	if g.Due != "" {
		due = a.text("due ", "截止 ") + g.Due
	}
	return strings.Join(slices.DeleteFunc([]string{g.Period, due}, func(s string) bool { return s == "" }), " · ")
}

func (a *app) taskRowContent(c *ui.Context, t goals.Task, chosen bool) {
	muted := a.messageMuted(chosen)
	ui.Text(c, taskStateMark[t.State]).TextColor(a.taskStateColor(t.State, chosen)).Width(a.theme.Rem(1)).Shrink(0)
	ui.Text(c, taskNumber(t.ID)).FontSize(theme.Caption).TextColor(muted).MinWidth(numberWidth(c, "T", a.goals.LastTask)).Shrink(0)
	title := ui.Text(c, t.Title).SingleLine().Ellipsis("…").Grow(1).MinWidth(0)
	if taskEnded(t) && !chosen {
		title.TextColor(a.theme.TextMuted)
	}
	if t.Priority != goals.P2 {
		ui.Text(c, string(t.Priority)).FontSize(theme.Caption).Bold().TextColor(a.priorityColor(t, chosen)).Shrink(0)
	}
	if n := a.openAsks(t.ID); n > 0 {
		ui.Text(c, fmt.Sprintf("ask %d", n)).FontSize(theme.Caption).Bold().TextColor(a.askColor(chosen)).Shrink(0)
	}
	if t.Owner != "" {
		ui.Text(c, t.Owner).FontSize(theme.Caption).TextColor(muted).SingleLine().Shrink(0)
	}
}

// numberWidth is the width of a number with as many digits as the last
// given, all 0s, one of the widest digits, for the titles after the
// numbers to line up where a font's digits differ in width.
func numberWidth(c *ui.Context, prefix string, last int) float32 {
	widest := prefix + strings.Repeat("0", len(strconv.Itoa(last)))
	w, _ := c.MeasureText(0, ui.Span{Text: widest, Size: theme.Caption})
	return w
}

var taskStateMark = map[goals.TaskState]string{goals.TaskTodo: "○", goals.TaskDoing: "◐", goals.TaskDone: "✓", goals.TaskDropped: "×"}

func (a *app) taskStateColor(s goals.TaskState, chosen bool) ui.Color {
	if chosen {
		return a.theme.AccentText
	}
	return map[goals.TaskState]ui.Color{goals.TaskTodo: a.theme.TextMuted, goals.TaskDoing: a.theme.Accent,
		goals.TaskDone: a.theme.Success, goals.TaskDropped: a.theme.TextMuted}[s]
}

// priorityColor marks P0 as a danger and P1 as a warning, while the task
// is open.
func (a *app) priorityColor(t goals.Task, chosen bool) ui.Color {
	if chosen {
		return a.theme.AccentText
	}
	if taskEnded(t) {
		return a.theme.TextMuted
	}
	if t.Priority == goals.P0 {
		return a.theme.Danger
	}
	return a.theme.Warning
}

func (a *app) askColor(chosen bool) ui.Color {
	if chosen {
		return a.theme.AccentText
	}
	return a.theme.Accent
}

// taskRowLabel is the row for assistive technology.
func (a *app) taskRowLabel(r taskRow) string {
	if r.Kind == rowHeading {
		return a.taskHeading(r)
	}
	if r.Kind == rowGoal {
		return fmt.Sprintf("G%d %s", r.Goal.ID, r.Goal.Title)
	}
	if r.Kind == rowTask {
		label := fmt.Sprintf("%s %s, %s", taskNumber(r.Task.ID), r.Task.Title, r.Task.State)
		if r.Task.Priority != goals.P2 {
			label += ", " + string(r.Task.Priority)
		}
		return label
	}
	return fmt.Sprintf(a.text("Ended %d", "已结束 %d"), r.Count)
}

// openAsks counts the messages about the task that still ask the user.
func (a *app) openAsks(task int) int {
	n := 0
	for _, m := range a.messages {
		if m.Task == task && m.Kind == tool.MessageAsk {
			n++
		}
	}
	return n
}

// revealTask shows the task in Tasks, chosen: Global and Tasks show, and the
// folds it is under open.
func (a *app) revealTask(id int) {
	t, ok := a.goals.Task(id)
	if !ok {
		a.report(fmt.Errorf(a.text("There is no task T%d", "没有任务 T%d"), id))
		return
	}
	if g, ok := a.goals.Goal(t.Goal); ok && g.State != goals.GoalOpen {
		a.taskFolds.EndedGoals = true
	}
	if taskEnded(t) {
		a.toggleFold(taskRow{Kind: rowEnded, Goal: goals.Goal{ID: t.Goal}}, true)
	}
	v := a.globalOf(layout.ViewTasks)
	if !a.state.GlobalVisible || !v.Show {
		a.state.GlobalVisible, v.Show = true, true
		a.globalChanged()
	}
	rows := taskRows(a.goals, time.Now(), a.taskFolds)
	i := slices.IndexFunc(rows, func(r taskRow) bool { return r.Kind == rowTask && !r.Now && r.Task.ID == id })
	a.taskRowSel, a.selectedTask = i, id
	a.tasksList.Selected = &a.taskRowSel // Tasks may not have shown yet
	a.tasksList.ScrollIntoView(i)
}

// globalOf is Global's view of the kind, which its preset always has.
func (a *app) globalOf(kind layout.ViewKind) *layout.View {
	return a.state.Global.Views[slices.IndexFunc(a.state.Global.Views, func(v *layout.View) bool { return v.Kind == kind })]
}

// openTask takes the user to what the task links to, else peeks the agent
// following it.
func (a *app) openTask(t goals.Task) {
	if t.Link != "" {
		target, ok := taskTarget(t.Link)
		if !ok {
			a.report(fmt.Errorf("%s links to %q, which is neither a file nor a project", taskNumber(t.ID), t.Link))
			return
		}
		link := messageLinks[target.Kind]
		p, err := link.project(a, target, a.project(t.Project))
		if err == nil && p == nil {
			err = fmt.Errorf(a.text("%s links to a file of no project", "%s 链接的文件不属于任何项目"), taskNumber(t.ID))
		}
		if err != nil {
			a.report(err)
			return
		}
		a.report(link.show(a, target, p, false))
		return
	}
	if id, ok := a.ownerTerminal(t.Owner); ok {
		a.peekAgent(id)
		return
	}
	a.report(fmt.Errorf(a.text("%s links to nothing, and no agent named %q runs", "%s 没有链接，也没有名为 %q 的 agent 在运行"), taskNumber(t.ID), t.Owner))
}

// taskTarget is the resource of a task's link, as goals.json keeps it.
func taskTarget(link string) (tool.Target, bool) {
	for _, kind := range []tool.TargetKind{tool.TargetFile, tool.TargetProject} {
		if name, ok := strings.CutPrefix(link, string(kind)+":"); ok {
			return tool.Target{Kind: kind, Name: name}, true
		}
	}
	return tool.Target{}, false
}

// taskNumber is what the user and agents call the task.
func taskNumber(id int) string { return fmt.Sprintf("T%d", id) }

// ownerTerminal is the terminal of the agent that has the role's name.
func (a *app) ownerTerminal(owner string) (terminal.ID, bool) {
	if owner == "" {
		return 0, false
	}
	for id, name := range a.names {
		if _, isAgent := a.agents[id]; isAgent && strings.EqualFold(name, owner) {
			return id, true
		}
	}
	return 0, false
}

// taskOwnerChosen tells whether the terminal's agent follows the task
// chosen in Tasks.
func (a *app) taskOwnerChosen(id terminal.ID) bool {
	t, ok := a.goals.Task(a.selectedTask)
	return ok && t.Owner != "" && strings.EqualFold(a.names[id], t.Owner)
}

// taskTint is how a row about the task chosen in Tasks shows.
func (a *app) taskTint() ui.Color { return a.theme.Accent.Alpha(0.12) }
