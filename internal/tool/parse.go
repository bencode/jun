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
	"unicode"
)

func parse(args []string, stdin io.Reader) (Request, error) {
	if len(args) == 0 {
		return Request{}, errors.New("no command")
	}
	command, rest := args[0], args[1:]
	if command == "help" && len(rest) == 0 {
		return Request{}, errHelp
	}
	if command == "terminals" {
		if len(rest) == 0 {
			return Request{Op: OpTerminals}, nil
		}
		return parseUpdateTerminal(rest)
	}
	if command == "roles" {
		return parseRoles(rest)
	}
	if command == "watch" && len(rest) == 0 {
		return Request{Op: OpWatch}, nil
	}
	if command == "leave" && len(rest) == 0 {
		return Request{Op: OpLeave}, nil
	}
	if command == "join" {
		if len(rest) != 1 {
			return Request{}, errors.New("join takes the terminal id of the agent to be led by")
		}
		id, err := strconv.ParseUint(rest[0], 10, 64)
		if err != nil || id == 0 {
			return Request{}, fmt.Errorf("terminal id must be a positive integer: %q", rest[0])
		}
		return Request{Op: OpJoin, Terminal: id}, nil
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
	if command == "message" {
		return parseMessage(rest, stdin)
	}
	if command == "goal" {
		return parseGoal(rest, stdin)
	}
	if command == "task" {
		return parseTask(rest, stdin)
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
	text, err := readText(args[1], stdin, MaxText)
	if err != nil {
		return Request{}, err
	}
	return Request{Op: OpSend, Terminal: id, Text: text}, nil
}

func parseMessage(args []string, stdin io.Reader) (Request, error) {
	if len(args) == 0 {
		return Request{Op: OpMessages}, nil
	}
	if args[0] == "update" && len(args) > 1 {
		return parseUpdateMessage(args[1:], stdin)
	}
	if args[0] == "delete" && len(args) > 1 {
		id, err := strconv.ParseUint(args[1], 10, 64)
		if err != nil || id == 0 || len(args) > 2 {
			return Request{}, errors.New("message delete takes one id, a positive integer")
		}
		return Request{Op: OpDeleteMessage, Message: id}, nil
	}
	req := Request{Op: OpMessage}
	flags := map[string]func(string) error{
		"--title": func(title string) error {
			if strings.TrimSpace(title) == "" || strings.ContainsAny(title, "\r\n") || len(title) > MaxTitle {
				return fmt.Errorf("title must be one line of 1 to %d bytes", MaxTitle)
			}
			req.Title = title
			return nil
		},
		"--kind": func(kind string) (err error) {
			req.Kind, err = parseKind(kind)
			return err
		},
		"--link": func(resource string) error {
			target, err := parseTarget(resource)
			if err != nil {
				return err
			}
			if target.Kind == TargetPeek || target.Kind == TargetPerspective {
				return errors.New("--link takes terminal:<id>, diff:<id>, file:<path> or project:<name>")
			}
			req.Target = &target
			return nil
		},
		"--task": func(id string) (err error) {
			req.Task, err = itemID("task", id)
			return err
		},
	}
	for len(args) > 1 && flags[args[0]] != nil {
		if err := flags[args[0]](args[1]); err != nil {
			return Request{}, err
		}
		delete(flags, args[0])
		args = args[2:]
	}
	if len(args) == 0 && req.Title != "" {
		return req, nil
	}
	if len(args) != 1 || isFlag(args[0]) {
		return Request{}, errors.New("message takes a text, or - for standard input, after --title <title>, --kind <kind>, --link <resource> and --task <id>, all optional")
	}
	text, err := readText(args[0], stdin, MaxMessage)
	if errors.Is(err, errBlank) && req.Title != "" {
		return req, nil
	}
	if err != nil {
		return Request{}, err
	}
	req.Text = text
	return req, nil
}

// parseUpdateMessage reads what follows `message update`: an id, then a
// kind, a task, a reply or any of them.
func parseUpdateMessage(args []string, stdin io.Reader) (Request, error) {
	const usage = "message update takes an id, then --kind <kind>, --task <id>, a text or - for standard input, at least one"
	id, err := strconv.ParseUint(args[0], 10, 64)
	if err != nil || id == 0 {
		return Request{}, fmt.Errorf("message id must be a positive integer: %q", args[0])
	}
	req, rest := Request{Op: OpUpdateMessage, Message: id}, args[1:]
	for len(rest) > 1 && (rest[0] == "--kind" && req.Kind == "" || rest[0] == "--task" && req.Task == 0) {
		if rest[0] == "--kind" {
			req.Kind, err = parseKind(rest[1])
		} else {
			req.Task, err = itemID("task", rest[1])
		}
		if err != nil {
			return Request{}, err
		}
		rest = rest[2:]
	}
	changed := req.Kind != "" || req.Task != 0
	if len(rest) == 0 && changed {
		return req, nil
	}
	if len(rest) != 1 || isFlag(rest[0]) {
		return Request{}, errors.New(usage)
	}
	text, err := readText(rest[0], stdin, MaxMessage)
	if errors.Is(err, errBlank) && changed {
		return req, nil
	}
	if err != nil {
		return Request{}, err
	}
	req.Text = text
	return req, nil
}

// parseUpdateTerminal reads what follows `terminals`: update, an id, then
// the options in any order, each once: --name, a timer's --every, --at and
// --text, or --no-timer.
func parseUpdateTerminal(args []string) (Request, error) {
	const usage = "terminals takes update <id> [--name <name>] [--every <every>] [--at <HH:MM>] [--text <text>] [--no-timer], " +
		"or nothing to list them"
	if len(args) < 3 || args[0] != "update" {
		return Request{}, errors.New(usage)
	}
	id, err := strconv.ParseUint(args[1], 10, 64)
	if err != nil || id == 0 {
		return Request{}, fmt.Errorf("terminal id must be a positive integer: %q", args[1])
	}
	req, text := Request{Op: OpUpdateTerminal, Terminal: id}, ""
	options := map[string]func(string) error{
		"--name": func(name string) error {
			name = strings.TrimSpace(name)
			req.Name = &name
			return CheckName(name)
		},
		"--every": func(every string) (err error) {
			req.Every, err = parseEvery(every)
			return err
		},
		"--at": func(clock string) (err error) {
			req.At, err = nextAt(time.Now(), clock)
			return err
		},
		"--text": func(t string) error {
			text = t
			return CheckTimerText(t)
		},
	}
	for rest := args[2:]; len(rest) > 0; {
		if rest[0] == "--no-timer" && !req.NoTimer {
			req.NoTimer, rest = true, rest[1:]
			continue
		}
		option := options[rest[0]]
		if option == nil || len(rest) < 2 {
			return Request{}, fmt.Errorf("unknown, repeated or incomplete option %q: %s", rest[0], usage)
		}
		if err := option(rest[1]); err != nil {
			return Request{}, err
		}
		delete(options, rest[0])
		rest = rest[2:]
	}
	timer := req.Every != 0 || !req.At.IsZero()
	if _, textLeft := options["--text"]; !textLeft && !timer {
		return Request{}, errors.New("--text goes with --every or --at")
	}
	if req.NoTimer && timer {
		return Request{}, errors.New("--no-timer goes with no --every or --at")
	}
	req.Text = text
	return req, nil
}

// parseRoles reads what follows `roles`: nothing, --mine, --open,
// --team and a role's name, or a role's name and --limit.
func parseRoles(args []string) (Request, error) {
	const usage = "roles takes nothing, --mine, --open, --team <name>, or a role's name and --limit <n>"
	if len(args) == 0 {
		return Request{Op: OpRoles}, nil
	}
	if len(args) == 1 && args[0] == "--mine" {
		return Request{Op: OpRoles, Mine: true, Limit: DefaultRoleSessions}, nil
	}
	if len(args) == 1 && args[0] == "--open" {
		return Request{Op: OpRoles, Away: true}, nil
	}
	if args[0] == "--team" {
		if len(args) != 2 {
			return Request{}, errors.New(usage)
		}
		name, err := roleName(args[1], usage)
		return Request{Op: OpRoles, Team: &name}, err
	}
	name, err := roleName(args[0], usage)
	if err != nil {
		return Request{}, err
	}
	req := Request{Op: OpRoles, Name: &name, Limit: DefaultRoleSessions}
	if len(args) == 1 {
		return req, nil
	}
	if len(args) != 3 || args[1] != "--limit" {
		return Request{}, errors.New(usage)
	}
	limit, err := strconv.Atoi(args[2])
	if err != nil || limit < 1 {
		return Request{}, fmt.Errorf("limit must be a positive integer: %q", args[2])
	}
	req.Limit = limit
	return req, nil
}

// roleName is the argument as a role's name, one a terminal can take.
func roleName(arg, usage string) (string, error) {
	name := strings.TrimSpace(arg)
	if err := CheckName(name); err != nil || name == "" {
		return "", errors.New(usage)
	}
	return name, nil
}

func parseEvery(every string) (time.Duration, error) {
	d, err := time.ParseDuration(every)
	if err == nil {
		err = CheckEvery(d)
	}
	if err != nil {
		return 0, fmt.Errorf("interval must be whole seconds, at least %s, as 90s, 5m or 1h: %q", MinEvery, every)
	}
	return d, nil
}

// CheckEvery tells whether a loop or a timer can go so often.
func CheckEvery(every time.Duration) error {
	if every < MinEvery || every%time.Second != 0 {
		return fmt.Errorf("interval must be whole seconds, at least %s", MinEvery)
	}
	return nil
}

// CheckTimerText tells whether a timer can signal with the text: one line
// of at most MaxTimerText bytes.
func CheckTimerText(text string) error {
	if strings.ContainsFunc(text, unicode.IsControl) || len(text) > MaxTimerText {
		return fmt.Errorf("timer text must be one line of at most %d bytes", MaxTimerText)
	}
	return nil
}

// nextAt is the first time after now that the clock, HH:MM, reads so:
// today, or tomorrow once past.
func nextAt(now time.Time, clock string) (time.Time, error) {
	t, err := time.Parse("15:04", clock)
	if err != nil {
		return time.Time{}, fmt.Errorf("--at takes a time of day as HH:MM: %q", clock)
	}
	at := time.Date(now.Year(), now.Month(), now.Day(), t.Hour(), t.Minute(), 0, 0, now.Location())
	if !at.After(now) {
		at = at.AddDate(0, 0, 1)
	}
	return at, nil
}

// CheckName tells whether a terminal can be named so, trimmed: a line of
// at most MaxName bytes that is not a number, as ids are; none drops a
// name.
func CheckName(name string) error {
	if name != strings.TrimSpace(name) || strings.ContainsFunc(name, unicode.IsControl) || len(name) > MaxName {
		return fmt.Errorf("name must be one line of at most %d bytes, trimmed: %q", MaxName, name)
	}
	if _, err := strconv.ParseUint(name, 10, 64); err == nil {
		return fmt.Errorf("name must not be a number, which ids are: %q", name)
	}
	return nil
}

// isFlag tells a mistyped option from a text, which has spaces.
func isFlag(arg string) bool {
	return strings.HasPrefix(arg, "--") && !strings.Contains(arg, " ") || arg == "-h"
}

func parseKind(kind string) (MessageKind, error) {
	if !slices.Contains(MessageKinds, MessageKind(kind)) {
		return "", fmt.Errorf("kind must be one of %v, not %q", MessageKinds, kind)
	}
	return MessageKind(kind), nil
}

var errBlank = errors.New("text must not be blank")

// readText is the text argument, or standard input for -, without the line
// breaks at its end.
func readText(arg string, stdin io.Reader, limit int) (string, error) {
	text := arg
	if text == "-" {
		// A line break at the end may come beyond the limit, as it goes; read
		// one byte more than that allows, to tell a longer text.
		const room = len("\r\n")
		b, err := io.ReadAll(io.LimitReader(stdin, int64(limit+room+1)))
		if err != nil {
			return "", fmt.Errorf("read standard input: %w", err)
		}
		if len(b) > limit+room {
			return "", fmt.Errorf("text must be at most %d bytes", limit)
		}
		text = string(b)
	}
	text = strings.TrimRight(text, "\r\n")
	if strings.TrimSpace(text) == "" {
		return "", errBlank
	}
	if len(text) > limit {
		return "", fmt.Errorf("text must be at most %d bytes", limit)
	}
	return text, nil
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
	every, err := parseEvery(args[0])
	if err != nil {
		return Request{}, err
	}
	req := Request{Op: OpLoop, Every: every}
	if err := parseCommand(args[1:], &req, false); err != nil {
		return Request{}, err
	}
	return req, nil
}

// parseCommand reads the options before a command, --open and --project
// only for run, then the command: the words left, joined by spaces, as ssh
// joins them. Dir is where it runs: --dir's, made absolute, else here, or
// with --project none, for Jun to take the project's.
func parseCommand(args []string, req *Request, run bool) error {
	dir, dirSet := "", false
	for len(args) > 0 && strings.HasPrefix(args[0], "--") {
		flag := args[0]
		if flag == "--open" && run && !req.Open {
			req.Open, args = true, args[1:]
			continue
		}
		if flag == "--project" && run && req.Project == "" && len(args) > 1 && strings.TrimSpace(args[1]) != "" {
			req.Project, args = strings.TrimSpace(args[1]), args[2:]
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
	if req.Project != "" && !dirSet {
		return nil
	}
	abs, err := filepath.Abs(cmp.Or(dir, "."))
	if err != nil {
		return fmt.Errorf("directory: %w", err)
	}
	req.Dir = abs
	return nil
}
