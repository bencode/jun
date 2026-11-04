# Agent Tools

Programs in Jun's terminals, agents above all, can ask the Jun that runs them about its terminals, and change what it shows, with `jun tool`. An agent you work with, in a project of its own, can then see what the other agents are doing, read their screens, bring you a terminal or a branch's changes to look at, and pass a message on to another agent. Jun gives the tools only; how an agent uses them, as a coordinator or otherwise, is up to the instructions or skill you give it.

`"$JUN_BIN" tool help` prints what this page says, for an agent to read.

## Commands

| Command | Output |
|---|---|
| `jun tool help` | The commands, fields, environment and exit status |
| `jun tool terminals` | Jun's running terminals, as a JSON array, by id |
| `jun tool screen <id> [--lines N]` | The last N lines of the terminal's text, as plain text |
| `jun tool open <target> [--in <perspective>] [--peek]` | Shows the target. Prints nothing. |
| `jun tool close <target>` | Closes the target. Prints nothing, or what it leaves pending. |
| `jun tool back` / `jun tool forward` | One step back or forward through where you have been, as ⌘[ and ⌘] go. Prints nothing. |
| `jun tool send <id> <text>` / `jun tool send <id> -` | Submits the text to the agent in the terminal. Prints nothing, or `sent, not started yet`. |

Run them as `"$JUN_BIN" tool …`, which works whatever `PATH` and aliases say.

### terminals

```json
[
  {
    "id": 3,
    "project": "web",
    "dir": "/path/to/web",
    "title": "Fix the login form",
    "agent": "claude",
    "status": "waiting",
    "self": false
  }
]
```

| Field | Meaning |
|---|---|
| `id` | The terminal's id, for `screen`. Ids hold while Jun runs; a new run numbers its terminals anew. |
| `project` | The project the terminal belongs to |
| `dir` | Its shell's directory |
| `title` | Its title, as the Agents view or its tab shows it |
| `agent` | `claude`, `codex` or `pi`, when one runs in it; absent otherwise |
| `status` | `waiting`, `working` or `idle`, as the [Agents](agents-view.md) view shows it; with `agent` only |
| `self` | `true` for the terminal the caller runs in |

Terminals with closed tabs are listed too: their shells keep running. Terminals whose shell has ended are not.

### screen

- The terminal's text with its scrollback, after the blank lines below the cursor: 40 lines by default, 1 to 500 with `--lines`.
- A line the terminal wraps on screen counts as one.
- An agent's text includes what it draws around its messages: frames, status bars and its input box.

### open and close

| Target | `open` | `close` |
|---|---|---|
| `terminal:<id>` | Shows the terminal's tab, switching project and perspective as needed, and reopens its tab when it was closed. With `--peek`, an agent's terminal shows in the [peek drawer](agents-view.md#peek) instead, or its tab when that is on screen. | Closes its tab in the current perspective. The shell keeps running. |
| `diff:<id>` | Opens the changes of the branch the terminal is on, as its agent card's Review Changes does, in the current perspective. With `--in <perspective>`, first switches to that perspective of the terminal's project, made when there is none. | Closes the Diff of that branch in the current perspective. |
| `project:<name>` | Switches to the project | — |
| `perspective:<name>` | Switches to the perspective of the current project, made when there is none | Deletes it, as Delete Perspective… does: with terminals in it, Jun asks you to confirm, and prints `waiting for the user to confirm`. A project keeps at least one perspective. |
| `peek` | — | Closes the peek drawer |

- Names match whatever their case.
- The first time a project's repository is asked for, Jun reads its branches and worktrees; `diff:` waits a few seconds for them. When the terminal is in none of the worktrees read, as when its agent made one since, Jun reads them again once.

A review, for instance, takes two calls: `open diff:7 --in Review`, and once you are done, `back`.

### When Jun changes what it shows

`open`, `close`, `back` and `forward` change what you see, so Jun runs them only:

- **For the terminal you are with**: the one shown in the peek drawer, or else the terminal you last focused. Otherwise: `the user is not with you`.
- **When you spoke to it last**: an agent whose last input came from `send` rather than your keys or a paste changes nothing until you type in its terminal. Otherwise: `your last input came from another agent, not the user`.
- **While no dialog waits for you**: the command palette, a form, a confirmation, an error or the shortcuts panel. Otherwise: `Jun is waiting for the user`.
- **On another terminal**: a terminal cannot open or close itself (`that is your own terminal`).

Opening another terminal gives it the focus, so you are then with that terminal: the caller's next change is refused until you come back to it.

### send

- The text goes in as a paste, then Enter, as when you paste a message and submit it: its line breaks stay in one message. Checked with Claude Code 2.1.318, Codex 0.166.0 and Pi 1.5.0.
- With `-`, the text is standard input, for long or multi-line messages. Line breaks at its end go; at most 1 MiB.
- Any agent takes it, whatever it is doing, the caller's own terminal too. A terminal running no agent does not: text there would run as a command.
- Once sent, Jun waits up to 5 seconds for the agent to work. If it does not by then, `send` prints `sent, not started yet`, still with status 0.

## Environment

Jun sets these in each of its terminals:

| Variable | Value |
|---|---|
| `JUN_API` | The socket of the Jun that runs the terminal. The commands try no other: without it they fail. |
| `JUN_BIN` | Jun's executable |
| `JUN_TERMINAL` | The terminal's id, for reference |

Jun tells which terminal a caller runs in by its processes, as macOS reports them, not by `JUN_TERMINAL`.

- **The socket**: `jun.sock`, beside the [configuration file](configuration.md).
- **Who can connect**: only your user.
- **Another Jun**: while a Jun serves the socket, another one with the same configuration file serves no tools, and says so. A socket left by a Jun that ended is replaced.
- **Outside Jun's terminals**: the read-only commands work from anywhere with `JUN_API` set, for instance to debug, and then `self` is `false` everywhere. Changes are refused (`caller is not a terminal of this Jun`).
- **A Jun in another's terminal**: its terminals reach it, not the outer Jun. A program there that points `JUN_API` at the outer Jun by hand counts, for the outer Jun, as running in the terminal the inner Jun was started from, as its processes descend from that terminal's shell.

## Exit status

| Status | Meaning |
|---|---|
| 0 | Done |
| 1 | Refused, or Jun did not answer or cannot be reached; the reason goes to standard error |
| 2 | Usage error |

## In a sandbox

The commands connect to a Unix socket, which some agents' sandboxes block:

| Agent | Socket in its sandbox |
|---|---|
| Claude Code | Reached, with the sandbox's defaults |
| Codex | Blocked in `read-only` and `workspace-write` (`connect: operation not permitted`). Approve running the command outside the sandbox, or allow commands network access in `~/.codex/config.toml`: `network_access = true` under `[sandbox_workspace_write]`. |
| Pi | Reached: Pi runs commands without a sandbox |

Checked with Claude Code 2.1.318, Codex 0.166.0 and Pi 1.5.0.

## A directory named tool

`jun tool` runs the tools. To open a directory named `tool` as a project, write `jun ./tool`.
