package app

import (
	"fmt"
	"slices"

	"github.com/bencode/jun/internal/layout"
	"github.com/bencode/jun/internal/theme"
	"github.com/egoist/mygo/ui"
)

// shelveProject folds the project away in the rail; going to it again
// brings it back.
func (a *app) shelveProject(name string) {
	previous := a.state.ActiveProject
	if err := a.state.ShelveProject(name); err != nil {
		a.report(err)
		return
	}
	a.layoutChanged()
	if a.state.ActiveProject != previous {
		a.focusCurrentPerspective()
		a.checkCurrentDirectory()
	}
}

// canShelve reports whether the project can go on the shelf: the last
// shown one stays.
func (a *app) canShelve(name string) bool {
	return slices.ContainsFunc(a.state.ShownProjects(), func(p *layout.Project) bool { return p.Name != name })
}

func (a *app) shelveCurrentProject() {
	if name := a.state.ActiveProject; name != "" && a.canShelve(name) {
		a.shelveProject(name)
	}
}

// shelvedPrefix begins the palette titles of shelved projects, so that the
// palette typed with it lists only them.
func (a *app) shelvedPrefix() string { return a.text("Shelved Project: ", "已收起项目：") }

func (a *app) openShelf() {
	if len(a.state.ShelvedProjects()) > 0 {
		a.openPalette(a.shelvedPrefix())
	}
}

// shelfButton stands for the shelved projects, with the agents waiting in
// them; it opens the palette listing them.
func (a *app) shelfButton(c *ui.Context, shelved []*layout.Project, waiting map[string]int) {
	waits := 0
	for _, p := range shelved {
		waits += waiting[p.Name]
	}
	expanded := a.state.RailExpanded
	label := fmt.Sprintf(a.text("Shelved projects: %d", "已收起的项目：%d"), len(shelved))
	waitingLabel := a.waitingLabel(waits)
	a.railAction(ui.ButtonBase(c.Key("shelf")).Height(a.railTileHeight()).FillWidth().Shrink(0).Padding(0, 4).Gap(8).Justify(ui.Start).
		Label(label).Tooltip(label + " (" + shelfKeys.String() + ")")).Children(func() {
		ui.Box(c).Width(32).Height(32).Shrink(0).Radius(8).Border(1, a.theme.Border).Center().Children(func() {
			ui.Icon(c, moreIcon).Size(18, 18).Shrink(0)
			if waits > 0 && !expanded {
				a.waitingDot(c, waitingLabel)
			}
		})
		if !expanded {
			return
		}
		ui.Text(c, a.text("Shelved", "已收起")).Grow(1).MinWidth(0).SingleLine().Ellipsis("…")
		ui.Text(c, fmt.Sprint(len(shelved))).FontSize(theme.Caption).Shrink(0)
		if waits > 0 {
			a.waitingCount(c, waits, waitingLabel)
		}
	}).OnClick(a.openShelf)
}
