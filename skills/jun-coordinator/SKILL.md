---
name: jun-coordinator
description: Coordinate the other agents and terminals of Jun, the terminal workspace, with "$JUN_BIN" tool. List Jun's terminals and the status of the Claude Code, Codex and Pi agents in them; read a terminal's screen; send a message to another agent; show the user a terminal or a branch's changes; run a command in a new terminal; run one on a schedule. Use when JUN_API is set and a task involves another agent or terminal in Jun, such as checking on agents, handing work to one, passing a message on, or showing the user something.
---

# Jun coordinator

Jun runs agents in terminals. `jun tool` lets a program in one of them see the others and act on them. Jun gives the tools only: how you coordinate is up to your task and the user's instructions.

## Before you start

- Jun sets `JUN_API` in each of its terminals. Without it, you are not in Jun: these commands cannot work.
- Always run `"$JUN_BIN" tool …`, which works whatever `PATH` says.
- `"$JUN_BIN" tool help` is the full reference. Read it when an option or output is unclear, rather than guessing.
- In Codex's sandbox, the socket is blocked (`connect: operation not permitted`): ask to run the command outside the sandbox. Claude Code and Pi reach it.
- Exit status: 0 done, 1 refused or Jun unreachable (the reason is on standard error), 2 usage error.

## Commands

| Command | Does |
|---|---|
| `terminals` | Jun's terminals as JSON: `id`, `project`, `dir`, `title`, `agent` (`claude`, `codex`, `pi`), `status`, `self` |
| `screen <id> [--lines N]` | The last N lines of a terminal's text, scrollback included (default 40, at most 500) |
| `send <id> <text>` / `send <id> -` | Submits the text to the agent in that terminal, as pasted then Enter; `-` reads standard input |
| `open terminal:<id> [--peek]` | Shows that terminal to the user: its tab, or the peek drawer |
| `open diff:<id> [--in <perspective>]` | Opens the changes of the branch that terminal is on |
| `open project:<name>` / `open perspective:<name>` | Switches the user's project or perspective |
| `close terminal:<id>` / `close diff:<id>` / `close peek` | Closes what `open` opened; a terminal's shell keeps running |
| `close perspective:<name>` | Deletes the perspective; with terminals in it, Jun asks the user to confirm |
| `back` / `forward` | Steps through where the user has been |
| `run [--dir <path>] [--open] <command…>` | Starts a terminal running the command; prints its id |
| `loop <every> [--dir <path>] <command…>` | Runs the command every interval (at least `1m`) in a terminal of its own; prints the loop's id |
| `loop` / `loop cancel <id>` | Lists the loops as JSON / ends one |

Ids are terminal ids, the same for every command. They last while Jun runs.

## Status

`status` comes with `agent`:

- `waiting`: the agent asks the user something, such as a permission or a choice.
- `working`: it is busy.
- `idle`: it waits for a new prompt. Idle does not tell done from stuck or from a question asked in its reply: read its `screen`.

## Common work

**Find an agent.** Run `terminals` and pick by `project`, `dir`, `title` and `agent`. `self: true` is your own terminal.

**See how it is going.** `screen <id> --lines 80`. The text includes the agent's frames, status bar and input box; its last reply is near the bottom.

**Hand it work or a message.**

```sh
"$JUN_BIN" tool send 7 'Run the tests and fix what fails, then summarize the changes.'
printf '%s\n' "$long_message" | "$JUN_BIN" tool send 7 -
```

- `send` prints nothing once the agent works, or `sent, not started yet` if it has not in 5 seconds: check its `screen`.
- Line breaks stay in one message. Use `-` for long or multi-line text.
- Afterwards, check `terminals` for its status, then `screen` for its answer. Trust what the screen shows over what the agent says it did.

**Show the user something.**

```sh
"$JUN_BIN" tool open diff:7            # the changes on terminal 7's branch
"$JUN_BIN" tool open terminal:7 --peek # an agent waiting for the user's answer
```

**Start another agent or a job.**

```sh
"$JUN_BIN" tool run --dir ../api codex
"$JUN_BIN" tool run 'make test 2>&1 | tail -20'
```

The new terminal belongs to your project and has no tab: follow it with `screen <id>`, or pass `--open` to show it to the user.

**Check in on a schedule.**

```sh
"$JUN_BIN" tool loop 10m "$JUN_BIN" tool send 7 "'Report your progress and anything blocking you.'"
"$JUN_BIN" tool loop                # the loops
"$JUN_BIN" tool loop cancel 12      # end one
```

## Pitfalls

- **Quoting.** The words of a command are joined by spaces into one command line, as `ssh` joins them. To keep quotes inside it, quote the whole command, or quote twice as in the loop above.
- **Loops run as scripts.** A loop runs its command with `$SHELL -l -c`: login files apply, `.zshrc` does not. Write absolute paths, or `"$JUN_BIN"`. A run waits for the one before to end.
- **Loops cost.** Each run that sends wakes an agent and spends its tokens. Choose the interval for the task, and cancel loops you no longer need: they last until cancelled or Jun quits.
- **Only agents take `send`.** A terminal running no agent refuses it, since the text would run as a command.
- **Drafts.** `send` goes into the agent's input box as is. Text already there, such as the user's draft or a prompt Claude Code put back after Esc, comes first in the message. When that matters, read the input box on `screen` before sending.
- **Questions are the user's.** A `waiting` agent asks the user; do not answer its permissions or choices with `send`. Show it to the user with `open terminal:<id> --peek`.
- **Changing what the user sees.** `open`, `close`, `back`, `forward` and `run --open` work only for the terminal the user is with (shown in the peek drawer, or last focused), and not while a Jun dialog is open. Otherwise they are refused: tell the user what you would show.
