package agent

import (
	"strings"
	"testing"
)

// The screens follow snapshots of the agents' terminals: a row each,
// separated by "\r\n", styles reset before each run.
func vt(rows ...string) []byte {
	return []byte(strings.Join(rows, "\r\n") + "\x1b[5;3H\x1b[0m")
}

const (
	rule     = "\x1b[0m\x1b[38;2;136;136;136m──────────\x1b[0m"
	piRule   = "\x1b[0m\x1b[38;5;5m──────────\x1b[0m"
	labelled = "\x1b[0m\x1b[38;2;136;136;136m────── fix-parser ─\x1b[0m"
)

func TestHasDraft(t *testing.T) {
	claudeStatus := "  ➜ jun git:(main) | Opus | tokens:0k"
	cases := []struct {
		name   string
		kind   Kind
		screen []byte
		want   bool
	}{
		{"claude empty", Claude, vt("⏺ Done.", "", rule, "❯ ", rule, claudeStatus), false},
		{"claude placeholder", Claude, vt("", rule, "❯ \x1b[0m\x1b[2mTry \"edit app.go to…\"\x1b[0m", rule, claudeStatus), false},
		{"claude suggestion, in 256 colors and dim", Claude, vt("", labelled, "❯ \x1b[0m\x1b[38;5;246;2mrun the tests\x1b[0m", rule, claudeStatus), false},
		{"claude draft", Claude, vt("", rule, "❯ hi draft", rule, claudeStatus), true},
		{"claude draft under a labelled rule", Claude, vt("", labelled, "❯ 可以的，提PR", rule, claudeStatus), true},
		{"claude draft on its second line", Claude, vt("", rule, "❯ ", "  and then", rule, claudeStatus), true},
		{"claude draft in bold", Claude, vt("", rule, "❯ \x1b[0m\x1b[1;2m\x1b[22mhi\x1b[0m", rule, claudeStatus), true},
		{"claude shell mode, empty", Claude, vt("", rule, "\x1b[0m\x1b[38;2;253;93;177m!\x1b[0m ", rule, "\x1b[0m\x1b[2m  ! for shell mode\x1b[0m"), false},
		{"claude shell mode draft", Claude, vt("", rule, "\x1b[0m\x1b[38;2;253;93;177m!\x1b[0m ls", rule), true},
		{"pi draft that starts like a prompt", Pi, vt("", piRule, "!", piRule), true},
		{"claude no input box", Claude, vt("Do you want to proceed?", "❯ 1. Yes", "  2. No"), false},
		{"codex placeholder", Codex, vt("› fix the bug", "", "• Fixed.", "", "\x1b[0m\x1b[1m›\x1b[0m \x1b[0m\x1b[2mAsk Codex to do anything\x1b[0m", "", "  ? for shortcuts"), false},
		{"codex draft", Codex, vt("› fix the bug", "", "• Fixed.", "", "\x1b[0m\x1b[1m›\x1b[0m hi draft", "", "  tab to queue message"), true},
		{"pi empty", Pi, vt("Done.", piRule, "", piRule, "\x1b[0m\x1b[2m~/work/jun (main)\x1b[0m"), false},
		{"pi working, empty", Pi, vt("── ⠦ Working ──", "", piRule, "\x1b[0m\x1b[2m~/work/jun (main)\x1b[0m"), false},
		{"pi draft", Pi, vt("Done.", piRule, "hi draft\x1b[0m\x1b[7m \x1b[0m", piRule, "\x1b[0m\x1b[2m~/work/jun (main)\x1b[0m"), true},
		{"strings and colors with colons skipped", Claude, vt("\x1b]8;;https://x\x07link\x1b]8;;\x1b\\", rule, "❯ \x1b[0m\x1b[38:2::1:2:3;2mhint\x1b[0m", rule), false},
	}
	for _, c := range cases {
		if got := HasDraft(c.kind, c.screen); got != c.want {
			t.Errorf("%s: %v, want %v\n%s", c.name, got, c.want, undimmed(c.screen))
		}
	}
}
