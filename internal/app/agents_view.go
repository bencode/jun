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

// Below this width an Agents view lists agents; at it, a wall of cards
// shows each agent's screen.
const agentWallWidth float32 = 560

// regionOf finds the shown region holding the view.
func (a *app) regionOf(view layout.ViewID) layout.RegionID {
	regions := []*layout.Region{a.state.Global}
	if p := a.state.CurrentPerspective(); p != nil {
		regions = append(regions, p.Bottom)
		regions = append(regions, p.Columns...)
	}
	for _, r := range regions {
		if slices.ContainsFunc(r.Views, func(v *layout.View) bool { return v.ID == view }) {
			return r.ID
		}
	}
	return 0
}

func (a *app) agentsView(c *ui.Context, v *layout.View) {
	sessions, region := a.agentSessions(), a.regionOf(v.ID)
	container := ui.Column(c.Key(v.ID)).Fill().Gap(8).Padding(12).Children(func() {
		a.agentCounts(c, sessions)
		if len(sessions) == 0 {
			ui.Column(c).Grow(1).FillWidth().AlignItems(ui.Center).Justify(ui.Center).Children(func() {
				ui.Text(c, a.text("No agents running", "没有正在运行的 agent")).Bold()
				ui.Text(c, a.text("Start claude, codex or pi in a terminal to see it here.", "在终端里启动 claude、codex 或 pi，就会显示在这里。")).
					TextColor(a.theme.TextMuted)
			})
			return
		}
		if a.agentWide[v.ID] {
			ui.GridView(c, nil, len(sessions), 360, 260, func(i int) {
				a.agentCard(c, sessions[i], region)
			}).Gap(8).Grow(1).FillWidth()
			return
		}
		ui.Scroll(c).Grow(1).FillWidth().Children(func() {
			ui.Column(c).FillWidth().Gap(2).Children(func() {
				for _, session := range sessions {
					a.agentRow(c, session, region)
				}
			})
		})
	})
	if width := container.LayoutBounds().W; width > 0 && (width >= agentWallWidth) != a.agentWide[v.ID] {
		a.agentWide[v.ID] = width >= agentWallWidth
		c.Invalidate()
	}
}

func (a *app) agentCounts(c *ui.Context, sessions []*terminal.Session) {
	counts := map[agent.Status]int{}
	for _, s := range sessions {
		counts[a.agents[s.ID].Status]++
	}
	ui.Row(c).FillWidth().Gap(6).Wrap().Children(func() {
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

func (a *app) agentRow(c *ui.Context, s *terminal.Session, region layout.RegionID) {
	info := a.agents[s.ID]
	row := ui.ButtonBase(c.Key(s.ID)).FillWidth().Height(44).Padding(0, 10).Gap(10).Radius(6).
		Label(a.agentLabel(s)).Children(func() { a.agentHeading(c, s, info) })
	if info.Status == agent.Waiting {
		row.Background(a.statusColor(agent.Waiting).Alpha(0.14))
	}
	row.OnClick(func() { a.revealSession(s.ID, region) })
}

func (a *app) agentCard(c *ui.Context, s *terminal.Session, region layout.RegionID) {
	info := a.agents[s.ID]
	border := a.theme.Border
	if info.Status == agent.Waiting {
		border = a.statusColor(agent.Waiting)
	}
	card := ui.ButtonBase(c.Key(s.ID)).Fill().Column().AlignItems(ui.Stretch).Gap(6).Padding(8).Radius(8).
		Border(1, border).Background(a.theme.Surface).Label(a.agentLabel(s))
	card.Children(func() {
		ui.Row(c).FillWidth().Height(32).Gap(10).AlignItems(ui.Center).Children(func() { a.agentHeading(c, s, info) })
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
	})
	card.OnClick(func() { a.revealSession(s.ID, region) })
}

func (a *app) agentHeading(c *ui.Context, s *terminal.Session, info agentInfo) {
	ui.Box(c).Width(8).Height(8).Radius(4).Shrink(0).Background(a.statusColor(info.Status))
	ui.Column(c).Grow(1).MinWidth(0).Children(func() {
		ui.Text(c, a.agentTitle(s)).SingleLine().Ellipsis("…")
		ui.Text(c, s.ProjectName+" · "+agentName(info.Kind)).FontSize(theme.Caption).TextColor(a.theme.TextMuted).SingleLine().Ellipsis("…")
	})
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
	return fmt.Sprintf("%s, %s, %s %s", a.agentTitle(s), a.statusName(info.Status), s.ProjectName, agentName(info.Kind))
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
