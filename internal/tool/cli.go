package tool

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
)

const usage = "usage: jun tool help | terminals | watch |\n" +
	"  terminals update <id> [--name <name>] [--every <every>] [--at <HH:MM>] [--text <text>] [--no-timer] |\n" +
	"  screen <id> [--lines N] | open <target> [--in <perspective>] [--peek] | close <target> | back | forward |\n" +
	"  send <id> <text> | send <id> - | message [--title <title>] [--kind <kind>] [--link <resource>] [<text|->] |\n" +
	"  message | message update <id> [--kind <kind>] [--task <id>] [<text|->] |\n" +
	"  goal [--all | --state <state>…] [--match <text>] | goal <id> | goal add … <title> | goal update <id> … | goal delete <id> |\n" +
	"  task [--all | --state <state>…] [--owner <role>] [--project <name>] [--goal <id>] [--match <text>] |\n" +
	"  task <id> | task add --goal <id> … <title> | task update <id> … | task delete <id> |\n" +
	"  roles | roles --mine | roles --open | roles --team <name> | roles <name> [--limit <n>] |\n" +
	"  run [--project <name>] [--dir <path>] [--open] <command…> | join <id> | leave |\n" +
	"  loop <every> [--dir <path>] <command…> | loop | loop cancel <id>"

