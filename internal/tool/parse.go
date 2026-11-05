package tool

import (
	"cmp"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
)

func parse(args []string, stdin io.Reader) (Request, error) {
	if len(args) == 0 {
		return Request{}, errors.New("no command")
	}
	command, rest := args[0], args[1:]
	if command == "help" && len(rest) == 0 {
		return Request{}, errHelp
	}
	if command == "terminals" && len(rest) == 0 {
		return Request{Op: OpTerminals}, nil
	}
	if command == "watch" && len(rest) == 0 {
		return Request{Op: OpWatch}, nil
	}
	if (command == "back" || command == "forward") && len(rest) == 0 {
		return Request{Op: Op(command)}, nil
	}
	if command == "open" && len(rest) > 0 {
		return parseOpen(rest)
	}
	if command == "close" && len(rest) == 1 {
		return parseClose(rest[0])
	}
	if command == "send" {
		return parseSend(rest, stdin)
	}
	if command == "run" {
		return parseRun(rest)
	}
	if command == "loop" {
		return parseLoop(rest)
	}
	if command != "screen" {
		return Request{}, fmt.Errorf("unknown command or arguments: %q", args)
	}
	if len(rest) != 1 && (len(rest) != 3 || rest[1] != "--lines") {
		return Request{}, fmt.Errorf("screen takes a terminal id and optionally --lines N")
	}
	id, err := strconv.ParseUint(rest[0], 10, 64)
	if err != nil || id == 0 {
		return Request{}, fmt.Errorf("terminal id must be a positive integer: %q", rest[0])
	}
	lines := DefaultLines
	if len(rest) == 3 {
		lines, err = strconv.Atoi(rest[2])
		if err != nil || lines < 1 || lines > MaxLines {
			return Request{}, fmt.Errorf("--lines must be 1 to %d: %q", MaxLines, rest[2])
		}
	}
	return Request{Op: OpScreen, Terminal: id, Lines: lines}, nil
}

func parseOpen(args []string) (Request, error) {
	target, err := parseTarget(args[0])
	if err != nil {
		return Request{}, err
	}
	req := Request{Op: OpOpen, Target: &target}
	if target.Kind == TargetPeek {
		return Request{}, errors.New("open takes no peek: open terminal:<id> --peek peeks a terminal")
	}
	for rest := args[1:]; len(rest) > 0; {
		flag := rest[0]
		if flag == "--peek" && target.Kind == TargetTerminal && !req.Peek {
			req.Peek, rest = true, rest[1:]
			continue
		}
		if flag == "--in" && target.Kind == TargetDiff && req.In == "" && len(rest) > 1 && strings.TrimSpace(rest[1]) != "" {
			req.In, rest = strings.TrimSpace(rest[1]), rest[2:]
			continue
		}
		return Request{}, fmt.Errorf("%s takes no %q: --peek goes with terminal:, --in <perspective> with diff:, once each", args[0], flag)
	}
	return req, nil
}

func parseSend(args []string, stdin io.Reader) (Request, error) {
	if len(args) != 2 {
		return Request{}, errors.New("send takes a terminal id and a text, or - for standard input")
	}
	id, err := strconv.ParseUint(args[0], 10, 64)
	if err != nil || id == 0 {
		return Request{}, fmt.Errorf("terminal id must be a positive integer: %q", args[0])
	}
	text := args[1]
	if text == "-" {
		// A line break at the end may come beyond MaxText, as it goes; read
		// one byte more than that allows, to tell a longer text.
		const room = len("\r\n")
		b, err := io.ReadAll(io.LimitReader(stdin, int64(MaxText+room+1)))
		if err != nil {
			return Request{}, fmt.Errorf("read standard input: %w", err)
		}
		if len(b) > MaxText+room {
			return Request{}, fmt.Errorf("text must be at most %d bytes", MaxText)
		}
		text = string(b)
	}
	text = strings.TrimRight(text, "\r\n")
	if strings.TrimSpace(text) == "" || len(text) > MaxText {
		return Request{}, fmt.Errorf("text must be 1 to %d bytes, not blank", MaxText)
	}
	return Request{Op: OpSend, Terminal: id, Text: text}, nil
}

