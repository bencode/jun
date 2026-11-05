package app

import (
	"slices"
	"strings"

	"github.com/bencode/jun/internal/layout"
	"github.com/bencode/jun/internal/settings"
)

// headingPrefix opens the palette on the headings of the Markdown tab.
const headingPrefix = "@"

func (a *app) goToHeading() {
	a.openPalette(headingPrefix)
	a.palette.caretEnd = true
}

// headingCommand is the palette's command going to a Markdown page's
// heading.
func (a *app) headingCommand() paletteItem {
	return paletteItem{title: a.text("Go to Heading…", "跳转到标题…"), keys: goToHeadingKeys.String(),
		keywords: "outline heading 大纲 标题", run: a.goToHeading}
}

// widthCommands are the palette's commands for the widths pages read at,
// the current one checked.
func (a *app) widthCommands() []paletteItem {
	var items []paletteItem
	for _, w := range settings.MarkdownWidths {
		item := paletteItem{title: a.text("Reading Width: ", "阅读宽度：") + a.widthLabel(w), keywords: "width reading markdown 宽度 阅读",
			run: func() { a.setMarkdownWidth(w) }}
		if w == a.markdownWidth() {
			item.detail = "✓"
		}
		items = append(items, item)
	}
	return items
}

// readingTab is the tab a page is read in: the active one where new tabs
// go, as a page shown from the side bar leaves the focus there.
func (a *app) readingTab() *layout.View {
	r := a.tabsTarget()
	if r == nil {
		return nil
	}
	return activeView(r)
}

// readsAtWidth is whether the reading tab shows a page that reads at
// Reading Width: a Markdown page or a session's transcript.
func (a *app) readsAtWidth() bool {
	v := a.readingTab()
	return v != nil && (v.Transcript != nil || a.activeMarkdown() != nil)
}

// activeMarkdown is the Markdown page the reading tab shows, or nil when
// it shows something else.
func (a *app) activeMarkdown() *markdownRender {
	v := a.readingTab()
	if v == nil {
		return nil
	}
	st := a.fileViews[v.ID]
	if st == nil || len(st.renders) == 0 {
		return nil
	}
	m, _ := st.renders[st.shown].(*markdownRender)
	return m
}

// headingItems are the palette's items going to the headings of the
// Markdown tab whose text contains filter, ignoring case, indented by
// their level below the document's top one.
func (a *app) headingItems(filter string) []paletteItem {
	m := a.activeMarkdown()
	if m == nil {
		return []paletteItem{{title: a.text("The tab is not Markdown", "当前标签不是 Markdown"), run: func() {}}}
	}
	if len(m.headings) == 0 {
		return []paletteItem{{title: a.text("No headings", "没有标题"), run: func() {}}}
	}
	top := slices.MinFunc(m.headings, func(x, y heading) int { return x.level - y.level }).level
	filter = strings.ToLower(strings.TrimSpace(filter))
	var items []paletteItem
	for k, h := range m.headings {
		if strings.Contains(strings.ToLower(h.text), filter) {
			items = append(items, paletteItem{title: h.text, indent: h.level - top, run: func() { m.goTo(k) }})
		}
	}
	return items
}
