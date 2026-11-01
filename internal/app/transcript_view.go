package app

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/bencode/jun/internal/agent"
	"github.com/bencode/jun/internal/history"
	"github.com/bencode/jun/internal/layout"
	"github.com/bencode/jun/internal/theme"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown"
)

// transcriptState is what a transcript view read of its session.
type transcriptState struct {
	ref      layout.SessionRef // the session read: a preview tab moves on to others
	read     bool
	loading  bool
	readAt   time.Time // when the file had last changed, as it was read
	messages []history.Message
	err      error
	list     ui.ListState
	theme    markdown.Theme // the answers' text size; the window theme's for the rest
	options  markdown.Options
}

// openTranscript shows the session read-only where new tabs go: in the
// preview tab, which the next one opened replaces, unless pinned.
func (a *app) openTranscript(project string, s history.Session, pinned bool) {
	target := a.tabsTarget()
	if target == nil {
		return
	}
	ref := layout.SessionRef{Agent: s.Agent, ID: s.ID, Path: s.Path}
	shown := slices.ContainsFunc(target.Views, func(v *layout.View) bool {
		return v.Transcript != nil && v.ProjectName == project && *v.Transcript == ref
	})
	preview := a.previewTabs[target.ID]
	id, err := a.state.OpenTranscript(target.ID, project, ref, preview)
	if err != nil {
		a.report(err)
		return
	}
	if pinned && id == preview {
		delete(a.previewTabs, target.ID)
	} else if !pinned && !shown {
		a.previewTabs[target.ID] = id
	}
	a.layoutChanged()
}

// sessionOf is the listed session a transcript view shows; with only its
// reference while the project's sessions are being read.
func (a *app) sessionOf(project *layout.Project, ref layout.SessionRef) history.Session {
	l := a.sessionsFor(project)
	if i := slices.IndexFunc(l.sessions, func(s history.Session) bool { return s.Path == ref.Path }); i >= 0 {
		return l.sessions[i]
	}
	return history.Session{Agent: ref.Agent, ID: ref.ID, Path: ref.Path}
}

// transcriptView shows an agent session's prompts and answers, the latest
// at the bottom, read again as the session goes on, with Resume.
func (a *app) transcriptView(c *ui.Context, v *layout.View) {
	project := a.project(v.ProjectName)
	if project == nil || v.Transcript == nil {
		return
	}
	s := a.sessionOf(project, *v.Transcript)
	st := a.transcripts[v.ID]
	if st == nil || st.ref != *v.Transcript {
		st = &transcriptState{ref: *v.Transcript, list: ui.ListState{FollowEnd: true}}
		st.options = markdown.Options{Theme: &st.theme, OnLink: a.openWebLink}
		a.transcripts[v.ID] = st
	}
	if !st.loading && (!st.read || s.Updated.After(st.readAt)) {
		a.readTranscript(st, s)
	}
	ui.Column(c.Key(v.ID)).Fill().Gap(0).Children(func() {
		a.transcriptHeader(c, project, s, st)
		if st.err != nil {
			ui.Text(c, st.err.Error()).TextColor(a.theme.Danger).Padding(16, 24)
			return
		}
		if !st.read {
			ui.Text(c, a.text("Reading…", "正在读取……")).TextColor(a.theme.TextMuted).Padding(16, 24)
			return
		}
		st.theme.FontSize = a.textSize()
		messages := st.messages
		ui.List(c, &st.list, len(messages), func(i int) {
			ui.Column(c).FillWidth().AlignItems(ui.Center).Padding(7, 32).Children(func() {
				ui.Column(c).FillWidth().MaxWidth(760).Gap(10).Children(func() {
					a.transcriptMessage(c, agent.Kind(s.Agent), messages[i], st)
				})
			})
		}).Grow(1).FillWidth()
	})
}

func (a *app) readTranscript(st *transcriptState, s history.Session) {
	st.loading = true
	var messages []history.Message
	var readAt time.Time
	var err error
	a.background(func() {
		info, statErr := os.Stat(s.Path)
		if statErr != nil {
			err = statErr
			return
		}
		readAt = info.ModTime()
		messages, err = history.Read(s)
	}, func() {
		st.loading, st.read, st.err = false, true, err
		if err != nil {
			st.readAt = s.Updated // tried: read again only when it changes
			return
		}
		st.messages, st.readAt = messages, readAt
	})
}