func parseClose(arg string) (Request, error) {
	target, err := parseTarget(arg)
	if err != nil {
		return Request{}, err
	}
	if target.Kind == TargetProject {
		return Request{}, errors.New("close takes no project")
	}
	return Request{Op: OpClose, Target: &target}, nil
}

// parseTarget reads kind:value, or peek.
func parseTarget(arg string) (Target, error) {
	if arg == string(TargetPeek) {
		return Target{Kind: TargetPeek}, nil
	}
	kind, value, ok := strings.Cut(arg, ":")
	target := Target{Kind: TargetKind(kind)}
	if !ok || !slices.Contains([]TargetKind{TargetTerminal, TargetDiff, TargetProject, TargetPerspective, TargetFile}, target.Kind) {
		return Target{}, fmt.Errorf("target must be terminal:<id>, diff:<id>, file:<path>, project:<name>, perspective:<name> or peek: %q", arg)
	}
	if target.Kind == TargetFile {
		if value == "" {
			return Target{}, errors.New("file needs a path")
		}
		path, err := filepath.Abs(value)
		if err != nil {
			return Target{}, err
		}
		target.Name = path
		return target, nil
	}
	if target.Kind == TargetProject || target.Kind == TargetPerspective {
		target.Name = strings.TrimSpace(value)
		if target.Name == "" {
			return Target{}, fmt.Errorf("%s needs a name", kind)
		}
		return target, nil
	}
	id, err := strconv.ParseUint(value, 10, 64)
	if err != nil || id == 0 {
		return Target{}, fmt.Errorf("terminal id must be a positive integer: %q", value)
	}
	target.ID = id
	return target, nil
}

func parseRun(args []string) (Request, error) {
	req := Request{Op: OpRun}
	if err := parseCommand(args, &req, true); err != nil {
		return Request{}, err
	}
	return req, nil
}

// parseLoop reads `loop`, `loop cancel <id>` or `loop <every> …`.
func parseLoop(args []string) (Request, error) {
	if len(args) == 0 {
		return Request{Op: OpLoops}, nil
	}
	if args[0] == "cancel" {
		if len(args) != 2 {
			return Request{}, errors.New("loop cancel takes a loop id")
		}
		id, err := strconv.ParseUint(args[1], 10, 64)
		if err != nil || id == 0 {
			return Request{}, fmt.Errorf("loop id must be a positive integer: %q", args[1])
		}
		return Request{Op: OpCancelLoop, Terminal: id}, nil
	}
	every, err := time.ParseDuration(args[0])
	if err != nil || every < MinEvery || every%time.Second != 0 {
		return Request{}, fmt.Errorf("interval must be whole seconds, at least %s, as 90s, 5m or 1h: %q", MinEvery, args[0])
	}
	req := Request{Op: OpLoop, Every: every}
	if err := parseCommand(args[1:], &req, false); err != nil {
		return Request{}, err
	}
	return req, nil
}

// parseCommand reads the options before a command, --open only where
// open, then the command: the words left, joined by spaces, as ssh joins
// them. Dir is where it runs: --dir's, made absolute, else here.
func parseCommand(args []string, req *Request, open bool) error {
	dir, dirSet := "", false
	for len(args) > 0 && strings.HasPrefix(args[0], "--") {
		flag := args[0]
		if flag == "--open" && open && !req.Open {
			req.Open, args = true, args[1:]
			continue
		}
		if flag == "--dir" && !dirSet && len(args) > 1 && args[1] != "" {
			dir, dirSet, args = args[1], true, args[2:]
			continue
		}
		return fmt.Errorf("unknown, repeated or incomplete option %q", flag)
	}
	req.Command = strings.TrimSpace(strings.Join(args, " "))
	if req.Command == "" {
		return errors.New("no command")
	}
	abs, err := filepath.Abs(cmp.Or(dir, "."))
	if err != nil {
		return fmt.Errorf("directory: %w", err)
	}
	req.Dir = abs
	return nil
}
