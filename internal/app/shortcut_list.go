package app

import (
	"regexp"
	"slices"
	"strings"
)

// bilingual is text in English and Chinese, to search in both.
type bilingual struct{ en, zh string }

func (a *app) say(b bilingual) string { return a.text(b.en, b.zh) }

// shortcutRow is a line of the Keyboard Shortcuts panel: an action, the
// keys doing it, any of them, and where they do.
type shortcutRow struct {
	action bilingual
	keys   []string
	note   bilingual
}

type shortcutGroup struct {
	name bilingual
	rows []shortcutRow
}

func keys(k ...string) []string { return k }

// shortcutGroups are Jun's shortcuts, those of its menu bar and the
// terminal's, and the keys of the command palette and Files' search field.
func shortcutGroups() []shortcutGroup {
	inRegion := bilingual{"In the active region, wrapping around", "当前区域内循环"}
	inPalette := bilingual{"In the command palette", "在命令面板中"}
	inSearch := bilingual{"In Files’ search field", "在 Files 搜索框中"}
	inPeek := bilingual{"While an agent is peeked; switches project and perspective as needed", "Peek 打开时；按需切换项目和视角"}
	scrollback := bilingual{"The scrollback; a full-screen program, such as vim, gets the keys", "滚动回看内容；vim 等全屏程序中按键交给程序"}
	return []shortcutGroup{
		{bilingual{"Tabs", "标签"}, []shortcutRow{
			{bilingual{"New terminal", "新建终端"}, keys(newTerminalKeys.String()), bilingual{
				"In the active column or bottom region; from the side bar or Global, in the one focused last, else the first column",
				"当前列或底部区域；焦点在侧边栏或 Global 时，放到最近聚焦的列或底部区域，否则第一列"}},
			{bilingual{"Close tab", "关闭当前标签"}, keys(closeTabKeys.String()), bilingual{
				"Columns and the bottom region; the terminal keeps running in the background. The side bar’s and Global’s tabs stay; the region stays open after its last tab",
				"列和底部区域；终端会话保留在后台。侧边栏和 Global 的固定标签不关闭；关闭最后一个标签后区域保留"}},
			{bilingual{"Next tab", "下一个标签"}, keys("⌃Tab", "⌘⇧]"), inRegion},
			{bilingual{"Previous tab", "上一个标签"}, keys("⌃⇧Tab", "⌘⇧["), inRegion},
			{bilingual{"Tab 1–8", "第 1–8 个标签"}, keys("⌘1–⌘8"), bilingual{"In the active region; beyond the last does nothing", "当前区域；超出时无操作"}},
			{bilingual{"Last tab", "最后一个标签"}, keys("⌘9"), bilingual{"In the active region; not the ninth tab", "当前区域；不是第九个标签"}},
		}},
		{bilingual{"Regions", "区域"}, []shortcutRow{
			{bilingual{"Focus the region left / right / above / below", "聚焦左 / 右 / 上 / 下方的区域"}, keys("⌘⌥← / ⌘⌥→ / ⌘⌥↑ / ⌘⌥↓"), bilingual{
				"By position, stopping at the edge; up from the bottom region and right from the side bar focus the first column",
				"按布局移动，到边界停止；从底部区域向上、从侧边栏向右都到第一列"}},
			{bilingual{"Show or hide the side bar", "展开或收起侧边栏"}, keys(toggleSideKeys.String()), bilingual{
				"Of the current perspective; a menu bar shortcut, so it works while a dialog is open", "当前视角；属于菜单栏快捷键，弹窗打开时也生效"}},
		}},
		{bilingual{"Navigation", "导航"}, []shortcutRow{
			{bilingual{"Back", "后退"}, keys("⌘["), bilingual{
				"To where you were before, across tabs, perspectives and projects; skips closed tabs",
				"回到之前所在的位置，可跨标签、视角和项目；跳过已关闭的标签"}},
			{bilingual{"Forward", "前进"}, keys("⌘]"), bilingual{"After going back", "后退之后再前进"}},
		}},
		{bilingual{"Perspectives", "视角"}, []shortcutRow{
			{bilingual{"Next perspective", "下一个视角"}, keys("⌘⌥]"), bilingual{"In the current project, wrapping around; focuses the first column", "当前项目内循环，切换后聚焦第一列"}},
			{bilingual{"Previous perspective", "上一个视角"}, keys("⌘⌥["), bilingual{"In the current project, wrapping around; focuses the first column", "当前项目内循环，切换后聚焦第一列"}},
			{bilingual{"Perspective 1–9", "第 1–9 个视角"}, keys("⌘⌥1–⌘⌥9"), bilingual{"In the Perspective menu’s order; a missing one does nothing", "按视角菜单顺序；不存在时无操作"}},
			{bilingual{"New perspective", "新建视角"}, keys(newPerspectiveKeys.String()), bilingual{}},
		}},
		{bilingual{"Projects", "项目"}, []shortcutRow{
			{bilingual{"Next project", "下一个项目"}, keys("⌃⌘]"), bilingual{"In the project rail’s order, wrapping around", "按项目栏顺序循环"}},
			{bilingual{"Previous project", "上一个项目"}, keys("⌃⌘["), bilingual{"In the project rail’s order, wrapping around", "按项目栏顺序循环"}},
			{bilingual{"Project 1–9", "第 1–9 个项目"}, keys("⌃⌘1–⌃⌘9"), bilingual{"A missing one does nothing", "不存在时无操作"}},
			{bilingual{"Open project", "打开项目"}, keys(openProjectKeys.String()), bilingual{"The command palette at ~/", "打开命令面板，预填 ~/"}},
		}},
		{bilingual{"Files", "文件"}, []shortcutRow{
			{bilingual{"Search files", "搜索文件"}, keys("⌘P"), bilingual{"Shows the side bar’s Files and focuses its search field", "展开侧边栏的 Files 并聚焦搜索框"}},
			{bilingual{"Preview the first match", "预览首个匹配文件"}, keys("↓"), inSearch},
			{bilingual{"Open the first match in a tab", "在标签中打开首个匹配文件"}, keys("Enter"), inSearch},
		}},
		{bilingual{"Agents", "Agents"}, []shortcutRow{
			{bilingual{"Peek the next waiting agent", "Peek 下一个等待的 agent"}, keys(peekKeys.String()), bilingual{
				"In a drawer beside Global; with none waiting, the current agent. Again goes on, and closes once no other waits",
				"在 Global 旁的抽屉中显示；没有等待的 agent 时显示当前 agent。再按切到下一个，没有其他等待时关闭"}},
			{bilingual{"Show the peeked agent’s tab", "显示 Peek 中 agent 的标签"}, keys("⌘↩"), inPeek},
			{bilingual{"Close the peek", "关闭 Peek"}, keys(closeTabKeys.String()), bilingual{
				"Esc goes to the agent; ✕, a click beside the drawer or on the agent's card close it too", "Esc 发给 agent；点 ✕、抽屉外的区域或 agent 的卡片也会关闭"}},
		}},
		{bilingual{"Terminal", "终端"}, []shortcutRow{
			{bilingual{"Copy / paste / select all", "复制 / 粘贴 / 全选"}, keys("⌘C / ⌘V / ⌘A"), bilingual{}},
			{bilingual{"Scroll up a page", "向上滚动一页"}, keys("⌘PageUp", "⇧PageUp"), scrollback},
			{bilingual{"Scroll down a page", "向下滚动一页"}, keys("⌘PageDown", "⇧PageDown"), scrollback},
			{bilingual{"Scroll to the top / bottom", "滚到顶部 / 底部"}, keys("⌘Home / ⌘End"), scrollback},
		}},
		{bilingual{"Window and app", "窗口与应用"}, []shortcutRow{
			{bilingual{"Command palette", "命令面板"}, keys(commandPaletteKeys.String()), bilingual{}},
			{bilingual{"Move among items", "在条目间移动"}, keys("↑ / ↓"), inPalette},
			{bilingual{"Go into the highlighted directory", "进入高亮目录"}, keys("Tab"), inPalette},
			{bilingual{"Run the highlighted item", "执行高亮项"}, keys("Enter"), inPalette},
			{bilingual{"Close a dialog", "关闭弹窗"}, keys("Esc"), bilingual{"The command palette and other dialogs", "命令面板和其他弹窗"}},
			{bilingual{"Review changes", "查看分支改动"}, keys(reviewChangesKeys.String()), bilingual{"Lists the project’s branches in the command palette", "在命令面板中选择改动来源"}},
			{bilingual{"Go to heading", "跳转到标题"}, keys(goToHeadingKeys.String()), bilingual{"Lists the Markdown tab’s headings in the command palette (@)", "在命令面板中列出 Markdown 标签页的标题（@）"}},
			{bilingual{"Larger text", "放大文字"}, keys("⌘+", "⌘="), bilingual{"Of terminals and file tabs, which share it; saved", "终端与文件标签共用字号，修改会保存"}},
			{bilingual{"Smaller text", "缩小文字"}, keys("⌘−"), bilingual{"Of terminals and file tabs, which share it; saved", "终端与文件标签共用字号，修改会保存"}},
			{bilingual{"Default text size", "恢复默认字号"}, keys("⌘0"), bilingual{"Back to 13", "恢复为 13"}},
			{bilingual{"Settings", "设置"}, keys("⌘,"), bilingual{"Opens the settings file in a tab of the current perspective", "在当前视角的标签中打开设置文件"}},
			{bilingual{"Save settings", "保存设置"}, keys("⌘S"), bilingual{"In the settings tab; they apply at once", "在设置标签中；保存后立即生效"}},
			{bilingual{"Undo / redo", "撤销 / 重做"}, keys("⌘Z / ⌘⇧Z"), bilingual{"Edit menu, for the text being edited", "Edit 菜单，作用于正在编辑的文字"}},
			{bilingual{"Cut / paste and match style", "剪切 / 粘贴并匹配样式"}, keys("⌘X / ⌘⌥⇧V"), bilingual{"Edit menu, for the text being edited", "Edit 菜单，作用于正在编辑的文字"}},
			{bilingual{"Full screen", "全屏"}, keys("⌃⌘F"), bilingual{}},
			{bilingual{"Minimize", "最小化"}, keys("⌘M"), bilingual{}},
			{bilingual{"Quit Jun", "退出 Jun"}, keys("⌘Q"), bilingual{"Asks first while programs run in terminals; quitting ends the terminals", "终端中有程序运行时先确认；退出会结束终端会话"}},
		}},
	}
}

