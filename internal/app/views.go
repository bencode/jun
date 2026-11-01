package app

import (
	"github.com/bencode/jun/internal/layout"
	"github.com/bencode/jun/internal/theme"
	"github.com/egoist/mygo/ui"
)

func (a *app) viewContent(c *ui.Context, v *layout.View) {
	if v.Kind == layout.ViewTerminal {
		a.terminalView(c, v)
		return
	}
	if v.Kind == layout.ViewAgents {
		a.agentsView(c, v)
		return
	}
	a.placeholder(c, v)
}

func (a *app) placeholder(c *ui.Context, v *layout.View) {
	if a.scroll[v.ID] == nil {
		a.scroll[v.ID] = &ui.ScrollState{}
	}
	ui.Scroll(c.Key(v.ID)).Fill().TrackScroll(a.scroll[v.ID]).Children(func() {
		ui.Column(c).Padding(28).Gap(12).FillWidth().Children(func() {
			ui.Text(c, a.kindName(v.Kind)).FontSize(theme.Heading).Bold()
			if v.ProjectName != "" {
				ui.Text(c, v.ProjectName).Bold()
				for _, p := range a.state.Projects {
					if p.Name == v.ProjectName {
						ui.Text(c, p.CWD).TextColor(a.theme.TextMuted)
					}
				}
			}
			ui.Text(c, a.text("Arrives in a later iteration.", "将在后续迭代接入。")).TextColor(a.theme.TextMuted)
		})
	})
}
