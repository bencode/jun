package app

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/bencode/jun/internal/history"
	"github.com/bencode/jun/internal/layout"
	"github.com/bencode/jun/internal/theme"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

var (
	claudeIcon = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 12 12"><path d="M6 .5 11.5 6 6 11.5.5 6Z" fill="currentColor"/></svg>`))
	copyIcon   = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><rect x="9" y="9" width="11" height="11" rx="2"/><path d="M5 15V6a2 2 0 0 1 2-2h9"/></svg>`))
	codexIcon  = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 12 12"><circle cx="6" cy="6" r="4.5" fill="none" stroke="currentColor" stroke-width="2"/></svg>`))
)

// sessionsLimit is how many sessions the list shows at most, the latest:
// a search finds older ones.
const sessionsLimit = 500

// sessionsRescan is how old a list of sessions is before showing it reads
// them again.
const sessionsRescan = 30 * time.Second

// sessionList is a project's agent sessions, as last read.
type sessionList struct {
	sessions []history.Session
	err      error
	loading  bool
	scanned  time.Time
	loads    int // counts scans, so that a later one's results replace an earlier one's
	repoRead int // the repository's refresh the directories came from
}

// sessionsViewState is a Sessions view's search and choice.
type sessionsViewState struct {
	query    string
	list     ui.ListState
	selected int
}

// sessionRow is a row of the list: a day's heading, or a session.
type sessionRow struct {
	day     string
	session history.Session
}

// sessionsFor returns the project's sessions, reading them again when the
// list is old or the project's worktrees changed, once they are known.
func (a *app) sessionsFor(project *layout.Project) *sessionList {
	l := a.sessionLists[project.CWD]
	if l == nil {
		l = &sessionList{}
		a.sessionLists[project.CWD] = l
	}
	r := a.repoFor(project.CWD)
	if !l.loading && !r.loading && (l.scanned.IsZero() || time.Since(l.scanned) > sessionsRescan || l.repoRead != r.loads) {
		a.scanSessions(l, r, project.CWD)
	}
	return l
}

// scanSessions reads the sessions in the project's directory and its
// worktrees.
func (a *app) scanSessions(l *sessionList, r *repoState, dir string) {
	dirs := []string{dir}
	for _, s := range r.sources {
		if s.Worktree != "" {
			dirs = append(dirs, s.Worktree)
		}
	}
	l.loads++
	load := l.loads
	l.loading, l.repoRead = true, r.loads
	var sessions []history.Session
	var err error
	a.background(func() { sessions, err = a.history.List(dirs) }, func() {
		if load != l.loads {
			return
		}
		l.sessions, l.err, l.loading, l.scanned = sessions, err, false, time.Now()
	})
}

// sessionsView lists the project's agent sessions by day, the latest
// first, or those matching a search: a click shows one read-only in a
// preview tab, a double click or Enter in a tab of its own.
func (a *app) sessionsView(c *ui.Context, v *layout.View) {
	project := a.project(v.ProjectName)
	if project == nil {
		return
	}
	l := a.sessionsFor(project)
	st := a.sessionViews[v.ID]
	if st == nil {
		st = &sessionsViewState{selected: -1}
		st.list.Selected = &st.selected
		a.sessionViews[v.ID] = st
	}
	ui.Column(c.Key(v.ID)).Fill().Gap(0).Children(func() {
		ui.Row(c).FillWidth().Height(36).Padding(0, 6, 0, 8).Gap(4).AlignItems(ui.Center).Children(func() {
			ui.SearchField(c, &st.query).Label(a.text("Search sessions", "搜索会话")).
				Placeholder(a.text("Search sessions", "搜索会话")).Height(26).FontSize(theme.Caption).Grow(1).MinWidth(60)
			if iconButton(c, refreshIcon, a.text("Read Again", "重新读取"), 24, 14).Clicked() && !l.loading {
				a.scanSessions(l, a.repoFor(project.CWD), project.CWD)
			}
		})
		if l.err != nil {
			ui.Text(c, l.err.Error()).FontSize(theme.Caption).TextColor(a.theme.Danger).Padding(4, 12)
		}
		matches, more := matchSessions(l.sessions, st.query, sessionsLimit)
		if more {
			ui.Text(c, fmt.Sprintf(a.text("The latest %d shown; search to find older ones.", "只显示最近的 %d 个，搜索可找到更早的。"), sessionsLimit)).
				FontSize(theme.Caption).TextColor(a.theme.TextMuted).Padding(2, 14, 4)
		}
		rows := sessionRows(matches, time.Now(), a.text)
		if len(rows) == 0 {
			a.sessionsMessage(c, l, st.query)
			return
		}
		a.sessionList(c, st, rows, project.Name)
	})
}

func (a *app) sessionsMessage(c *ui.Context, l *sessionList, query string) {
	message := a.text("No Claude Code or Codex sessions in this project yet.", "这个项目还没有 Claude Code 或 Codex 会话。")
	if l.loading && l.scanned.IsZero() {
		message = a.text("Reading sessions…", "正在读取会话……")
	} else if strings.TrimSpace(query) != "" {
		message = a.text("No sessions match.", "没有匹配的会话。")
	}
	ui.Text(c, message).TextColor(a.theme.TextMuted).Padding(12)
}

