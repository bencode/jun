package tool

import (
	"bytes"
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
)

// socketPath is short: a socket's path has a limit that t.TempDir's
// exceed on macOS.
func socketPath(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "jt")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Error(err)
		}
	})
	return filepath.Join(dir, "jun.sock")
}

func run(path string, args ...string) (int, string, string) {
	return runWith(path, "", args...)
}

func runWith(path, stdin string, args ...string) (int, string, string) {
	var stdout, stderr bytes.Buffer
	code := Main(args, strings.NewReader(stdin), &stdout, &stderr, func(name string) string {
		if name == EnvAPI {
			return path
		}
		return ""
	})
	return code, stdout.String(), stderr.String()
}

func TestMainUsage(t *testing.T) {
	if code, out, _ := run(""); code != 2 || out != "" {
		t.Fatalf("no command: %d %q", code, out)
	}
	if code, out, _ := run("", "help"); code != 0 || !strings.Contains(out, "jun tool screen <id>") {
		t.Fatalf("help without Jun: %d", code)
	}
	for _, args := range [][]string{
		{"help", "me"}, {"terminal"}, {"terminals", "1"}, {"screen"}, {"screen", "0"}, {"screen", "x"},
		{"screen", "1", "--lines"}, {"screen", "1", "--lines", "0"}, {"screen", "1", "--lines", "501"}, {"screen", "1", "-n", "5"},
		{"open"}, {"open", "7"}, {"open", "terminal:0"}, {"open", "terminal:x"}, {"open", "project: "}, {"open", "file:a"},
		{"open", "peek"}, {"open", "peek:1"}, {"open", "diff:7", "--peek"}, {"open", "terminal:7", "--in", "Review"},
		{"open", "diff:7", "--in"}, {"open", "diff:7", "--in", "A", "--in", "B"}, {"open", "terminal:7", "--peek", "--peek"},
		{"open", "terminal:7", "extra"}, {"close"}, {"close", "project:Jun"}, {"close", "peek", "1"}, {"back", "1"}, {"forward", "x"},
		{"send"}, {"send", "3"}, {"send", "0", "hi"}, {"send", "3", " \n"}, {"send", "3", "hi", "there"},
		{"send", "3", strings.Repeat("x", MaxText+1)},
	} {
		if code, _, stderr := run("/nowhere", args...); code != 2 || !strings.Contains(stderr, usage) {
			t.Errorf("%q: exit %d, %q", args, code, stderr)
		}
	}
	// Standard input beyond the limit is refused, not cut to it.
	if code, _, stderr := runWith("/nowhere", strings.Repeat("x", MaxText)+"\n\n\n", "send", "3", "-"); code != 2 || !strings.Contains(stderr, "at most") {
		t.Errorf("long standard input: exit %d, %q", code, stderr)
	}
	if code, _, stderr := run("", "terminals"); code != 1 || !strings.Contains(stderr, EnvAPI+" is not set") {
		t.Fatalf("without JUN_API: %d %q", code, stderr)
	}
	if code, _, stderr := run(socketPath(t), "terminals"); code != 1 || !strings.Contains(stderr, "cannot reach Jun") {
		t.Fatalf("without Jun: %d %q", code, stderr)
	}
}