func (a *app) transcriptHeader(c *ui.Context, project *layout.Project, s history.Session, st *transcriptState) {
	ui.Row(c).FillWidth().Gap(16).AlignItems(ui.Start).Padding(16, 24, 14).Children(func() {
		ui.Column(c).Grow(1).MinWidth(0).Gap(6).Children(func() {
			ui.Text(c, transcriptTitle(a, s)).FontSize(17).Bold().MinWidth(0).SingleLine().Ellipsis("…")
			ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
				a.agentIcon(c, s.Agent, 9)
				muted := func(text string) {
					ui.Text(c, text).FontSize(theme.Caption).TextColor(a.theme.TextMuted).MinWidth(0).SingleLine().Ellipsis("…")
				}
				muted(agentName(agent.Kind(s.Agent)))
				if s.Branch != "" {
					ui.Text(c, s.Branch).FontSize(theme.Caption).SingleLine().Padding(1, 6).Radius(4).Background(a.theme.SurfaceHover)
				}
				if rel, err := filepath.Rel(project.CWD, s.CWD); err == nil && rel != "." && filepath.IsLocal(rel) {
					muted(rel)
				}
				if !s.Start.IsZero() {
					muted(fmt.Sprintf(a.text("started %s · last %s", "开始 %s · 最后 %s"), when(s.Start), when(s.Updated)))
				}
				if st.read && st.err == nil {
					muted(fmt.Sprintf(a.text("%d messages", "%d 条消息"), len(st.messages)))
				}
				ui.Text(c, s.ID).FontSize(theme.Caption).Font("monospace").TextColor(a.theme.TextMuted).
					MinWidth(0).SingleLine().Ellipsis("…").Selectable()
				if iconButton(c, copyIcon, a.text("Copy Session ID", "复制会话 ID"), 20, 12).Clicked() {
					mygo.Clipboard.WriteText(s.ID)
				}
			})
		})
		ui.PrimaryButton(c, a.text("Resume", "Resume")).Shrink(0).
			Tooltip(a.text("Continue it in a new terminal, in its directory", "在新终端里、在它的目录中继续")).
			OnClick(func() { a.resumeSession(project.Name, s) })
	})
	ui.Box(c).FillWidth().Height(1).Shrink(0).Background(a.theme.Border)
}

func (a *app) transcriptMessage(c *ui.Context, kind agent.Kind, m history.Message, st *transcriptState) {
	if m.Steps > 0 {
		steps := a.text("· %d tool steps", "· %d 步工具调用")
		if m.Steps == 1 {
			steps = a.text("· %d tool step", "· %d 步工具调用")
		}
		ui.Text(c, fmt.Sprintf(steps, m.Steps)).
			FontSize(theme.Caption).TextColor(a.theme.TextMuted).Padding(0, 2)
	}
	if m.User {
		ui.Column(c).FillWidth().Gap(4).Padding(10, 14).Radius(8).Background(a.theme.Surface).Children(func() {
			ui.Text(c, a.text("You", "你")).FontSize(theme.Caption).Bold().TextColor(a.theme.Accent)
			ui.Text(c, m.Text).FontSize(a.textSize()).FillWidth().Selectable()
		})
		return
	}
	ui.Column(c).FillWidth().Gap(4).Padding(0, 14).Children(func() {
		ui.Text(c, agentName(kind)).FontSize(theme.Caption).Bold().TextColor(a.theme.TextMuted)
		markdown.View(c, m.Text, &st.options).FillWidth()
	})
}

// openWebLink opens a link of an answer in the browser, if it is a web
// page's.
func (a *app) openWebLink(dest string) {
	if !ui.SafeToOpen(dest) {
		return
	}
	if err := mygo.Shell.OpenExternal(dest); err != nil {
		a.report(err)
	}
}

func transcriptTitle(a *app, s history.Session) string {
	if s.Title != "" {
		return s.Title
	}
	return a.text("Session", "会话")
}

// when is a time of today as its time of day, else with its date.
func when(t time.Time) string {
	t = t.Local()
	if t.Format(time.DateOnly) == time.Now().Format(time.DateOnly) {
		return t.Format("15:04")
	}
	return t.Format("01-02 15:04")
}
