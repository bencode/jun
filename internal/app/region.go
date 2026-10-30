package app

import (
	"fmt"
	"slices"

	"github.com/bencode/jun/internal/layout"
	"github.com/egoist/mygo/ui"
)

func (a *app) region(c *ui.Context, r *layout.Region) {
	views := slices.Clone(r.Views)
	p := a.state.CurrentPerspective()
	count, bottom := 0, false
	if p != nil {
		count, bottom = len(p.Columns), p.BottomVisible
	}
	// A menu choice can replace the layout during this build; keep focus pending for its next frame.
	current := func() bool {
		if r == a.state.Global {
			return a.state.GlobalVisible
		}
		return p != nil && a.state.CurrentPerspective() == p && len(p.Columns) == count && p.BottomVisible == bottom &&
			(slices.Contains(p.Columns, r) || r == p.Bottom && bottom)
	}
	region := ui.Column(c.Key(r.ID)).Fill().MinWidth(0).Clip().Children(func() {
		if !current() {
			return
		}
		ui.Row(c).Height(40).FillWidth().AlignItems(ui.Center).Background(a.theme.Surface).Gap(2).Children(func() {
			a.tabs(c, r, views)
			a.regionMenu(c, r)
			if p := a.state.CurrentPerspective(); p != nil && r == p.Columns[len(p.Columns)-1] {
				ui.Box(c).Width(38).Shrink(0)
			}
		})
		if !current() {
			return
		}
		body := ui.Column(c).Key("body").Grow(1).FillWidth().MinHeight(0)
		a.drop(body, r.ID, 0)
		body.Children(func() {
			if len(views) == 0 {
				ui.Column(c).Fill().AlignItems(ui.Center).Justify(ui.Center).Gap(12).Children(func() {
					ui.Text(c, a.text("Drop a view here", "将视图拖到这里")).TextColor(a.theme.TextMuted)
					ui.Button(c, a.text("Open View", "打开视图")).Menu(func(m *ui.Menu) { a.openViewMenu(m, r.ID) })
				})
			}
			for _, v := range views {
				if v.ID == r.Active {
					a.viewContent(c, v)
				}
			}
		})
	})
	if current() && region.FocusWithin() {
		a.activeRegion = r.ID
	}
}

func (a *app) tabs(c *ui.Context, r *layout.Region, views []*layout.View) {
	selected := slices.IndexFunc(views, func(v *layout.View) bool { return v.ID == r.Active })
	ui.ScrollHorizontal(c).Grow(1).MinWidth(0).Height(40).Children(func() {
		tabs := ui.TabsBase(c, &selected, len(views))
		tabs.List.Height(40).Gap(0).OnChange(func() {
			if selected >= 0 && selected < len(views) {
				if err := a.state.SelectView(r.ID, views[selected].ID); err != nil {
					a.report(err)
				} else {
					a.layoutChanged()
				}
			}
		}).Children(func() {
			for i, v := range views {
				c.Key(v.ID)
				tab := tabs.Tab(i).Height(40).Padding(0, 10).Gap(6).Drag(v.ID).Label(a.title(v))
				if v.ID == r.Active {
					tab.Background(a.theme.Background)
				}
				tab.OnClick(func() { a.focusRegion(r.ID) })
				tab.ContextMenu(func(m *ui.Menu) { a.tabMenu(m, r.ID, v.ID) })
				if session := a.sessionFor(v.ID); session != nil {
					tab.Tooltip(session.ProjectName + "\n" + session.CWD)
				}
				if a.pendingFocus == r.ID && v.ID == r.Active && v.Kind != layout.ViewTerminal {
					tab.Focus()
					a.pendingFocus = 0
				}
				a.drop(tab, r.ID, v.ID)
				tab.Children(func() {
					ui.Text(c, a.title(v)).MinWidth(0).MaxWidth(160).SingleLine().Ellipsis("…")
					if v.ID == r.Active || tab.Hovered() || tab.FocusWithin() {
						ui.Button(c, closeIcon).Padding(2, 4).Label(fmt.Sprintf("%s %s", a.text("Close", "关闭"), a.title(v))).
							Tooltip(a.text("Close Tab", "关闭标签")).OnClick(func() { a.closeView(r.ID, v.ID) })
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
