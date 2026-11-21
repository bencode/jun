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
	"time"
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
		{"help", "me"}, {"terminal"}, {"terminals", "1"}, {"watch", "1"}, {"screen"}, {"screen", "0"}, {"screen", "x"},
		{"screen", "1", "--lines"}, {"screen", "1", "--lines", "0"}, {"screen", "1", "--lines", "501"}, {"screen", "1", "-n", "5"},
		{"open"}, {"open", "7"}, {"open", "terminal:0"}, {"open", "terminal:x"}, {"open", "project: "}, {"open", "note:a"},
		{"open", "peek"}, {"open", "peek:1"}, {"open", "diff:7", "--peek"}, {"open", "file:"}, {"open", "file:a", "--peek"}, {"open", "file:a", "--in", "Review"}, {"close", "file:"}, {"open", "terminal:7", "--in", "Review"},
		{"open", "diff:7", "--in"}, {"open", "diff:7", "--in", "A", "--in", "B"}, {"open", "terminal:7", "--peek", "--peek"},
		{"open", "terminal:7", "extra"}, {"close"}, {"close", "project:Jun"}, {"close", "peek", "1"}, {"back", "1"}, {"forward", "x"},
		{"send"}, {"send", "3"}, {"send", "0", "hi"}, {"send", "3", " \n"}, {"send", "3", "hi", "there"},
		{"send", "3", strings.Repeat("x", MaxText+1)},
		{"message", " \n"}, {"message", "a", "b"}, {"message", "--about", "3", "hi"}, {"message", "--title"},
		{"message", "--help"}, {"message", "--kind", "ask"}, {"message", "--link", "terminal:3"}, {"message", strings.Repeat("x", MaxMessage+1)},
		{"message", "--title", " ", "hi"}, {"message", "--title", "a\nb", "hi"}, {"message", "--title", strings.Repeat("x", MaxTitle+1), "hi"},
		{"message", "--title", "a", "--title", "b", "hi"}, {"message", "--kind", "urgent", "hi"},
		{"message", "--link", "peek", "hi"}, {"message", "--link", "perspective:Review", "hi"}, {"message", "--link", "terminal:0", "hi"},
		{"message", "update", "0", "hi"}, {"message", "update", "x", "hi"}, {"message", "update", "3"}, {"message", "update", "3", " \n"},
		{"message", "update", "3", "--kind"}, {"message", "update", "3", "--kind", "urgent"}, {"message", "update", "3", "a", "b"},
		{"message", "update", "3", "--title", "a", "b"}, {"message", "update", "3", "-h"}, {"message", "update", "3", strings.Repeat("x", MaxMessage+1)},
		{"run"}, {"run", "--open"}, {"run", "--dir"}, {"run", "--dir", "", "x"}, {"run", "--open", "--open", "x"}, {"run", "--every", "x"},
		{"loop", "5"}, {"loop", "30s", "x"}, {"loop", "90.5s", "x"}, {"loop", "5m"}, {"loop", "5m", "--open", "x"},
		{"loop", "cancel"}, {"loop", "cancel", "0"}, {"loop", "cancel", "1", "2"},
		{"run", "--project"}, {"run", "--project", " ", "x"}, {"run", "--project", "a", "--project", "b", "x"}, {"loop", "5m", "--project", "a", "x"},
		{"roles", "42"}, {"roles", "a", "b"}, {"roles", "--mine", "x"}, {"roles", "a", "--limit"}, {"roles", "a", "--limit", "0"}, {"roles", "a", "--limit", "x"},
		{"roles", "--open", "x"}, {"roles", "--team"}, {"roles", "--team", "a", "b"}, {"roles", "--team", "42"},
		{"join"}, {"join", "0"}, {"join", "x"}, {"join", "1", "2"}, {"leave", "1"},
		{"terminals", "update"}, {"terminals", "update", "3"}, {"terminals", "update", "3", "达芬奇"}, {"terminals", "update", "0", "--name", "a"},
		{"terminals", "update", "x", "--name", "a"}, {"terminals", "update", "3", "--name", "42"}, {"terminals", "update", "3", "--name", "a\nb"},
		{"terminals", "update", "3", "--name", strings.Repeat("x", MaxName+1)}, {"terminals", "update", "3", "--name", "a", "b"},
		{"terminals", "name", "3", "--name", "a"}, {"terminals", "update", "3", "--name", "a\tb"}, {"terminals", "update", "3", "--name", "a\x1b[2mb"},
		{"terminals", "update", "3", "--every", "30s"}, {"terminals", "update", "3", "--every", "90.5s"}, {"terminals", "update", "3", "--every"},
		{"terminals", "update", "3", "--at", "25:00"}, {"terminals", "update", "3", "--at", "9"}, {"terminals", "update", "3", "--text", "hi"}, {"terminals", "update", "3", "--text", ""},
		{"terminals", "update", "3", "--no-timer", "--every", "5m"}, {"terminals", "update", "3", "--at", "09:00", "--no-timer"},
		{"terminals", "update", "3", "--every", "5m", "--every", "1h"}, {"terminals", "update", "3", "--no-timer", "--no-timer"},
		{"terminals", "update", "3", "--every", "5m", "--text", "a\nb"},
		{"terminals", "update", "3", "--every", "5m", "--text", strings.Repeat("x", MaxTimerText+1)},
	} {
		if code, _, stderr := run("/nowhere", args...); code != 2 || !strings.Contains(stderr, usage) {
			t.Errorf("%q: exit %d, %q", args, code, stderr)
		}
	}
	// Standard input beyond the limit is refused, not cut to it.
	if code, _, stderr := runWith("/nowhere", strings.Repeat("x", MaxText)+"\n\n\n", "send", "3", "-"); code != 2 || !strings.Contains(stderr, "at most") {
		t.Errorf("long standard input: exit %d, %q", code, stderr)
	}
	if code, _, stderr := run("", "terminals"); code != 1 || !strings.Contains(stderr, EnvAPI+" is not set") || !strings.Contains(stderr, EnvCommand) {
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
		if req.Op == OpRun || req.Op == OpLoop {
			return Response{Terminal: 12}
		}
		if req.Op == OpLoops {
			return Response{Loops: []Loop{{ID: 12, Every: "5m", Command: "echo hi"}}}
		}
		if req.Op == OpMessage {
			return Response{Created: 7}
		}
		if req.Op == OpMessages {
			return Response{Messages: []Message{{ID: 7, Kind: MessageAsk, Link: "terminal:3", Replies: []Reply{{Text: "Answered"}}}}}
		}
		if req.Op == OpRoles {
			return Response{Roles: []Role{{Name: "欧拉", Count: 1, Last: &RoleSession{Agent: "codex", ID: "t1"}}}}
		}
		if req.Op != OpTerminals {
			return Response{}
		}
		return Response{Terminals: []Terminal{{ID: 3, Project: "Jun", Agent: "claude", Status: "idle", Self: true}}}
	}, nil)
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

	here, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
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
	// A caller outside Jun's terminals names its own, from JUN_TERMINAL.
	named := func(value string) (int, string) {
		var stderr bytes.Buffer
		code := Main([]string{"back"}, strings.NewReader(""), &bytes.Buffer{}, &stderr, func(name string) string {
			return map[string]string{EnvAPI: path, EnvTerminal: value}[name]
		})
		return code, stderr.String()
	}
	if code, stderr := named("5"); code != 0 {
		t.Fatalf("named terminal: %d %q", code, stderr)
	}
	if code, stderr := named("five"); code != 2 || !strings.Contains(stderr, "is not a terminal id") {
		t.Fatalf("a name that is no id: %d %q", code, stderr)
	}
	if code, _, _ := run(path, "send", "3", "a\nb\n"); code != 0 {
		t.Fatalf("send: %d", code)
	}
	if code, _, _ := runWith(path, "x\n", "send", "3", "-"); code != 0 {
		t.Fatalf("send from standard input: %d", code)
	}
	if code, out, _ := run(path, "message", "--title", "Asks", "--kind", "ask", "--link", "terminal:3", "It waits"); code != 0 || out != "7\n" {
		t.Fatalf("message: %d %q", code, out)
	}
	var messages []Message
	if code, out, _ := run(path, "message"); code != 0 || json.Unmarshal([]byte(out), &messages) != nil || len(messages) != 1 || messages[0].Replies[0].Text != "Answered" {
		t.Fatalf("messages: %d %q", code, out)
	}
	if code, out, _ := run(path, "message", "update", "7", "--kind", "done", "Answered"); code != 0 || out != "" {
		t.Fatalf("update: %d %q", code, out)
	}
	if code, _, _ := run(path, "message", "update", "7", "--kind", "done"); code != 0 {
		t.Fatalf("update of the kind alone: %d", code)
	}
	if code, _, _ := runWith(path, "a\nb\n", "message", "update", "7", "-"); code != 0 {
		t.Fatalf("update from standard input: %d", code)
	}
	if code, _, _ := run(path, "message", "update"); code != 0 {
		t.Fatalf("the one word update is a text: %d", code)
	}
	if code, _, _ := run(path, "message", "--title", "Done"); code != 0 {
		t.Fatalf("message with a title alone: %d", code)
	}
	if code, _, _ := run(path, "message", "-- deploy failed"); code != 0 {
		t.Fatalf("message starting with dashes: %d", code)
	}
	if code, _, _ := runWith(path, "\n", "message", "--title", "Quiet", "-"); code != 0 {
		t.Fatalf("a title over an empty standard input: %d", code)
	}
	if code, _, _ := runWith(path, "a\nb\n", "message", "-"); code != 0 {
		t.Fatalf("message from standard input: %d", code)
	}
	if code, out, _ := run(path, "run", "--open", "codex", "--full-auto"); code != 0 || out != "12\n" {
		t.Fatalf("run: %d %q", code, out)
	}
	for _, args := range [][]string{{"run", "--project", " Web ", "codex"}, {"run", "--project", "web", "--dir", "sub", "codex"}} {
		if code, out, _ := run(path, args...); code != 0 || out != "12\n" {
			t.Fatalf("%q: %d %q", args, code, out)
		}
	}
	for _, args := range [][]string{{"join", "3"}, {"leave"}} {
		if code, out, _ := run(path, args...); code != 0 || out != "" {
			t.Fatalf("%q: %d %q", args, code, out)
		}
	}
	if code, out, _ := run(path, "loop", "5m", "--dir", "sub", "echo", "hi"); code != 0 || out != "12\n" {
		t.Fatalf("loop: %d %q", code, out)
	}
	var loops []Loop
	if code, out, _ := run(path, "loop"); code != 0 || json.Unmarshal([]byte(out), &loops) != nil || len(loops) != 1 {
		t.Fatalf("loops: %d %q", code, out)
	}
	if code, out, _ := run(path, "loop", "cancel", "12"); code != 0 || out != "" {
		t.Fatalf("loop cancel: %d %q", code, out)
	}
	for _, args := range [][]string{{"terminals", "update", "3", "--name", " 达芬奇 "}, {"terminals", "update", "3", "--name", ""},
		{"terminals", "update", "3", "--text", "看看新工单", "--every", "30m", "--name", "欧拉"}, {"terminals", "update", "3", "--no-timer"}} {
		if code, out, _ := run(path, args...); code != 0 || out != "" {
			t.Fatalf("%q: %d %q", args, code, out)
		}
	}
	// A name goes with the caller's agent sessions, as their variables tell.
	withSessions := func(args ...string) int {
		return Main(args, strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{}, func(name string) string {
			return map[string]string{EnvAPI: path, "CODEX_THREAD_ID": "t1", "CODEX_HOME": "/h/codex", "CLAUDE_CODE_SESSION_ID": "c1",
				"PI_SESSION_ID": "p1", "PI_SESSION_FILE": "/h/pi/p1.jsonl"}[name]
		})
	}
	if withSessions("terminals", "update", "3", "--name", "欧拉") != 0 || withSessions("terminals", "update", "3", "--every", "5m") != 0 {
		t.Fatal("update with sessions")
	}
	var roles []Role
	if code, out, _ := run(path, "roles"); code != 0 || json.Unmarshal([]byte(out), &roles) != nil || len(roles) != 1 || roles[0].Last.ID != "t1" {
		t.Fatalf("roles: %d %q", code, out)
	}
	for _, args := range [][]string{{"roles", " 欧拉 "}, {"roles", "欧拉", "--limit", "3"}, {"roles", "--open"}, {"roles", "--team", "欧拉"}} {
		if code, _, _ := run(path, args...); code != 0 {
			t.Fatalf("%q: %d", args, code)
		}
	}
	if withSessions("roles", "--mine") != 0 {
		t.Fatal("roles --mine")
	}
	if code, out, _ := run(path, "open", "file:docs/a.md"); code != 0 || out != "" {
		t.Fatalf("open file: %d %q", code, out)
	}
	sessions := Sessions{Claude: AgentSession{ID: "c1"}, Codex: AgentSession{ID: "t1", Home: "/h/codex"},
		Pi: AgentSession{ID: "p1", Path: "/h/pi/p1.jsonl"}}
	want := []Request{{Op: OpTerminals}, {Op: OpScreen, Terminal: 3, Lines: 2}, {Op: OpScreen, Terminal: 9, Lines: DefaultLines},
		{Op: OpOpen, Target: &Target{Kind: TargetDiff, ID: 7}, In: "Review"},
		{Op: OpClose, Target: &Target{Kind: TargetPerspective, Name: "Review"}}, {Op: OpBack}, {Op: OpBack, Caller: 5},
		{Op: OpSend, Terminal: 3, Text: "a\nb"}, {Op: OpSend, Terminal: 3, Text: "x"},
		{Op: OpMessage, Title: "Asks", Kind: MessageAsk, Target: &Target{Kind: TargetTerminal, ID: 3}, Text: "It waits"},
		{Op: OpMessages}, {Op: OpUpdateMessage, Message: 7, Kind: MessageDone, Text: "Answered"}, {Op: OpUpdateMessage, Message: 7, Kind: MessageDone},
		{Op: OpUpdateMessage, Message: 7, Text: "a\nb"}, {Op: OpMessage, Text: "update"},
		{Op: OpMessage, Title: "Done"}, {Op: OpMessage, Text: "-- deploy failed"}, {Op: OpMessage, Title: "Quiet"}, {Op: OpMessage, Text: "a\nb"},
		{Op: OpRun, Command: "codex --full-auto", Dir: here, Open: true},
		{Op: OpRun, Command: "codex", Project: "Web"}, {Op: OpRun, Command: "codex", Project: "web", Dir: filepath.Join(here, "sub")},
		{Op: OpJoin, Terminal: 3}, {Op: OpLeave},
		{Op: OpLoop, Command: "echo hi", Dir: filepath.Join(here, "sub"), Every: 5 * time.Minute},
		{Op: OpLoops}, {Op: OpCancelLoop, Terminal: 12},
		{Op: OpUpdateTerminal, Terminal: 3, Name: ptr("达芬奇")}, {Op: OpUpdateTerminal, Terminal: 3, Name: ptr("")},
		{Op: OpUpdateTerminal, Terminal: 3, Name: ptr("欧拉"), Every: 30 * time.Minute, Text: "看看新工单"},
		{Op: OpUpdateTerminal, Terminal: 3, NoTimer: true},
		{Op: OpUpdateTerminal, Terminal: 3, Name: ptr("欧拉"), Sessions: sessions},
		{Op: OpUpdateTerminal, Terminal: 3, Every: 5 * time.Minute},
		{Op: OpRoles}, {Op: OpRoles, Name: ptr("欧拉"), Limit: DefaultRoleSessions}, {Op: OpRoles, Name: ptr("欧拉"), Limit: 3},
		{Op: OpRoles, Away: true}, {Op: OpRoles, Team: ptr("欧拉")},
		{Op: OpRoles, Mine: true, Limit: DefaultRoleSessions, Sessions: sessions},
		{Op: OpOpen, Target: &Target{Kind: TargetFile, Name: filepath.Join(here, "docs", "a.md")}}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("requests %+v", got)
	}

	// A second Jun on the socket is refused while the first answers.
	if _, err := Start(path, nil, nil); err == nil || !strings.Contains(err.Error(), "another Jun") {
		t.Fatalf("second server: %v", err)
	}
}

