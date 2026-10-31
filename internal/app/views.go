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
			ui.Text(c, a.text("Layout preview", "布局预览")).TextColor(a.theme.Accent)
			if v.ProjectName != "" {
				ui.Text(c, v.ProjectName).Bold()
				for _, p := range a.state.Projects {
					if p.Name == v.ProjectName {
						ui.Text(c, p.CWD).TextColor(a.theme.TextMuted)
					}
				}
			}
			ui.Text(c, a.text("Content arrives in I05.", "内容将在 I05 接入。")).TextColor(a.theme.TextMuted)
			ui.Text(c, a.text("Move this tab between columns, the bottom region, and Global.", "可以将标签移动到其他列、底部区域或 Global。")).TextColor(a.theme.TextMuted)
		})
	})
}
