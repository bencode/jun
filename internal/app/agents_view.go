package app

import (
	"fmt"
	"slices"

	"github.com/bencode/jun/internal/agent"
	"github.com/bencode/jun/internal/layout"
	"github.com/bencode/jun/internal/terminal"
	"github.com/bencode/jun/internal/theme"
	"github.com/egoist/mygo/ui"
)

// agentsViewState is what an Agents view keeps between frames.
type agentsViewState struct {
	room     ui.Rect     // the area under the counts, measured last frame
	idleOpen bool        // the folded idle agents were shown
	followed terminal.ID // the current agent last scrolled to
}

func (a *app) agentsState(view layout.ViewID) *agentsViewState {
	st := a.agentViews[view]
	if st == nil {
		st = &agentsViewState{}
		a.agentViews[view] = st
	}
	return st
}

func (a *app) agentsView(c *ui.Context, v *layout.View) {
	groups := a.agentGroups()
	sessions := slices.Concat(groups...)
	ui.Column(c.Key(v.ID)).Fill().Gap(8).Padding(12).Children(func() {
		ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Start).Children(func() {
			a.agentCounts(c, sessions)
		})
		if len(sessions) == 0 {
			ui.Column(c).Grow(1).FillWidth().AlignItems(ui.Center).Justify(ui.Center).Children(func() {
				ui.Text(c, a.text("No agents running", "没有正在运行的 agent")).Bold()
				ui.Text(c, a.text("Start claude, codex or pi in a terminal to see it here.", "在终端里启动 claude、codex 或 pi，就会显示在这里。")).
					TextColor(a.theme.TextMuted)
			})
			return
		}
		// The list fits its room, so the room is measured first.
		area := ui.Box(c).Grow(1).FillWidth().MinHeight(0)
		st, room := a.agentsState(v.ID), area.LayoutBounds()
		if room.W != st.room.W || room.H != st.room.H {
			st.room = room
			c.Invalidate()
		}
		// A newly current agent is scrolled into view, once.
		follow := st.followed != a.lastTerminal &&
			slices.ContainsFunc(sessions, func(s *terminal.Session) bool { return s.ID == a.lastTerminal })
		if follow {
			st.followed = a.lastTerminal
		}
		area.Children(func() { a.agentList(c, st, groups, room.H, follow) })
	})
}

// memberIndent is how far an agent of a group is set in from its lead.
const memberIndent float32 = 18

// agentList lists the groups of agents under a heading for the status of
// their most urgent agent, in compact rows when rows would not fit, and
// with the idle ones folded under their heading when compact rows would
// not either. A heading counts the agents under it.
func (a *app) agentList(c *ui.Context, st *agentsViewState, groups [][]*terminal.Session, height float32, follow bool) {
	var sections [][][]*terminal.Session
	for i, group := range groups {
		if i == 0 || a.urgency(group) != a.urgency(groups[i-1]) {
			sections = append(sections, nil)
		}
		sections[len(sections)-1] = append(sections[len(sections)-1], group)
	}
	compact, foldIdle := listFit(height, len(sections), len(slices.Concat(groups...)))
	ui.Scroll(c).Fill().Children(func() {
		ui.Column(c).FillWidth().Gap(rowGap).Children(func() {
			for _, section := range sections {
				status := a.urgency(section[0])
				heading := fmt.Sprintf("%s %d", a.statusName(status), len(slices.Concat(section...)))
				if status == agent.Idle && foldIdle && !st.idleOpen {
					ui.ButtonBase(c).FillWidth().Height(groupHeight).Padding(0, 10).Radius(4).Justify(ui.Start).Children(func() {
						ui.Text(c, heading+" ▸").FontSize(theme.Caption).TextColor(a.theme.TextMuted)
					}).OnClick(func() { st.idleOpen = true })
					continue
				}
				ui.Row(c).FillWidth().Height(groupHeight).Padding(0, 10).AlignItems(ui.Center).Children(func() {
					ui.Text(c, heading).FontSize(theme.Caption).TextColor(a.theme.TextMuted)
				})
				for _, group := range section {
					for i, s := range group {
						row := a.groupRow(c, s, compact, group[0].ID, i, len(group))
						if follow && s.ID == a.lastTerminal {
							row.ScrollIntoView()
						}
					}
				}
			}
		})
	})
}

