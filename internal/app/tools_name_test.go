package app

import (
	"fmt"
	"strings"
	"testing"

	"github.com/bencode/jun/internal/terminal"
	"github.com/bencode/jun/internal/tool"
)

func TestTerminalNamesAreUniqueWhileTheTerminalRuns(t *testing.T) {
	a, tt, sessions := agentsApp(t)
	first, second := sessions[0], sessions[1]
	name := func(s *terminal.Session, name string) error {
		return a.toolUpdateTerminal(tool.Request{Op: tool.OpUpdateTerminal, Terminal: uint64(s.ID), Name: name})
	}
	listed := func(id terminal.ID) string {
		t.Helper()
		for _, terminal := range a.toolTerminals(nil, 0) {
			if terminal.ID == uint64(id) {
				return terminal.Name
			}
		}
		t.Fatalf("terminal %d not listed", id)
		return ""
	}

	// Named, it is listed, told to watchers and shown in Agents.
	if err := name(first, "达芬奇"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if listed(first.ID) != "达芬奇" || a.agentEvents()[first.ID].Name != "达芬奇" || !tt.HasText("达芬奇") {
		t.Fatalf("named: listed %q, told %q, shown %v", listed(first.ID), a.agentEvents()[first.ID].Name, tt.HasText("达芬奇"))
	}
	if listed(second.ID) != "" {
		t.Fatalf("an unnamed terminal listed as %q", listed(second.ID))
	}

	// Another running terminal's name is refused; its own again is not.
	if err := name(second, "达芬奇"); err == nil || err.Error() != fmt.Sprintf(`name "达芬奇" is terminal %d's`, first.ID) {
		t.Fatalf("a name taken: %v", err)
	}
	if err := name(first, "达芬奇"); err != nil {
		t.Fatalf("its own name again: %v", err)
	}
	// Taken in another case, it is taken.
	if err := name(first, "Leonardo"); err != nil {
		t.Fatal(err)
	}
	if err := name(second, "leonardo"); err == nil {
		t.Fatal("a name taken in another case is not refused")
	}
	// Jun checks a name as the command does, for a request from elsewhere.
	for _, bad := range []string{"42", "a\nb", " a", strings.Repeat("x", tool.MaxName+1)} {
		if err := name(second, bad); err == nil {
			t.Errorf("name %q taken", bad)
		}
	}
	// A new name replaces it, and frees the old; none drops it.
	if err := name(first, "欧拉"); err != nil || listed(first.ID) != "欧拉" {
		t.Fatalf("renamed: %v, %q", err, listed(first.ID))
	}
	if err := name(second, "达芬奇"); err != nil {
		t.Fatalf("the old name, freed: %v", err)
	}
	if err := name(first, ""); err != nil || listed(first.ID) != "" {
		t.Fatalf("dropped: %v, %q", err, listed(first.ID))
	}
	if err := name(&terminal.Session{ID: 999}, "高斯"); err == nil || err.Error() != "no terminal 999" {
		t.Fatalf("no such terminal: %v", err)
	}

	// A terminal that ended gives its name up.
	second.Terminal.Send([]byte("exit\n"))
	eventually(t, "the shell to exit", second.Exited)
	if err := name(first, "达芬奇"); err != nil {
		t.Fatalf("the name of a terminal that ended: %v", err)
	}
	a.reapExitedTerminals(0)
	if _, ok := a.names[second.ID]; ok {
		t.Fatal("the name of a closed terminal is kept")
	}
}
