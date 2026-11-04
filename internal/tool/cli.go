package tool

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"time"
)

const usage = "usage: jun tool help | terminals | screen <id> [--lines N] |\n" +
	"  open <target> [--in <perspective>] [--peek] | close <target> | back | forward |\n" +
	"  send <id> <text> | send <id> - |\n" +
	"  run [--dir <path>] [--open] <command…> | loop <every> [--dir <path>] <command…> | loop | loop cancel <id>"

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

  Names match whatever their case. Jun changes what it shows only for the
  terminal the user is with: the one shown in the peek drawer, or else
  the terminal last focused, whose last input came from the user rather
  than from send; and not while a dialog waits for the user.
  A change done prints nothing, or what it leaves pending.

Sending:
  jun tool send <id> <text>
  jun tool send <id> -
      Submits the text to the agent in the terminal, as pasted then
      Enter: line breaks stay in one message. With -, the text is
      standard input. Line breaks at its end go; at most 1 MiB. Any agent
      takes it, whatever it is doing, the caller's own too; a terminal
      running no agent does not. Prints nothing once the agent works,
      else "sent, not started yet" after 5 seconds.

Running commands, each in a terminal of its own, in the caller's
project, with no tab unless asked for:
  jun tool run [--dir <path>] [--open] <command…>
      Starts a terminal in the directory (default: the current one) and
      types the command into its shell, which stays once the command
      ends. Prints the terminal's id. --open shows it in a tab, focused,
      as open does, for the terminal the user is with only.
  jun tool loop <every> [--dir <path>] <command…>
      Runs the command again and again in a terminal of its own, every
      interval (90s, 5m, 1h: at least 1m) after a run ends, with
      $SHELL -l -c, as a script runs: write absolute paths or $JUN_BIN.
      Each run shows its time and exit status there. Prints the loop's
      id, its terminal's.
  jun tool loop
      The loops, as a JSON array: id, every, command, project, dir.
  jun tool loop cancel <id>
      Ends the loop and its terminal; Control-C in it does too.
  The words of a command are joined by spaces into one command line, as
  ssh joins them: quote a command whose own quotes must stay.

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
	if err := output(stdout, req.Op, resp); err != nil {
		fmt.Fprintf(stderr, "jun tool: %v\n", err)
		return 1
	}
	return 0
}

// output writes what an op answers: text, an id, JSON, or what a change
// leaves pending.
func output(w io.Writer, op Op, resp Response) error {
	var err error
	if op == OpScreen {
		_, err = fmt.Fprint(w, resp.Screen)
		return err
	}
	if op == OpRun || op == OpLoop {
		_, err = fmt.Fprintln(w, resp.Terminal)
		return err
	}
	var list any
	if op == OpTerminals {
		list = append([]Terminal{}, resp.Terminals...)
	} else if op == OpLoops {
		list = append([]Loop{}, resp.Loops...)
	} else {
		if resp.Message != "" {
			_, err = fmt.Fprintln(w, resp.Message)
		}
		return err
	}
	out, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "%s\n", out)
	return err
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
