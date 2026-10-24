package app

import (
	"fmt"

	"github.com/bencode/jun/internal/layout"
	"github.com/egoist/mygo/ui"
)

func (a *app) viewContent(c *ui.Context, v *layout.View) {
	if v.Kind == layout.ViewTerminal {
		a.terminalView(c, v)
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
			ui.Text(c, a.kindName(v.Kind)).FontSize(22).Bold()
			ui.Text(c, a.text("Layout preview", "布局预览")).TextColor(a.theme.Accent)
			if v.ProjectName != "" {
				ui.Text(c, v.ProjectName).Bold()
				for _, p := range a.state.Projects {
					if p.Name == v.ProjectName {
						ui.Text(c, p.CWD).TextColor(a.theme.TextMuted)
					}
				}
			}
			stage := "I05"
			if v.Kind == layout.ViewAgents {
				stage = "I04"
			}
			ui.Text(c, fmt.Sprintf(a.text("Content arrives in %s.", "内容将在 %s 接入。"), stage)).TextColor(a.theme.TextMuted)
			ui.Text(c, a.text("Move this tab between columns, the bottom region, and Global.", "可以将标签移动到其他列、底部区域或 Global。")).TextColor(a.theme.TextMuted)
		})
	})
}
