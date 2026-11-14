package tool

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMistakenCommand(t *testing.T) {
	wd := t.TempDir()
	if got := MistakenCommand([]string{"open", "file:/x/a.md"}, wd); !strings.Contains(got, `"$JUN_BIN" tool open file:/x/a.md`) {
		t.Errorf("jun open file: gave %q", got)
	}
	if got := MistakenCommand([]string{"send", "7", "run the tests, don't stop"}, wd); !strings.Contains(got, `tool send 7 'run the tests, don'\''t stop'`) {
		t.Errorf("jun send with spaces gave %q", got)
	}
	for _, args := range [][]string{nil, {"."}, {"/x/a.md"}, {"-psn_0_1"}} {
		if got := MistakenCommand(args, wd); got != "" {
			t.Errorf("MistakenCommand(%q) = %q, want none", args, got)
		}
	}
	if err := os.Mkdir(filepath.Join(wd, "run"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got := MistakenCommand([]string{"run"}, wd); got != "" {
		t.Errorf("a directory named run is mistaken: %q", got)
	}
}
