package app

import (
	"cmp"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/bencode/jun/internal/goals"
	"github.com/bencode/jun/internal/layout"
	"github.com/bencode/jun/internal/theme"
	"github.com/egoist/mygo/ui"
)

// allTasksState is what an All Tasks tab keeps while Jun runs.
type allTasksState struct {
	query          string
	states         []goals.TaskState // none: all
	owner, project string            // "": any
	open           map[string]bool   // the month groups shown; "" is those not ended
	list           ui.ListState
	selected       int
}

func (a *app) allTasksOf(view layout.ViewID) *allTasksState {
	if a.allTasks == nil {
		a.allTasks = map[layout.ViewID]*allTasksState{}
	}
	st := a.allTasks[view]
	if st == nil {
		st = &allTasksState{open: map[string]bool{"": true}, selected: -1}
		st.list.Selected = &st.selected
		a.allTasks[view] = st
	}
	return st
}

// openAllTasks shows the current perspective's All Tasks tab, else opens
// one where new tabs go, as openSettings does.
func (a *app) openAllTasks() {
	project, target := a.state.CurrentProject(), a.tabsTarget()
	if project == nil || target == nil {
		a.report(errors.New(a.text("Open a project first: All Tasks opens in one of its tabs", "请先打开一个项目：全部任务在项目的标签页中打开")))
		return
	}
	for _, r := range a.perspectiveTabRegions() {
		if i := slices.IndexFunc(r.Views, func(v *layout.View) bool { return v.Kind == layout.ViewAllTasks }); i >= 0 {
			a.report(a.state.SelectView(r.ID, r.Views[i].ID))
			a.layoutChanged()
			a.focusRegion(r.ID)
			return
		}
	}
	if _, err := a.state.OpenView(target.ID, layout.ViewAllTasks, project.Name); err != nil {
		a.report(err)
		return
	}
	a.layoutChanged()
	a.focusRegion(target.ID)
}

// allTasksFilter is the tab's filter, as jun tool task takes it.
func allTasksFilter(st *allTasksState) goals.TaskFilter {
	return goals.TaskFilter{States: st.states, All: len(st.states) == 0, Owner: st.owner, Project: st.project, Match: st.query}
}

// allTaskRow is a row of All Tasks: a month's heading, or a task.
type allTaskRow struct {
	group monthGroup
	task  *goals.Task
}

// allTaskRows lists the groups, each with its tasks when open; with a query,
// all are open.
func allTaskRows(groups []monthGroup, st *allTasksState) []allTaskRow {
	var rows []allTaskRow
	for _, g := range groups {
		rows = append(rows, allTaskRow{group: g})
		if st.open[g.Month] || st.query != "" {
			for _, t := range g.Tasks {
				rows = append(rows, allTaskRow{group: g, task: &t})
			}
		}
	}
	return rows
}

// allTasksView lists every task the filter picks, those not ended first,
// then by the month they ended.
func (a *app) allTasksView(c *ui.Context, v *layout.View) {
	st := a.allTasksOf(v.ID)
	rows := allTaskRows(monthGroups(goals.SelectTasks(a.goals, allTasksFilter(st))), st)
	ui.Column(c.Key(v.ID)).Fill().Padding(12).Gap(8).Children(func() {
		a.allTasksBar(c, st)
		if len(rows) == 0 {
			ui.Text(c, a.text("No task matches.", "没有匹配的任务。")).TextColor(a.theme.TextMuted).Padding(12)
			return
		}
		a.allTaskList(c, st, rows)
	})
}

// allTasksBar is the filter: the text, then the states and the owner and
// project.
func (a *app) allTasksBar(c *ui.Context, st *allTasksState) {
	ui.SearchField(c, &st.query).Label(a.text("Search tasks", "搜索任务")).
		Placeholder(a.text("Title, outcome or text", "标题、完成标准或说明")).Height(26).FontSize(theme.Caption).FillWidth()
	ui.Row(c).FillWidth().Gap(6).AlignItems(ui.Center).Children(func() {
		for _, s := range goals.TaskStates {
			on := slices.Contains(st.states, s)
			chip := ui.ButtonBase(c).Height(26).Padding(0, 8).Radius(4).Shrink(0).Children(func() {
				ui.Text(c, a.taskStateName(s)).FontSize(theme.Caption).TextColor(map[bool]ui.Color{false: a.theme.TextMuted, true: a.theme.AccentText}[on])
			})
			if on {
				chip.Background(a.theme.Accent)
			}
			chip.Label(a.taskStateName(s)).OnClick(func() {
				if on {
					st.states = slices.DeleteFunc(st.states, func(o goals.TaskState) bool { return o == s })
				} else {
					st.states = append(st.states, s)
				}
			})
		}
		a.allTasksChoice(c, a.text("Owner", "跟进人"), &st.owner, func(t goals.Task) string { return t.Owner })
		a.allTasksChoice(c, a.text("Project", "项目"), &st.project, func(t goals.Task) string { return t.Project })
	})
}

