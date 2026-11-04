package tool

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"
)

const usage = "usage: jun tool help | terminals | screen <id> [--lines N] |\n" +
	"  open <target> [--in <perspective>] [--peek] | close <target> | back | forward |\n" +
	"  send <id> <text> | send <id> -"

const help = `jun tool: ask the Jun that runs this terminal about its terminals, and
change what it shows.

Run it as "$JUN_BIN" tool …, which works whatever PATH says.

  jun tool help
      This text.
  jun tool terminals
      Jun's running terminals, as a JSON array, by id:
        id       the terminal's id, for the other commands
        project  the project it belongs to
        dir      its shell's directory
        title    its title, as Jun shows it
        agent    claude, codex or pi, when one runs in it; absent otherwise
        status   waiting, working or idle, with agent
        self     true for the terminal the caller runs in
  jun tool screen <id> [--lines N]
      The last N lines of the terminal's text, scrollback included, as
      plain text (default 40, at most 500). A line the terminal wraps is
      one line. Agents' text includes their frames, status bars and input
      boxes.

Changing what Jun shows, for the user to see something:
  jun tool open <target>
      terminal:<id>        shows the terminal's tab, switching project and
                           perspective as needed; reopens a closed tab
      terminal:<id> --peek shows an agent's terminal in the peek drawer, or
                           its tab when that is on screen
      diff:<id>            opens the changes of the branch the terminal is
                           on, in the current perspective
      diff:<id> --in <perspective>
                           first switches to that perspective of the
                           terminal's project, made when there is none
      project:<name>       switches to the project
      perspective:<name>   switches to the perspective of the current
                           project, made when there is none
  jun tool close <target>
      terminal:<id>        closes its tab in the current perspective; the
                           shell keeps running
      diff:<id>            closes the Diff of its branch there
      peek                 closes the peek drawer
      perspective:<name>   deletes the perspective; with terminals in it,
                           Jun asks the user to confirm, and says so
  jun tool back | forward
      One step through where the user has been, as ⌘[ and ⌘] go.

Sending:
  jun tool send <id> <text>
  jun tool send <id> -
      Submits the text to the agent in the terminal, as pasted then
      Enter: line breaks stay in one message. With -, the text is
      standard input. Line breaks at its end go; at most 1 MiB. Any agent
      takes it, whatever it is doing, the caller's own too; a terminal
      running no agent does not. Prints nothing once the agent works,
      else "sent, not started yet" after 5 seconds.

  Names match whatever their case. Jun changes what it shows only for the
  terminal the user is with: the one shown in the peek drawer, or else
  the terminal last focused, whose last input came from the user rather
  than from send; and not while a dialog waits for the user.
  A change done prints nothing, or what it leaves pending.

Environment, set in each of Jun's terminals:
  JUN_API       the socket of the Jun that runs the terminal; nothing else
                is tried without it
  JUN_BIN       Jun's executable
  JUN_TERMINAL  the terminal's id, for reference: Jun tells the caller by
                its processes, not by this

Exit status: 0 done, 1 refused or Jun unreachable, 2 usage error.
Errors go to standard error.

In a sandbox: the commands connect to the Unix socket in JUN_API. Codex's
sandbox blocks it ("connect: operation not permitted"): approve running
the command outside the sandbox, or set network_access = true under
[sandbox_workspace_write] in ~/.codex/config.toml. Claude Code's sandbox
and Pi let it through.
`

var errHelp = errors.New("help")

// Main runs jun tool with args, the words after "tool", returning the
// exit status.
func Main(args []string, stdin io.Reader, stdout, stderr io.Writer, getenv func(string) string) int {
	req, err := parse(args, stdin)
	if errors.Is(err, errHelp) {
		fmt.Fprint(stdout, help)
		return 0
	}
	if err != nil {
		fmt.Fprintf(stderr, "jun tool: %v\n%s\n", err, usage)
		return 2
	}
	path := getenv(EnvAPI)
	if path == "" {
		fmt.Fprintf(stderr, "jun tool: %s is not set; run inside a Jun terminal\n", EnvAPI)
		return 1
	}
	resp, err := call(path, req)
	if err != nil {
		fmt.Fprintf(stderr, "jun tool: %v\n", err)
		return 1
	}
	if resp.Error != "" {
		fmt.Fprintf(stderr, "jun tool: %s\n", resp.Error)
		return 1
	}
	if req.Op == OpScreen {
		fmt.Fprint(stdout, resp.Screen)
		return 0
	}
	if req.Op != OpTerminals {
		if resp.Message != "" {
			fmt.Fprintln(stdout, resp.Message)
		}
		return 0
	}
	out, err := json.MarshalIndent(append([]Terminal{}, resp.Terminals...), "", "  ")
	if err != nil {
		fmt.Fprintf(stderr, "jun tool: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "%s\n", out)
	return 0
}

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
	if !ok || !slices.Contains([]TargetKind{TargetTerminal, TargetDiff, TargetProject, TargetPerspective}, target.Kind) {
		return Target{}, fmt.Errorf("target must be terminal:<id>, diff:<id>, project:<name>, perspective:<name> or peek: %q", arg)
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

func call(path string, req Request) (resp Response, err error) {
	conn, err := net.DialTimeout("unix", path, 2*time.Second)
	if err != nil {
		return Response{}, fmt.Errorf("cannot reach Jun at %s: %w", path, err)
	}
	defer func() { err = errors.Join(err, conn.Close()) }()
	if err := conn.SetDeadline(time.Now().Add(exchangeTimeout)); err != nil {
		return Response{}, err
	}
	if err := json.NewEncoder(conn).Encode(req); err != nil {
		return Response{}, answerError(err)
	}
	line, err := bufio.NewReader(conn).ReadBytes('\n')
	if err != nil {
		return Response{}, answerError(err)
	}
	if err := json.Unmarshal(line, &resp); err != nil {
		return Response{}, fmt.Errorf("bad response: %w", err)
	}
	return resp, nil
}

func answerError(err error) error {
	if errors.Is(err, os.ErrDeadlineExceeded) {
		return errors.New("Jun did not answer")
	}
	return err
}