const help = `jun tool: ask the Jun that runs this terminal about its terminals, and
change what it shows.

Run it as "$JUN_BIN" tool …, which works whatever PATH says.

  jun tool help
      This text.
  jun tool terminals
      Jun's running terminals, as a JSON array, by id:
        id       the terminal's id, for the other commands
        name     what agents call it, when it has one; absent otherwise
        project  the project it belongs to
        dir      its shell's directory
        title    its title, as Jun shows it
        agent    claude, codex or pi, when one runs in it; absent otherwise
        status   waiting, working or idle, with agent
        self     true for the terminal the caller runs in
        lead     the terminal of the agent that leads it, when one does:
                 the agent that ran it, or the one it joined
        timer    its timer, when it has one: every (absent for one that
                 fires once), next, the time it fires next, and text
      A name the user gives an agent is its name here: find its id by it.
  jun tool terminals update <id> [--name <name>] [--every <every>]
                                 [--at <HH:MM>] [--text <text>] [--no-timer]
      Changes the terminal, each option once, in any order; any terminal
      may change any running one. Prints nothing.
      --name    names it, for the user and agents to call its agent by, in
                place of any name before; '' drops it. A name is one line
                of at most 64 bytes, not a number, and no other running
                terminal's, whatever its case. Jun shows it in Agents,
                beside the terminal's id. The other commands still take
                the id. A name is a role: an agent naming its own
                terminal gives the role its session, from the agent's
                variables, out of any role it had, and '' takes it out.
                A role is kept while it has a session, after the terminal
                ends and Jun quits; naming a terminal a role's name, when
                no running terminal has it, takes that role on. A
                terminal whose agent ends gives up a name that is a role.
      --every   sets its timer, in place of any before, for the agent in it
                to be signaled every interval, as loop's: first after one
                interval, or at --at.
      --at      sets its timer to fire at the next HH:MM, local time, once,
                unless --every goes with it.
      --text    what the signal says after its tag, one line of at most
                500 bytes: [jun-timer 30m] <text>, or [jun-timer 15:00]
                <text> for one that fires once.
      --no-timer drops the timer.
      A timer needs an agent in the terminal. Jun signals it once it is
      due and the agent idle, with nothing in its input: a signal missed
      while it works comes once, when it is done. Jun forgets names and
      timers when the terminal ends or Jun quits; roles stay.
  jun tool roles
      The roles, as a JSON array, by name: name, lead (the role it
      reports to, if any), open (an agent took it and has not ended;
      one still open after Jun quit was at work then), terminal (the
      running one of that name, if any), count (its sessions) and last
      (its latest session: agent, id, home (the agent's directory, when
      not its default), dir, at, when it took the role, and resume, the
      shell command that resumes it).
  jun tool roles <name> [--limit <n>]
      That role, with its latest sessions in sessions, the latest first:
      20 unless --limit says.
  jun tool roles --team <name>
      That role, then the roles that report to it.
  jun tool roles --open
      The open roles no running terminal has: the ones to bring back,
      each lead before the roles that report to it.
  jun tool roles --mine
      The role of the agent session the command runs in, as roles <name>
      gives it, from the agent's variables: an empty array when it has
      none. A resumed session finds its name here.
  jun tool watch
      Agents' status as it changes, a JSON object a line, until Jun quits
      or the reader falls behind; then status 1. It starts with each
      agent, from empty:
        id, name, project, agent,   as terminals has them
        title, lead
        from, to                    the status before and now; from is
                                    empty for a new agent, to for one gone
      Jun tells idle after it holds for a few seconds, the rest at once.
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
      file:<path>          opens the file in a tab of its own, read-only,
                           wherever it is; opened again, reads it again
      project:<name>       switches to the project
      perspective:<name>   switches to the perspective of the current
                           project, made when there is none
  jun tool close <target>
      terminal:<id>        closes its tab in the current perspective; the
                           shell keeps running
      diff:<id>            closes the Diff of its branch there
      file:<path>          closes its tab there
      peek                 closes the peek drawer
      perspective:<name>   deletes the perspective; with terminals in it,
                           Jun asks the user to confirm, and says so
  jun tool back | forward
      One step through where the user has been, as ⌘[ and ⌘] go.

  Names match whatever their case. Jun changes what it shows only for the
  terminal the user is with: the one shown in the peek drawer, or else
  the terminal last focused; and not while a dialog waits for the user.
  A change done prints nothing, or what it leaves pending.

Sending:
  jun tool send <id> <text>
  jun tool send <id> -
      Submits the text to the agent in the terminal, as pasted then
      Enter: line breaks stay in one message. With -, the text is
      standard input. Line breaks at its end go; at most 1 MiB. Any agent
      takes it, whatever it is doing, the caller's own too; a terminal
      running no agent does not. Prints nothing once the agent works,
      else "sent, not started yet" after 5 seconds. While the user has
      typed in the agent's input and not submitted it, Jun holds the
      text, after any held before, and submits it once the input is
      empty; it prints "held: ..." at once.
  jun tool message [--title <title>] [--kind <kind>] [--link <resource>] [--task <id>] <text>
  jun tool message [--title <title>] [--kind <kind>] [--link <resource>] [--task <id>] -
      Shows the text to the user, in Messages in Global, without
      disturbing them: any terminal may, at any time. A title of one
      line, at most 200 bytes, goes above it; with a title the text may
      go. The kind, ask, warn, done or info, the default, gives it its
      icon. The resource, terminal:<id>, diff:<id>, file:<path> or
      project:<name>, is where it takes the user, and Jun tells the
      project from it; report on another agent with terminal:<its id>.
      Without one it is plain news, leading nowhere. A resource that
      does not exist is status 1 and shows nothing. --task names the task
      it is about (see task). With -, the text is standard input.
      Line breaks at its end go; at most 4 KiB. Jun keeps the last 500,
      until it quits. Prints the message's id, which shows as M<id>.
  jun tool message update <id> [--kind <kind>] [--task <id>] [<text>]
  jun tool message update <id> [--kind <kind>] [--task <id>] -
      Changes the message's kind or task, adds the text as a reply below
      it, or any of them: ask becoming done tells the user it is handled. A reply moves
      the message to the top; a message keeps its last 50. Any terminal
      may, whoever sent it. No such message is status 1. Prints nothing.
  jun tool message
      The messages, the latest first, as a JSON array: id, time, kind,
      title, text, project, link, as --link takes it, task, and replies,
      each with time and text.

Goals and tasks, what the user means to get done, kept across Jun's runs.
A goal serves the goal above it, if any; a task is a piece of one goal,
followed by one role. Jun shows a goal as G<id> and a task as T<id>; the
commands take the id. An id is never given again. Goals of one parent
come in the order of their importance, tasks of one goal in the order to
do them. Any terminal may change any of them.
  jun tool goal [--all | --state <state>…] [--match <text>]
      The goals, as a JSON array, each followed by those under it: open
      ones, unless --all or --state says (open, done, dropped; again for
      more). --match keeps those with the text in their title, outcome or
      text, whatever its case. Each has id, parent (absent for none),
      title, outcome (when it is done), text, link, due, period, state,
      createdAt and endedAt (when it became done or dropped).
  jun tool goal <id>
      {"goals": the goal and those above it, the top one first, "tasks":
      the goal's own tasks, whatever their state}.
  jun tool goal add [--parent <id>] --outcome <outcome> [--due <date>]
                    [--period <period>] [--link <resource>] [--text <text>|-] <title>
      Adds an open goal, the last of its siblings, and prints its id.
      The title and outcome are one line each, at most 200 and 500 bytes;
      the text at most 4 KiB: more goes in a file the link names. The
      link is file:<path> or project:<name>, which last. --due is a date
      as 2026-11-28; --period the user's plan for it: a month as 2026-11,
      an ISO week as 2026-W48, or a day as 2026-11-23.
  jun tool goal update <id> [--title <title>] [--outcome <outcome>] [--parent <id>]
                    [--due <date>] [--period <period>] [--link <resource>]
                    [--text <text>|-] [--rank <n>] [--state open|done|dropped]
      Changes the goal: '' clears due, period, link and text; --parent 0
      makes it a top one. --rank places it among its siblings, from 1;
      moved to another parent, it goes last there unless --rank says.
      done or dropped is for whoever judges the outcome, not the tasks.
  jun tool goal delete <id>
      Takes out a goal with no goals or tasks under it; drop any other.
  jun tool task [--all | --state <state>…] [--owner <role>] [--project <name>]
                [--goal <id>] [--match <text>]
      The tasks, as a JSON array, by their goals in order: the ones to do
      and doing, unless --all or --state says (todo, doing, done,
      dropped). --goal keeps that goal's and those of the goals under it;
      the others as goal's; all of them together. Each has id, goal,
      title, outcome, link, project, owner, state, createdAt, startedAt
      (when it first became doing) and endedAt; task <id> adds text and
      sessions.
  jun tool task <id>
      {"goals": the task's goal and those above it, the top one first,
      "tasks": the task}, with sessions: the agents' sessions that touched
      it, the first first, each with its role then, agent, id, home, dir,
      at and resume, the shell command that resumes it.
  jun tool task add --goal <id> [--owner <role>] [--project <name>] [--outcome <outcome>]
                    [--link <resource>] [--text <text>|-] <title>
      Adds a task to do, the last of its goal's, and prints its id. The
      owner is a role's name; the project is the caller's unless given.
  jun tool task update <id> [--state todo|doing|done|dropped] [--owner <role>]
                    [--goal <id>] [--project <name>] [--title <title>]
                    [--outcome <outcome>] [--link <resource>] [--text <text>|-] [--rank <n>]
      Changes the task, as goal update does; '' clears owner, outcome,
      link and text. To take one: --owner <your name> --state doing. Jun
      keeps its times: doing first starts it, done or dropped ends it,
      todo starts it over.
  jun tool task delete <id>
      Takes out a task still to do; drop any other.
  An agent that adds or changes a task is recorded in its sessions,
  from the agent's variables, once a session.

Running commands, each in a terminal of its own, in the caller's
project, with no tab unless asked for:
  jun tool run [--project <name>] [--dir <path>] [--open] <command…>
      Starts a terminal in the directory (default: the current one) and
      types the command into its shell, which stays once the command
      ends. Prints the terminal's id. --project starts it in that project,
      in its directory unless --dir says otherwise. --open shows it in a
      tab, focused, as open does, for the terminal the user is with only.
      Run by an agent, the terminal has that agent as its lead.
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

Leading, for an agent that has others work for it:
  jun tool join <id>
      Makes the agent in that terminal the caller's lead, in place of any
      other: terminals and watch tell it, and Jun shows it beside the
      caller in Agents. A terminal running no agent, the caller's own and
      one the caller leads, however far down, are refused.
  jun tool leave
      Drops the caller's lead. A terminal that ends drops its own, and
      those of the terminals it leads. Leads last until Jun quits; a
      role's, set by join and dropped by leave, stays (see roles).

Environment, set in each of Jun's terminals:
  JUN_API       the socket of the Jun that runs the terminal; nothing else
                is tried without it
  JUN_BIN       Jun's executable
  JUN_TERMINAL  the terminal's id. Jun tells the caller by its processes,
                and takes this only for one outside its terminals

Exit status: 0 done, 1 refused or Jun unreachable, 2 usage error.
Errors go to standard error.

Outside Jun's terminals, as in the commands Codex's shared server runs,
the variables are missing. Have the user run this in the Jun terminal of
the session, and set what it prints:
  ` + EnvCommand + `

In a sandbox: the commands connect to the Unix socket in JUN_API. Codex's
sandbox blocks it ("connect: operation not permitted"): approve running
the command outside the sandbox, or set network_access = true under
[sandbox_workspace_write] in ~/.codex/config.toml. Claude Code's sandbox
and Pi let it through.
`