func TestWatchPrintsEventsUntilJunCloses(t *testing.T) {
	path := socketPath(t)
	events := make(chan Event, 2)
	canceled := make(chan struct{})
	s, err := Start(path, nil, func() (<-chan Event, func()) { return events, func() { close(canceled) } })
	if err != nil {
		t.Fatal(err)
	}
	events <- Event{ID: 3, Project: "Jun", Agent: "claude", Title: "Fix", To: "working"}
	events <- Event{ID: 3, Project: "Jun", Agent: "claude", Title: "Fix", From: "working", To: "waiting"}

	stdout := lineWriter{lines: make(chan string, 2)}
	code := make(chan int, 1)
	go func() {
		code <- Main([]string{"watch"}, strings.NewReader(""), stdout, &bytes.Buffer{}, func(string) string { return path })
	}()
	// Each event arrives as it happens, not at the end.
	for _, want := range []string{`"from":""`, `"from":"working","to":"waiting"`} {
		if line := <-stdout.lines; !strings.Contains(line, want) {
			t.Fatalf("event %s, want %s", line, want)
		}
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if got := <-code; got != 1 {
		t.Fatalf("exit %d", got)
	}
	<-canceled
}

func TestWatchEndsForAReaderThatFellBehind(t *testing.T) {
	path := socketPath(t)
	events := make(chan Event)
	close(events)
	s, err := Start(path, nil, func() (<-chan Event, func()) { return events, func() {} })
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := s.Close(); err != nil {
			t.Error(err)
		}
	}()
	if code, out, stderr := run(path, "watch"); code != 1 || out != "" || !strings.Contains(stderr, "the watch ended") {
		t.Fatalf("exit %d, %q, %q", code, out, stderr)
	}
}

