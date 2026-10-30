package app

import (
	"fmt"
	"path"
	"regexp"
	"strings"

	"github.com/bencode/jun/internal/layout"
	"github.com/bencode/jun/internal/terminal"
)

func (a *app) text(en, zh string) string {
	if a.chinese {
		return zh
	}
	return en
}

func (a *app) kindName(kind layout.ViewKind) string {
	names := map[layout.ViewKind][2]string{
		layout.ViewTerminal: {"Terminal", "终端"}, layout.ViewTree: {"Files", "文件树"},
		layout.ViewDiff: {"Diff", "差异"}, layout.ViewFile: {"File", "文件"},
		layout.ViewAgents: {"Agents", "Agents"},
	}
	pair := names[kind]
	return a.text(pair[0], pair[1])
}

func (a *app) title(v *layout.View) string {
	if session := a.sessionFor(v.ID); session != nil {
		return a.sessionTitle(session)
	}
	if v.Kind == layout.ViewAgents {
		return a.kindName(v.Kind)
	}
	return fmt.Sprintf("%s %d", a.kindName(v.Kind), v.ID)
}

func (a *app) sessionTitle(s *terminal.Session) string {
	title := shortTitle(strings.TrimSpace(s.Terminal.Title()))
	if title == "" {
		title = fmt.Sprintf("%s %d", a.text("Terminal", "终端"), s.ID)
	}
	select {
	case <-s.Terminal.Done():
		title += fmt.Sprintf(a.text(" [exited %d]", " [已退出 %d]"), s.Terminal.ExitCode())
	default:
	}
	return title
}

var shellPrompt = regexp.MustCompile(`^[^@\s]+@[^:\s]+:(.+)$`)

// shortTitle keeps only the directory of a shell's "user@host:path" title;
// titles that programs set are kept as they are.
func shortTitle(title string) string {
	if m := shellPrompt.FindStringSubmatch(title); m != nil {
		return path.Base(m[1])
	}
	return title
}