var errHelp = errors.New("help")

// commands are the words that may follow "tool", those parse takes.
var commands = []string{"help", "terminals", "watch", "screen", "open", "close", "back", "forward", "send", "message", "goal", "task", "run", "join", "leave", "loop"}

// MistakenCommand tells, for the words jun was started with, how to run
// the tool command they look like when "tool" was left out, as "jun open
// file:x.md", or "": the app would take them as directories to open as
// projects, and the command would exit 0 though it ran nothing. A word
// that is a path in wd, as a directory named run, is not mistaken.
func MistakenCommand(args []string, wd string) string {
	if len(args) == 0 || !slices.Contains(commands, args[0]) {
		return ""
	}
	if _, err := os.Stat(filepath.Join(wd, args[0])); err == nil {
		return ""
	}
	words := make([]string, len(args))
	for i, w := range args {
		words[i] = shellQuote(w)
	}
	return fmt.Sprintf("jun: %q is a tool command; run it as:\n  \"$JUN_BIN\" tool %s\n", args[0], strings.Join(words, " "))
}

// shellQuote writes w for a shell to read back as one word.
func shellQuote(w string) string {
	if w != "" && strings.Trim(w, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_./:=@,+%") == "" {
		return w
	}
	return "'" + strings.ReplaceAll(w, "'", `'\''`) + "'"
}