func (a *app) sessionList(c *ui.Context, st *sessionsViewState, rows []sessionRow, project string) {
	st.list.Key = func(i int) any {
		if rows[i].day != "" {
			return "day:" + rows[i].day
		}
		return rows[i].session.Path
	}
	st.list.Header = func(i int) bool { return rows[i].day != "" }
	st.list.Label = func(i int) string { return cmp.Or(rows[i].day, rows[i].session.Title) }
	list := ui.List(c, &st.list, len(rows), func(i int) {
		row := rows[i]
		if row.day != "" {
			ui.Text(c, row.day).FontSize(theme.Caption).Bold().TextColor(a.theme.TextMuted).
				FillWidth().Padding(8, 8, 4).Background(a.theme.Background)
			return
		}
		a.sessionRow(c, row.session, project)
	}).Grow(1).FillWidth()
	chosen := func() (history.Session, bool) {
		if st.selected < 0 || st.selected >= len(rows) || rows[st.selected].day != "" {
			return history.Session{}, false
		}
		return rows[st.selected].session, true
	}
	if list.Changed() {
		if s, ok := chosen(); ok {
			a.openTranscript(project, s, false)
		}
	}
	if list.Submitted() {
		if s, ok := chosen(); ok {
			a.openTranscript(project, s, true)
		}
	}
}

func (a *app) sessionRow(c *ui.Context, s history.Session, project string) {
	ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Start).Padding(4, 8).ContextMenu(func(m *ui.Menu) {
		if m.Item(a.text("Resume", "Resume")).Chosen() {
			a.resumeSession(project, s)
		}
		if m.Item(a.text("Open", "打开")).Chosen() {
			a.openTranscript(project, s, true)
		}
		if m.Item(a.text("Copy Session ID", "复制会话 ID")).Chosen() {
			mygo.Clipboard.WriteText(s.ID)
		}
	}).Children(func() {
		a.agentIcon(c, s.Agent, 11).Margin(3, 0, 0, 0)
		ui.Column(c).Grow(1).MinWidth(0).Gap(2).Children(func() {
			ui.Text(c, s.Title).MinWidth(0).SingleLine().Ellipsis("…")
			ui.Text(c, sessionDetail(s)).FontSize(theme.Caption).TextColor(a.theme.TextMuted).MinWidth(0).SingleLine().Ellipsis("…")
		})
	})
}

// agentIcon tells the agents apart: a diamond for Claude Code, a ring
// for Codex.
func (a *app) agentIcon(c *ui.Context, agent string, size float32) ui.Element {
	if agent == "codex" {
		return ui.Icon(c, codexIcon).Size(size, size).Shrink(0).TextColor(a.theme.Accent)
	}
	return ui.Icon(c, claudeIcon).Size(size, size).Shrink(0).TextColor(a.theme.Warning)
}

// sessionDetail is a row's second line: the branch and the time of day
// it was last active.
func sessionDetail(s history.Session) string {
	at := s.Updated.Local().Format("15:04")
	if s.Branch == "" {
		return at
	}
	return s.Branch + " · " + at
}

// matchSessions keeps the first limit sessions whose title or first or
// last prompt has every word of the query, ignoring case; more reports
// that others match too.
func matchSessions(sessions []history.Session, query string, limit int) (matches []history.Session, more bool) {
	words := strings.Fields(strings.ToLower(query))
	for _, s := range sessions {
		text := strings.ToLower(s.Title + "\n" + s.FirstPrompt + "\n" + s.LastPrompt)
		if slices.ContainsFunc(words, func(w string) bool { return !strings.Contains(text, w) }) {
			continue
		}
		if len(matches) == limit {
			return matches, true
		}
		matches = append(matches, s)
	}
	return matches, false
}

// sessionRows heads each day's sessions, latest first, with the day:
// today, yesterday, or its date.
func sessionRows(sessions []history.Session, now time.Time, text func(en, zh string) string) []sessionRow {
	var rows []sessionRow
	last := ""
	for _, s := range sessions {
		if day := dayName(s.Updated.Local(), now.Local(), text); day != last {
			rows = append(rows, sessionRow{day: day})
			last = day
		}
		rows = append(rows, sessionRow{session: s})
	}
	return rows
}

func dayName(t, now time.Time, text func(en, zh string) string) string {
	y, m, d := t.Date()
	day := time.Date(y, m, d, 0, 0, 0, 0, t.Location())
	y, m, d = now.Date()
	today := time.Date(y, m, d, 0, 0, 0, 0, now.Location())
	if day.Equal(today) {
		return text("Today", "今天")
	}
	if day.Equal(today.AddDate(0, 0, -1)) {
		return text("Yesterday", "昨天")
	}
	if t.Year() != now.Year() {
		return text(t.Format("Jan 2, 2006"), t.Format("2006年1月2日"))
	}
	return text(t.Format("Jan 2"), t.Format("1月2日"))
}
