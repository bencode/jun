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

func TestParseBusy(t *testing.T) {
	output := `
70001 70001 70001 ttys009  -zsh
70100 70001 70300 ttys010  -zsh
70300 70300 70300 ttys010  /usr/local/bin/codex
70301 70300 70300 ttys010  node
70400 70400 70500 ttys011  -zsh
70600 70600 70600 ttys012  sleep
`
	shells := map[string]int{"ttys009": 70001, "ttys010": 70100, "ttys011": 70400}
	// Idle (009), its leader gone (011), and not one of Jun's terminals (012).
	want := map[string]string{"ttys010": "codex"}
	if got := ParseBusy(output, shells); !maps.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}
