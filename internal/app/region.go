package app

import (
	"fmt"
	"slices"

	"github.com/bencode/jun/internal/layout"
	"github.com/egoist/mygo/ui"
)

func (a *app) region(c *ui.Context, r *layout.Region, first bool) {
	views := slices.Clone(r.Views)
	ui.Column(c.Key(r.ID)).Fill().MinWidth(0).Clip().Children(func() {
		ui.Row(c).Height(40).FillWidth().AlignItems(ui.Center).Background(a.theme.Surface).Gap(2).Children(func() {
			if first {
				a.perspectiveSelect(c)
			}
			a.tabs(c, r, views)
			a.regionMenu(c, r)
			if p := a.state.CurrentPerspective(); p != nil && r == p.Columns[len(p.Columns)-1] {
				ui.Box(c).Width(38).Shrink(0)
			}
		})
		body := ui.Column(c).Key("body").Grow(1).FillWidth().MinHeight(0)
		a.drop(body, r.ID, 0)
		body.Children(func() {
			if len(views) == 0 {
				ui.Column(c).Fill().AlignItems(ui.Center).Justify(ui.Center).Gap(12).Children(func() {
					ui.Text(c, a.text("Drop a view here", "将视图拖到这里")).TextColor(a.theme.TextMuted)
					ui.Button(c, a.text("New Terminal", "新建终端")).Disabled(a.state.CurrentProject() == nil).
						OnClick(func() { a.newTerminal(r.ID) })
				})
			}
			for _, v := range views {
				if v.ID == r.Active {
					a.placeholder(c, v)
				}
			}
		})
	})
}

func (a *app) tabs(c *ui.Context, r *layout.Region, views []*layout.View) {
	selected := slices.IndexFunc(views, func(v *layout.View) bool { return v.ID == r.Active })
	ui.ScrollHorizontal(c).Grow(1).MinWidth(0).Height(40).Children(func() {
		tabs := ui.TabsBase(c, &selected, len(views))
		tabs.List.Height(40).Gap(0).OnChange(func() {
			if selected >= 0 && selected < len(views) {
				a.report(a.state.SelectView(r.ID, views[selected].ID))
			}
		}).Children(func() {
			for i, v := range views {
				c.Key(v.ID)
				tab := tabs.Tab(i).Height(40).Padding(0, 10).Gap(6).Drag(v.ID).Label(a.title(v))
				if v.ID == r.Active {
					tab.Background(a.theme.Background)
				}
				if a.pendingFocus == r.ID && v.ID == r.Active {
					tab.Focus()
					a.pendingFocus = 0
				}
				a.drop(tab, r.ID, v.ID)
				tab.Children(func() {
					ui.Text(c, a.title(v)).SingleLine()
					if v.ID == r.Active || tab.Hovered() || tab.FocusWithin() {
						ui.Button(c, closeIcon).Padding(2, 4).Label(fmt.Sprintf("%s %s", a.text("Close", "关闭"), a.title(v))).
							Tooltip(a.text("Close View", "关闭视图")).OnClick(func() { a.closeView(r.ID, v.ID) })
					}
				})
			}
		})
	})
}

func (a *app) drop(element ui.Element, target layout.RegionID, before layout.ViewID) {
	if _, over := ui.DragOver[layout.ViewID](element); over {
		element.Border(1, a.theme.Accent)
	}
	if view, ok := ui.Drop[layout.ViewID](element); ok {
		a.moveView(view, target, before)
	}
}
