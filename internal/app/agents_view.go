package app

import (
	"fmt"
	"slices"

	"github.com/bencode/jun/internal/agent"
	"github.com/bencode/jun/internal/layout"
	"github.com/bencode/jun/internal/terminal"
	"github.com/bencode/jun/internal/theme"
	ghostty "github.com/egoist/mygo/plugins/terminal"
	"github.com/egoist/mygo/ui"
)

// agentsViewState is what an Agents view keeps between frames.
type agentsViewState struct {
	room     ui.Rect      // the area under the counts, measured last frame
	idleOpen bool         // the folded idle agents were shown
	followed terminal.ID  // the current agent last scrolled to
	grid     ui.GridState // the cards' scrolling
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
	sessions := a.agentSessions()
	ui.Column(c.Key(v.ID)).Fill().Gap(8).Padding(12).Children(func() {
		ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Start).Children(func() {
			a.agentCounts(c, sessions)
			a.agentsLayoutChoice(c, v)
		})
		if len(sessions) == 0 {
			ui.Column(c).Grow(1).FillWidth().AlignItems(ui.Center).Justify(ui.Center).Children(func() {
				ui.Text(c, a.text("No agents running", "没有正在运行的 agent")).Bold()
				ui.Text(c, a.text("Start claude, codex or pi in a terminal to see it here.", "在终端里启动 claude、codex 或 pi，就会显示在这里。")).
					TextColor(a.theme.TextMuted)
			})
			return
		}
		// The agents show as their room allows, so the room is measured first.
		area := ui.Box(c).Grow(1).FillWidth().MinHeight(0)
		st, room := a.agentsState(v.ID), area.LayoutBounds()
		if room.W != st.room.W || room.H != st.room.H {
			st.room = room
			c.Invalidate()
		}
		// A newly current agent is scrolled into view, once.
		current := slices.IndexFunc(sessions, func(s *terminal.Session) bool { return s.ID == a.lastTerminal })
		follow := current >= 0 && st.followed != a.lastTerminal
		if follow {
			st.followed = a.lastTerminal
		}
		area.Children(func() {
			width, height, fits := cardGrid(room.W, room.H, len(sessions))
			if v.AgentsLayout == layout.AgentsList || v.AgentsLayout == layout.AgentsAuto && !fits {
				a.agentList(c, st, sessions, room.H, follow)
				return
			}
			if !fits {
				width, height = cardWidth, cardHeight
			}
			if follow {
				st.grid.ScrollIntoView(current)
			}
			// Under the width of the cards, as GridView rounds down its columns.
			ui.GridView(c, &st.grid, len(sessions), width-0.5, height, func(i int) {
				a.agentCard(c, sessions[i])
			}).Fill()
		})
	})
}

// agentsLayoutChoice switches the view among cards when they fit, cards
// and a list, from a menu, which leaves the counts its width.
func (a *app) agentsLayoutChoice(c *ui.Context, v *layout.View) {
	names := map[layout.AgentsLayout][2]string{
		layout.AgentsAuto: {"Auto", "自动"}, layout.AgentsCards: {"Cards", "卡片"}, layout.AgentsList: {"List", "列表"},
	}
	name := func(l layout.AgentsLayout) string { return a.text(names[l][0], names[l][1]) }
	ui.ButtonBase(c).Height(24).Padding(0, 8).Gap(4).Radius(12).Shrink(0).
		Label(a.text("Agents layout", "Agents 显示方式")).Children(func() {
		ui.Text(c, name(v.AgentsLayout)).TextColor(a.theme.TextMuted)
		ui.Icon(c, downIcon).Size(12, 12).TextColor(a.theme.TextMuted)
	}).Menu(func(m *ui.Menu) {
		for _, l := range []layout.AgentsLayout{layout.AgentsAuto, layout.AgentsCards, layout.AgentsList} {
			if m.Item(name(l)).Radio(v.AgentsLayout == l).Chosen() {
				v.AgentsLayout = l
				a.layoutChanged()
			}
		}
	})
}