func TestToolsAnswerOverTheSocket(t *testing.T) {
	path := socketPath(t)
	var got []Request
	var callers []int
	s, err := Start(path, func(req Request, c []int) Response {
		got, callers = append(got, req), c
		if req.Op == OpScreen && req.Terminal == 9 {
			return Response{Error: "no terminal 9"}
		}
		if req.Op == OpScreen {
			return Response{Screen: "a\nb\n"}
		}
		if req.Op == OpClose {
			return Response{Message: "waiting for the user to confirm"}
		}
		if req.Op != OpTerminals {
			return Response{}
		}
		return Response{Terminals: []Terminal{{ID: 3, Project: "Jun", Agent: "claude", Status: "idle", Self: true}}}
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Error(err)
		}
	})
	if info, err := os.Stat(path); err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("socket mode: %v %v", info, err)
	}

	code, out, _ := run(path, "terminals")
	var terminals []Terminal
	if err := json.Unmarshal([]byte(out), &terminals); code != 0 || err != nil || len(terminals) != 1 || !terminals[0].Self {
		t.Fatalf("terminals: %d %q %v", code, out, err)
	}
	// The caller is this process, by the system's word, with its ancestors
	// before launchd.
	ppid := os.Getppid()
	if len(callers) == 0 || callers[0] != os.Getpid() || ppid > 1 && !slices.Contains(callers, ppid) {
		t.Fatalf("callers %v, want %d then its ancestors", callers, os.Getpid())
	}
	if code, out, _ := run(path, "screen", "3", "--lines", "2"); code != 0 || out != "a\nb\n" {
		t.Fatalf("screen: %d %q", code, out)
	}
	if code, _, stderr := run(path, "screen", "9"); code != 1 || stderr != "jun tool: no terminal 9\n" {
		t.Fatalf("refused: %d %q", code, stderr)
	}
	if code, out, _ := run(path, "open", "diff:7", "--in", " Review "); code != 0 || out != "" {
		t.Fatalf("open: %d %q", code, out)
	}
	if code, out, _ := run(path, "close", "perspective:Review"); code != 0 || out != "waiting for the user to confirm\n" {
		t.Fatalf("close: %d %q", code, out)
	}
	if code, out, _ := run(path, "back"); code != 0 || out != "" {
		t.Fatalf("back: %d %q", code, out)
	}
	if code, _, _ := run(path, "send", "3", "a\nb\n"); code != 0 {
		t.Fatalf("send: %d", code)
	}
	if code, _, _ := runWith(path, "x\n", "send", "3", "-"); code != 0 {
		t.Fatalf("send from standard input: %d", code)
	}
	want := []Request{{Op: OpTerminals}, {Op: OpScreen, Terminal: 3, Lines: 2}, {Op: OpScreen, Terminal: 9, Lines: DefaultLines},
		{Op: OpOpen, Target: &Target{Kind: TargetDiff, ID: 7}, In: "Review"},
		{Op: OpClose, Target: &Target{Kind: TargetPerspective, Name: "Review"}}, {Op: OpBack},
		{Op: OpSend, Terminal: 3, Text: "a\nb"}, {Op: OpSend, Terminal: 3, Text: "x"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("requests %+v", got)
	}

	// A second Jun on the socket is refused while the first answers.
	if _, err := Start(path, nil); err == nil || !strings.Contains(err.Error(), "another Jun") {
		t.Fatalf("second server: %v", err)
	}
}

func TestStartReplacesAStaleSocket(t *testing.T) {
	path := socketPath(t)
	stale, err := net.ListenUnix("unix", &net.UnixAddr{Name: path, Net: "unix"})
	if err != nil {
		t.Fatal(err)
	}
	stale.SetUnlinkOnClose(false) // as a Jun that crashed leaves it
	if err := stale.Close(); err != nil {
		t.Fatal(err)
	}
	s, err := Start(path, func(Request, []int) Response { return Response{} })
	if err != nil {
		t.Fatalf("stale socket: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Close left the socket: %v", err)
	}
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Start(path, nil); err == nil {
		t.Fatal("Start replaced a file that is no socket")
	}
}

func TestAncestry(t *testing.T) {
	table := map[int]int{40: 30, 30: 20, 20: 1, 7: 8, 8: 7}
	parent := func(pid int) (int, error) {
		if p, ok := table[pid]; ok {
			return p, nil
		}
		return 0, errors.New("no such process")
	}
	if chain, err := ancestry(40, parent); err != nil || !reflect.DeepEqual(chain, []int{40, 30, 20}) {
		t.Fatalf("chain %v %v", chain, err)
	}
	if _, err := ancestry(50, parent); err == nil {
		t.Fatal("a process gone should fail")
	}
	if _, err := ancestry(7, parent); err == nil {
		t.Fatal("a cycle should fail")
	}
}

func TestTail(t *testing.T) {
	text := "one\ntwo, a long line the terminal wrapped\nthree\n\n  \n"
	if got := Tail(text, 2); got != "two, a long line the terminal wrapped\nthree\n" {
		t.Fatalf("tail 2: %q", got)
	}
	if got := Tail(text, 10); got != "one\ntwo, a long line the terminal wrapped\nthree\n" {
		t.Fatalf("tail 10: %q", got)
	}
	if got := Tail("\n \n", 5); got != "" {
		t.Fatalf("blank: %q", got)
	}
}