// groupRow shows the agent at index i of its group: the lead as any row,
// the others set in under it, with a line from it.
func (a *app) groupRow(c *ui.Context, s *terminal.Session, compact bool, groupLead terminal.ID, i, n int) ui.Element {
	if i == 0 {
		return a.agentRow(c, s, compact, groupLead)
	}
	height := rowHeight
	if compact {
		height = compactRowHeight
	}
	// The line comes down through the gap above; the last row's ends at it.
	line := height + rowGap
	if i == n-1 {
		line = height/2 + rowGap
	}
	color := a.theme.TextMuted.Alpha(0.5)
	var row ui.Element
	ui.Row(c).FillWidth().Padding(0, 0, 0, memberIndent).Children(func() {
		ui.Box(c).Absolute().Left(8).Top(-rowGap).Width(1).Height(line).Background(color)
		ui.Box(c).Absolute().Left(8).Top(height / 2).Width(7).Height(1).Background(color)
		row = a.agentRow(c, s, compact, groupLead)
	})
	return row
}

func (a *app) agentCounts(c *ui.Context, sessions []*terminal.Session) {
	counts := map[agent.Status]int{}
	for _, s := range sessions {
		counts[a.agents[s.ID].Status]++
	}
	ui.Row(c).Grow(1).MinWidth(0).Gap(6).Wrap().Children(func() {
		a.countChip(c, a.text("All", "全部"), len(sessions), nil)
		for _, status := range []agent.Status{agent.Waiting, agent.Working, agent.Idle} {
			color := a.statusColor(status)
			a.countChip(c, a.statusName(status), counts[status], &color)
		}
	})
}

// countChip shows a count as a pill, led by the color of its status; a
// count of none is muted.
func (a *app) countChip(c *ui.Context, name string, n int, dot *ui.Color) {
	text := a.theme.Text
	if n == 0 {
		text = a.theme.TextMuted
	}
	ui.Row(c).Height(24).Padding(0, 10).Gap(6).Radius(12).AlignItems(ui.Center).Background(a.theme.Surface).Children(func() {
		if dot != nil {
			color := *dot
			if n == 0 {
				color = a.theme.TextMuted
			}
			ui.Box(c).Width(6).Height(6).Radius(3).Shrink(0).Background(color)
		}
		ui.Text(c, fmt.Sprintf("%s %d", name, n)).TextColor(text)
	})
}

// agentRow shows an agent in a list: its title over its project and kind,
// or, compact, its title beside its project's tile. The current agent is
// marked as the rail marks the current project. It marks the agent's lead
// unless that is its group's, whose line marks it.
func (a *app) agentRow(c *ui.Context, s *terminal.Session, compact bool, groupLead terminal.ID) ui.Element {
	info := a.agents[s.ID]
	height, content := rowHeight, func() { a.agentHeading(c, s, info, groupLead) }
	if compact {
		height, content = compactRowHeight, func() { a.compactHeading(c, s, info, groupLead) }
	}
	row := ui.ButtonBase(c.Key(s.ID)).FillWidth().Height(height).Padding(0, 10).Gap(10).Radius(6).
		Label(a.agentLabel(s)).Children(func() {
		content()
		if s.ID == a.lastTerminal {
			ui.Box(c).Absolute().Left(0).Top((height - 16) / 2).Width(2).Height(16).Radius(1).Background(a.theme.Accent)
		}
	})
	if compact {
		row.Tooltip(s.ProjectName + " · " + agentName(info.Kind))
	}
	if info.Status == agent.Waiting {
		row.Background(a.statusColor(agent.Waiting).Alpha(0.14))
	}
	if s.ID == a.peek.id {
		row.Border(2, a.theme.Accent)
	}
	row.OnClick(a.agentClick(row, s.ID)).ContextMenu(func(m *ui.Menu) { a.agentMenu(m, s) })
	return row
}