// matchShortcuts keeps the rows whose action, in either language, or keys
// hold the query, ignoring case, and the groups left with any.
func matchShortcuts(groups []shortcutGroup, query string) []shortcutGroup {
	q := strings.ToLower(strings.TrimSpace(query))
	k := normalizeKeys(q)
	var found []shortcutGroup
	for _, g := range groups {
		rows := slices.DeleteFunc(slices.Clone(g.rows), func(r shortcutRow) bool {
			return !strings.Contains(strings.ToLower(r.action.en), q) && !strings.Contains(r.action.zh, q) &&
				!strings.Contains(normalizeKeys(strings.Join(r.keys, " / ")), k)
		})
		if len(rows) > 0 {
			found = append(found, shortcutGroup{g.name, rows})
		}
	}
	return found
}

var modifierWord = regexp.MustCompile(`(?i)\b(command|cmd|control|ctrl|option|opt|alt|shift)\b\s*[+-]?`)

var modifierSymbols = map[string]string{
	"command": "⌘", "cmd": "⌘", "control": "⌃", "ctrl": "⌃", "option": "⌥", "opt": "⌥", "alt": "⌥", "shift": "⇧",
}

// modifierOrder is how shortcuts order their modifiers.
const modifierOrder = "⌃⌘⌥⇧"

// normalizeKeys writes keys typed as words, such as Shift+Cmd+N, as the
// panel shows them, ⌘⇧n, lowercase and without spaces, so that the two
// compare.
func normalizeKeys(s string) string {
	s = modifierWord.ReplaceAllStringFunc(s, func(m string) string {
		return modifierSymbols[strings.ToLower(strings.TrimRight(m, "+- \t"))]
	})
	s = strings.Join(strings.Fields(strings.ToLower(strings.ReplaceAll(s, "−", "-"))), "")
	runes := []rune(s)
	for i := 0; i < len(runes); {
		j := i
		for j < len(runes) && strings.ContainsRune(modifierOrder, runes[j]) {
			j++
		}
		slices.SortFunc(runes[i:j], func(x, y rune) int {
			return strings.IndexRune(modifierOrder, x) - strings.IndexRune(modifierOrder, y)
		})
		i = j + 1
	}
	return string(runes)
}
