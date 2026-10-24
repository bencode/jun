package app

import (
	"fmt"

	"github.com/bencode/jun/internal/layout"
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
	if v.Kind == layout.ViewAgents {
		return a.kindName(v.Kind)
	}
	return fmt.Sprintf("%s %d", a.kindName(v.Kind), v.ID)
}

func (a *app) perspectiveName(kind layout.PerspectiveKind) string {
	if kind == layout.PerspectiveReview {
		return a.text("Review", "审阅")
	}
	return a.text("Work", "工作")
}