type lineWriter struct{ lines chan string }

func (w lineWriter) Write(p []byte) (int, error) {
	w.lines <- string(p)
	return len(p), nil
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
	s, err := Start(path, func(Request, []int) Response { return Response{} }, nil)
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
	if _, err := Start(path, nil, nil); err == nil {
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

func ptr(s string) *string { return &s }

func TestAtIsTheNextTimeTheClockReadsSo(t *testing.T) {
	now := time.Date(2026, 11, 20, 14, 30, 0, 0, time.Local)
	for clock, want := range map[string]time.Time{
		"15:00": time.Date(2026, 11, 20, 15, 0, 0, 0, time.Local),
		"14:30": time.Date(2026, 11, 21, 14, 30, 0, 0, time.Local), // now is past
		"09:05": time.Date(2026, 11, 21, 9, 5, 0, 0, time.Local),
	} {
		if at, err := nextAt(now, clock); err != nil || !at.Equal(want) {
			t.Errorf("%s: %v %v, want %v", clock, at, err, want)
		}
	}
	// --at goes on its own, or first before --every.
	req, err := parse([]string{"terminals", "update", "3", "--at", "15:00", "--every", "1h"}, nil)
	if err != nil || req.At.IsZero() || !req.At.After(time.Now()) || req.Every != time.Hour {
		t.Fatalf("--at with --every: %+v %v", req, err)
	}
}