// allTasksChoice is a menu of the values the tasks have, and any.
func (a *app) allTasksChoice(c *ui.Context, name string, chosen *string, of func(goals.Task) string) {
	var values []string
	for _, t := range a.goals.Tasks {
		if v := of(t); v != "" && !slices.Contains(values, v) {
			values = append(values, v)
		}
	}
	slices.Sort(values)
	ui.MenuButton(c, cmp.Or(*chosen, name), func(m *ui.Menu) {
		if m.Item(a.text("Any", "全部")).Checked(*chosen == "").Chosen() {
			*chosen = ""
		}
		for _, v := range values {
			if m.Item(v).Checked(*chosen == v).Chosen() {
				*chosen = v
			}
		}
	}).Label(name).Height(26).FontSize(theme.Caption).Shrink(0)
}

func (a *app) taskStateName(s goals.TaskState) string {
	names := map[goals.TaskState][2]string{goals.TaskTodo: {"To do", "待办"}, goals.TaskDoing: {"Doing", "进行中"},
		goals.TaskDone: {"Done", "已完成"}, goals.TaskDropped: {"Dropped", "已放弃"}}
	return a.text(names[s][0], names[s][1])
}

// allTaskList keeps the keyboard on the list: ↩ shows the task in Tasks,
// → and ← open and close a month.
func (a *app) allTaskList(c *ui.Context, st *allTasksState, rows []allTaskRow) {
	key := func(r allTaskRow) any {
		if r.task == nil {
			return "month " + r.group.Month
		}
		return r.task.ID
	}
	st.list.Key = func(i int) any { return key(rows[i]) }
	st.list.Label = func(i int) string { return a.allTaskLabel(rows[i]) }
	clicked := false
	list := ui.List(c, &st.list, len(rows), func(i int) {
		if a.allTaskRowView(c, st, rows[i], i == st.selected) {
			st.selected, clicked = i, true
		}
	}).Gap(2).Grow(1).FillWidth()
	if clicked {
		list.Focus()
		c.Invalidate()
	}
	if st.selected < 0 || st.selected >= len(rows) {
		return
	}
	r := rows[st.selected]
	if r.task == nil && clicked {
		st.open[r.group.Month] = !st.open[r.group.Month]
	}
	if list.Submitted() && r.task != nil {
		a.revealTask(r.task.ID)
		c.Invalidate()
	}
	for k, open := range map[ui.Key]bool{ui.KeyRight: true, ui.KeyLeft: false} {
		if list.Shortcut(0, k) && r.task == nil {
			st.open[r.group.Month] = open
			c.Invalidate()
		}
	}
}

func (a *app) monthName(month string) string {
	if month == "" {
		return a.text("Not ended", "未结束")
	}
	return month
}

func (a *app) allTaskLabel(r allTaskRow) string {
	if r.task == nil {
		return fmt.Sprintf("%s %d", a.monthName(r.group.Month), len(r.group.Tasks))
	}
	return fmt.Sprintf("%s %s, %s", taskNumber(r.task.ID), r.task.Title, a.taskStateName(r.task.State))
}

// allTaskRowView draws the row and reports a click on it.
func (a *app) allTaskRowView(c *ui.Context, st *allTasksState, r allTaskRow, chosen bool) bool {
	muted, caption := a.messageMuted(chosen), a.messageCaption()
	row := ui.Row(c).FillWidth().Padding(4, 6).Radius(4).Gap(messageGap).AlignItems(ui.Center)
	row.Children(func() {
		if r.task == nil {
			mark := map[bool]string{false: " ▸", true: " ▾"}[st.open[r.group.Month] || st.query != ""]
			ui.Text(c, fmt.Sprintf("%s %d%s", a.monthName(r.group.Month), len(r.group.Tasks), mark)).FontSize(caption).Bold().TextColor(muted)
			return
		}
		t := *r.task
		a.taskRowContent(c, t, chosen)
		if g, ok := a.goals.Goal(t.Goal); ok {
			ui.Text(c, g.Title).FontSize(caption).TextColor(muted).SingleLine().Ellipsis("…").MaxWidth(200).Shrink(1)
		}
		if t.Project != "" {
			ui.Text(c, t.Project).FontSize(caption).TextColor(muted).Shrink(0)
		}
		if t.Ended != nil {
			ui.Text(c, t.Ended.Local().Format(time.DateOnly)).FontSize(caption).TextColor(muted).Shrink(0)
		}
	})
	return row.Clicked()
}