func (a *app) compactHeading(c *ui.Context, s *terminal.Session, info agentInfo, groupLead terminal.ID) {
	ui.Box(c).Width(8).Height(8).Radius(4).Shrink(0).Background(a.statusColor(info.Status))
	a.terminalNumber(c, s)
	ui.Text(c, a.agentTitle(s)).Grow(1).MinWidth(0).SingleLine().Ellipsis("…")
	a.leadMark(c, s, groupLead)
	ui.Text(c, a.statusName(info.Status)).FontSize(theme.Caption).Shrink(0).TextColor(a.statusColor(info.Status))
	ui.Box(c).Width(22).Height(16).Radius(4).Shrink(0).Background(projectColor(s.ProjectName)).Center().Children(func() {
		ui.Text(c, projectAbbrev(s.ProjectName)).FontSize(theme.Caption).Bold().TextColor(ui.RGB(255, 255, 255))
	})
}

// agentClick peeks the agent, or closes the drawer peeking it; with ⌘
// held, it shows its tab.
func (a *app) agentClick(e ui.Element, id terminal.ID) func() {
	return func() { a.clickAgent(id, e.ClickModifiers()&ui.Super != 0) }
}

// clickAgent peeks the agent, or closes its peek; with reveal, it shows the
// agent's terminal in its tab.
func (a *app) clickAgent(id terminal.ID, reveal bool) {
	if reveal {
		a.revealSession(id)
	} else if id == a.peek.id {
		a.closePeek()
	} else {
		a.peekAgent(id)
	}
}

func (a *app) agentHeading(c *ui.Context, s *terminal.Session, info agentInfo, groupLead terminal.ID) {
	ui.Box(c).Width(8).Height(8).Radius(4).Shrink(0).Background(a.statusColor(info.Status))
	ui.Column(c).Grow(1).MinWidth(0).Children(func() {
		ui.Row(c).FillWidth().Gap(4).AlignItems(ui.Center).Children(func() {
			a.terminalNumber(c, s)
			ui.Text(c, a.agentTitle(s)).Grow(1).MinWidth(0).SingleLine().Ellipsis("…")
		})
		ui.Text(c, s.ProjectName+" · "+agentName(info.Kind)).FontSize(theme.Caption).TextColor(a.theme.TextMuted).SingleLine().Ellipsis("…")
	})
	if s.ID == a.peek.id {
		ui.Text(c, a.text("Peeking", "查看中")).FontSize(theme.Caption).Shrink(0).TextColor(a.theme.Accent)
	}
	a.leadMark(c, s, groupLead)
	ui.Text(c, a.statusName(info.Status)).FontSize(theme.Caption).Shrink(0).TextColor(a.statusColor(info.Status))
}

// leadMark names the terminal of the agent that leads the agent, if one
// does other than its group's lead.
func (a *app) leadMark(c *ui.Context, s *terminal.Session, groupLead terminal.ID) {
	if lead := a.lead(s.ID); lead != 0 && lead != groupLead {
		ui.Text(c, fmt.Sprintf("↳ #%d", lead)).FontSize(theme.Caption).Shrink(0).TextColor(a.theme.TextMuted)
	}
}

func (a *app) agentTitle(s *terminal.Session) string {
	if title := agent.DisplayTitle(s.Terminal.Title()); title != "" {
		return title
	}
	return a.sessionTitle(s)
}

func (a *app) agentLabel(s *terminal.Session) string {
	info := a.agents[s.ID]
	label := fmt.Sprintf("%s, %s, %s %s", a.agentTitle(s), a.statusName(info.Status), s.ProjectName, agentName(info.Kind))
	if lead := a.lead(s.ID); lead != 0 {
		label += fmt.Sprintf(a.text(", led by #%d", "，协调者 #%d"), lead)
	}
	if s.ID == a.lastTerminal {
		label += a.text(", current", "，当前")
	}
	return label
}

func agentName(kind agent.Kind) string {
	return map[agent.Kind]string{agent.Claude: "Claude", agent.Codex: "Codex", agent.Pi: "Pi"}[kind]
}

func (a *app) statusName(status agent.Status) string {
	names := map[agent.Status][2]string{agent.Waiting: {"Waiting", "等待中"}, agent.Working: {"Working", "工作中"}, agent.Idle: {"Idle", "空闲"}}
	pair := names[status]
	return a.text(pair[0], pair[1])
}

func (a *app) statusColor(status agent.Status) ui.Color {
	return map[agent.Status]ui.Color{agent.Waiting: a.theme.Warning, agent.Working: a.theme.Accent, agent.Idle: a.theme.TextMuted}[status]
}
