package app

import (
	"slices"
	"strings"

	"github.com/bencode/jun/internal/settings"
)

// headingPrefix opens the palette on the headings of the Markdown tab.
const headingPrefix = "@"

func (a *app) goToHeading() {
	a.openPalette(headingPrefix)
	a.palette.caretEnd = true
}

// readingCommands are the palette's commands for Markdown pages: going to
// a heading, and the widths, the current one checked.
func (a *app) readingCommands() []paletteItem {
	items := []paletteItem{{title: a.text("Go to Heading…", "跳转到标题…"), keys: goToHeadingKeys.String(),
		keywords: "outline heading 大纲 标题", run: a.goToHeading}}
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

// activeMarkdown is the Markdown page the active region's tab shows, or
// nil when it shows something else.
func (a *app) activeMarkdown() *markdownRender {
	r := a.targetRegion()
	if r == nil {
		return nil
	}
	v := activeView(r)
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
