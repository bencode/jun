package agent

import (
	"maps"
	"testing"
)

func TestParseForeground(t *testing.T) {
	output := `
69910 69910 69910 ttys002  /Users/me/.local/bin/claude
12847 12847 12847 ttys000  codex
73453 73453 73453 ttys021  pi
70001 70001 70001 ttys009  -zsh
80002 80001 80001 ttys011  codex
 3001  3001     0 ??       claude
`
	want := map[string]Kind{"ttys002": Claude, "ttys000": Codex, "ttys021": Pi}
	if got := ParseForeground(output); !maps.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}