// agentList lists the agents under a heading for each status, in compact
// rows when rows would not fit, and with the idle ones folded under their
// heading when compact rows would not either.
func (a *app) agentList(c *ui.Context, st *agentsViewState, sessions []*terminal.Session, height float32, follow bool) {
	var groups [][]*terminal.Session
	for i, s := range sessions {
		if i == 0 || a.agents[s.ID].Status != a.agents[sessions[i-1].ID].Status {
			groups = append(groups, nil)
		}
		groups[len(groups)-1] = append(groups[len(groups)-1], s)
	}
	compact, foldIdle := listFit(height, len(groups), len(sessions))
	ui.Scroll(c).Fill().Children(func() {
		ui.Column(c).FillWidth().Gap(rowGap).Children(func() {
			for _, group := range groups {
				status := a.agents[group[0].ID].Status
				heading := fmt.Sprintf("%s %d", a.statusName(status), len(group))
				if status == agent.Idle && foldIdle && !st.idleOpen {
					ui.ButtonBase(c).FillWidth().Height(groupHeight).Padding(0, 10).Radius(4).Justify(ui.Start).Children(func() {
						ui.Text(c, heading+" ▸").FontSize(theme.Caption).TextColor(a.theme.TextMuted)
					}).OnClick(func() { st.idleOpen = true })
					continue
				}
				ui.Row(c).FillWidth().Height(groupHeight).Padding(0, 10).AlignItems(ui.Center).Children(func() {
					ui.Text(c, heading).FontSize(theme.Caption).TextColor(a.theme.TextMuted)
				})
				for _, s := range group {
					row := a.agentRow(c, s, compact)
					if follow && s.ID == a.lastTerminal {
						row.ScrollIntoView()
					}
				}
			}
		})
	})
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
// marked as the rail marks the current project.
func (a *app) agentRow(c *ui.Context, s *terminal.Session, compact bool) ui.Element {
	info := a.agents[s.ID]
	height, content := rowHeight, func() { a.agentHeading(c, s, info, false) }
	if compact {
		height, content = compactRowHeight, func() { a.compactHeading(c, s, info) }
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
	row.OnClick(func() { a.revealSession(s.ID) }).ContextMenu(func(m *ui.Menu) { a.agentMenu(m, s) })
	return row
}

func (a *app) compactHeading(c *ui.Context, s *terminal.Session, info agentInfo) {
	ui.Box(c).Width(8).Height(8).Radius(4).Shrink(0).Background(a.statusColor(info.Status))
	ui.Text(c, a.agentTitle(s)).Grow(1).MinWidth(0).SingleLine().Ellipsis("…")
	ui.Text(c, a.statusName(info.Status)).FontSize(theme.Caption).Shrink(0).TextColor(a.statusColor(info.Status))
	ui.Box(c).Width(22).Height(16).Radius(4).Shrink(0).Background(projectColor(s.ProjectName)).Center().Children(func() {
		ui.Text(c, projectAbbrev(s.ProjectName)).FontSize(theme.Caption).Bold().TextColor(ui.RGB(255, 255, 255))
	})
}

func (a *app) agentCard(c *ui.Context, s *terminal.Session) {
	info := a.agents[s.ID]
	border := a.theme.Border
	if info.Status == agent.Waiting {
		border = a.statusColor(agent.Waiting)
	}
	card := ui.ButtonBase(c.Key(s.ID)).Fill().Column().AlignItems(ui.Stretch).Gap(6).Padding(8).Radius(8).
		Border(1, border).Background(a.theme.Surface).Label(a.agentLabel(s))
	current := s.ID == a.lastTerminal
	card.Children(func() {
		heading := ui.Row(c).FillWidth().Height(32).Gap(10).Padding(0, 4).Radius(4).AlignItems(ui.Center).
			Children(func() { a.agentHeading(c, s, info, current) })
		if current {
			heading.Background(a.theme.Accent.Alpha(0.10))
		}
		a.cardScreen(c, s)
		// The current agent is marked as the focused tab is, clear of the corners.
		if current {
			ui.Box(c).Absolute().Top(0).Left(8).Right(8).Height(2).Background(a.theme.Accent)
		}
	})
	card.OnClick(func() { a.revealSession(s.ID) }).ContextMenu(func(m *ui.Menu) { a.agentMenu(m, s) })
}

// cardScreen shows a live copy of the agent's screen.
func (a *app) cardScreen(c *ui.Context, s *terminal.Session) {
	p := a.previewFor(s)
	if p == nil {
		return
	}
	// The area keeps its place as the font changes; the screen shows once
	// its font fits, as a new card's would first show at the default size.
	area := ui.Box(c).Grow(1).FillWidth().MinHeight(0).Clip()
	bounds := area.LayoutBounds()
	if p.fit(bounds) || bounds.W == 0 {
		c.Invalidate()
	}
	if p.fontSize > 0 {
		area.Children(func() { ghostty.View(c, p.term).Fill().PassThrough() })
	}
}

func (a *app) agentHeading(c *ui.Context, s *terminal.Session, info agentInfo, current bool) {
	ui.Box(c).Width(8).Height(8).Radius(4).Shrink(0).Background(a.statusColor(info.Status))
	ui.Column(c).Grow(1).MinWidth(0).Children(func() {
		ui.Text(c, a.agentTitle(s)).SingleLine().Ellipsis("…")
		ui.Text(c, s.ProjectName+" · "+agentName(info.Kind)).FontSize(theme.Caption).TextColor(a.theme.TextMuted).SingleLine().Ellipsis("…")
	})
	if current {
		ui.Text(c, a.text("Current", "当前")).FontSize(theme.Caption).Shrink(0).TextColor(a.theme.Accent)
	}
	ui.Text(c, a.statusName(info.Status)).FontSize(theme.Caption).Shrink(0).TextColor(a.statusColor(info.Status))
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
