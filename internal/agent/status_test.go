package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fixture(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestDetectStatus(t *testing.T) {
	cases := []struct {
		name   string
		kind   Kind
		title  string
		screen string
		want   Status
	}{
		{"claude braille spinner", Claude, "⠐ add-basic-validation-checks", fixture(t, "working.txt"), Working},
		{"claude two-frame spinner", Claude, "◐ 沙箱技术深度调研", "", Working},
		{"claude idle prompt", Claude, "✳ refactor-parser-module", fixture(t, "idle.txt"), Idle},
		{"claude permission dialog", Claude, "✳ some-task", fixture(t, "waiting.txt"), Waiting},
		{"claude numbered prose without cursor", Claude, "✳ fix", "Do you want to:\n1. Log in\n2. Hold\n───\n❯\n───", Idle},
		{"claude dialog without idle title", Claude, "project", fixture(t, "waiting.txt"), Idle},
		{"codex action required title", Codex, "[ . ] Action Required | yimi", "", Waiting},
		{"codex selection control", Codex, "⠹ yimi", fixture(t, "codex-waiting.txt"), Waiting},
		{"codex answered question", Codex, "yimi", "• Questions 1/1 answered\n\n› Explain this codebase", Idle},
		{"codex footer alone", Codex, "yimi", "enter to submit answer", Idle},
		{"codex spinner", Codex, "⠹ cc-mission-control", "", Working},
		{"pi working loader", Pi, "π - project", " ⠹ Working (esc to interrupt)\n───\n\n───", Working},
		{"pi ignores claude signals", Pi, "◐ custom", "❯ 1. Example", Idle},
		{"pi loader scrolled out of the tail", Pi, "π - project", "⠹ Working\n" + strings.Repeat("completed\n", 31), Idle},
	}
	for _, c := range cases {
		if got := DetectStatus(c.kind, c.title, c.screen); got != c.want {
			t.Errorf("%s: got %s, want %s", c.name, got, c.want)
		}
	}
}

func TestDisplayTitle(t *testing.T) {
	for title, want := range map[string]string{
		"✳ Claude Code":                "Claude Code",
		"⠹ cc-mission-control":         "cc-mission-control",
		"◐ 沙箱技术深度调研":                   "沙箱技术深度调研",
		"[ . ] Action Required | yimi": "yimi",
		"π - work":                     "π - work",
	} {
		if got := DisplayTitle(title); got != want {
			t.Errorf("DisplayTitle(%q) = %q, want %q", title, got, want)
		}
	}
}