// Main runs jun tool with args, the words after "tool", returning the
// exit status.
func Main(args []string, stdin io.Reader, stdout, stderr io.Writer, getenv func(string) string) int {
	req, err := parse(args, stdin)
	if errors.Is(err, errHelp) {
		io.WriteString(stdout, help)
		return 0
	}
	if err != nil {
		fmt.Fprintf(stderr, "jun tool: %v\n%s\n", err, usage)
		return 2
	}
	path := getenv(EnvAPI)
	if path == "" {
		fmt.Fprintf(stderr, "jun tool: %s is not set: this command runs outside Jun's terminals. Ask the user to run this "+
			"in the Jun terminal your session runs in and paste what it prints, then set those variables for each command:\n  %s\n",
			EnvAPI, EnvCommand)
		return 1
	}
	if req.Op == OpWatch {
		fmt.Fprintf(stderr, "jun tool: %v\n", watch(path, stdout))
		return 1
	}
	if req.Op == OpUpdateTerminal && req.Name != nil || req.Mine || slices.Contains(GoalOps, req.Op) {
		req.Sessions = SessionsFrom(getenv)
	}
	if named := getenv(EnvTerminal); named != "" {
		if req.Caller, err = strconv.ParseUint(named, 10, 64); err != nil {
			fmt.Fprintf(stderr, "jun tool: %s=%q is not a terminal id: set it to what echo $%s prints in the terminal your session runs in\n",
				EnvTerminal, named, EnvTerminal)
			return 2
		}
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
	if op == OpMessage || op == OpAddGoal || op == OpAddTask {
		_, err = fmt.Fprintln(w, resp.Created)
		return err
	}
	var list any
	if op == OpTerminals {
		list = append([]Terminal{}, resp.Terminals...)
	} else if op == OpLoops {
		list = append([]Loop{}, resp.Loops...)
	} else if op == OpMessages {
		list = append([]Message{}, resp.Messages...)
	} else if op == OpRoles {
		list = append([]Role{}, resp.Roles...)
	} else if op == OpGoals {
		list = append([]Goal{}, resp.Goals...)
	} else if op == OpTasks {
		list = append([]Task{}, resp.Tasks...)
	} else if op == OpGoal || op == OpTask {
		list = struct {
			Goals []Goal `json:"goals"`
			Tasks []Task `json:"tasks"`
		}{append([]Goal{}, resp.Goals...), append([]Task{}, resp.Tasks...)}
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
