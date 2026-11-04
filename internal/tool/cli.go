package tool

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"time"
)

const usage = "usage: jun tool help | terminals | screen <id> [--lines N]"

const help = `jun tool: ask the Jun that runs this terminal about its terminals.

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
func Main(args []string, stdout, stderr io.Writer, getenv func(string) string) int {
	req, err := parse(args)
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
	out, err := json.MarshalIndent(append([]Terminal{}, resp.Terminals...), "", "  ")
	if err != nil {
		fmt.Fprintf(stderr, "jun tool: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "%s\n", out)
	return 0
}

func parse(args []string) (Request, error) {
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
