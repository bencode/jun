---
name: jun-tools
description: Work with the other agents and terminals of Jun, the terminal workspace, through "$JUN_BIN" tool. List Jun's terminals and the status of the Claude Code, Codex and Pi agents in them; read a terminal's screen; send a message to another agent; tell the user something; show the user a terminal, a file or a branch's changes; run a command in a new terminal; run one on a schedule. Use when JUN_API is set and a task involves another agent or terminal in Jun, such as checking on agents, handing work to one, passing a message on, or showing the user something.
---

# Jun tools

Jun runs agents in terminals. `jun tool` lets a program in one of them see the others and act on them. Jun gives the tools only: how you use them is up to your task and the user's instructions.

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
| `watch` | Runs until Jun quits: agents' status as it changes, a JSON line each with `id`, `project`, `agent`, `title`, `from`, `to` |
| `screen <id> [--lines N]` | The last N lines of a terminal's text, scrollback included (default 40, at most 500) |
| `send <id> <text>` / `send <id> -` | Submits the text to the agent in that terminal, as pasted then Enter; `-` reads standard input |
| `open terminal:<id> [--peek]` | Shows that terminal to the user: its tab, or the peek drawer |
| `open diff:<id> [--in <perspective>]` | Opens the changes of the branch that terminal is on |
| `open file:<path>` | Opens the file in a tab, read-only, wherever it is; opened again, reads it again |
| `open project:<name>` / `open perspective:<name>` | Switches the user's project or perspective |
| `close terminal:<id>` / `close diff:<id>` / `close file:<path>` / `close peek` | Closes what `open` opened; a terminal's shell keeps running |
| `close perspective:<name>` | Deletes the perspective; with terminals in it, Jun asks the user to confirm |
| `back` / `forward` | Steps through where the user has been |
| `message [--title <t>] [--kind ask\|warn\|done\|info] [--link <resource>] <text>` / `… -` | Tells the user, in Messages in Global, without disturbing them, and prints the message's id; `-` reads standard input, and with a title the text may go |
| `message update <id> [--kind <kind>] [<text>]` / `… -` | Changes the message's kind, adds the text as a reply, or both |
| `message` | The messages as JSON, the latest first: `id`, `time`, `kind`, `title`, `text`, `project`, `link`, `replies` |
| `run [--dir <path>] [--open] <command…>` | Starts a terminal running the command; prints its id |
| `loop <every> [--dir <path>] <command…>` | Runs the command every interval (at least `1m`) in a terminal of its own; prints the loop's id |
| `loop` / `loop cancel <id>` | Lists the loops as JSON / ends one |

Ids are terminal ids, the same for every command. They last while Jun runs.

## Status

`status` comes with `agent`:

- `waiting`: the agent asks the user something, such as a permission or a choice, or Claude Code shows a notice that waits for Enter.
- `working`: it is busy.
- `idle`: nothing runs. Idle does not tell done from stuck or from a question asked in its reply: read its `screen`.

## Reaching someone

| To | Use |
|---|---|
| Answer the user you are talking with | Your reply: no `message` as well |
| Tell the user something they are not here to read | `message` |
| Show the user something they asked to see | `open`, which works only while they are with you |
| Have another agent do something | `send` |

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
"$JUN_BIN" tool open file:report.md    # a file, from your directory or any other
"$JUN_BIN" tool open terminal:7 --peek # a terminal, in the peek drawer
```

**Tell the user something.** `message` shows it in Messages in Global, for the user to read when they look. Any terminal may send one at any time. Messages last until Jun quits.

- `--title`: what happened, in one line of at most 200 bytes.
- The text: what the user needs beyond that, at most 4 KiB.
- `--kind` gives it its icon: `ask` when the user has something to do, `warn`, `done`, or `info`, the default.
- `--link` names what it is about, as `open` does: `terminal:<id>`, `diff:<id>` with the same id from `terminals`, `file:<path>` or `project:<name>`. Jun tells the project from it, and a click takes the user there, peeking an agent. A resource that does not exist is an error and shows nothing.

Give `--link` whenever the message is about something: without it, the message is plain news, which takes the user nowhere. That holds for your own work too: `$JUN_TERMINAL` is your terminal's id.

```sh
"$JUN_BIN" tool message --title 'Asks to open a PR' --kind ask --link terminal:7 'I suggest yes.'
"$JUN_BIN" tool message --title 'Parser fix ready to review' --kind done --link "diff:$JUN_TERMINAL"
"$JUN_BIN" tool message 'The nightly run failed: 2 tests, both in the parser.'
printf '%s\n' "$long_message" | "$JUN_BIN" tool message --link "file:report.md" -
```

**Follow a message up.** `message` prints the id; the user sees `12` as `M12`, and may call it that. A message is one matter: `ask` means the user has something to do, so when they have done it, make it `done`. Add what happens next as replies, rather than new messages. A reply moves the message to the top. Any terminal may update any message.

```sh
"$JUN_BIN" tool message update 12 --kind done 'You agreed; it goes on.'
"$JUN_BIN" tool message update 12 'PR #53 is open, the tests pass.'
"$JUN_BIN" tool message | jq -c '.[] | select(.kind == "ask")'    # what still waits for the user
```

**Start another agent or a job.**

```sh
"$JUN_BIN" tool run --dir ../api codex
"$JUN_BIN" tool run 'make test 2>&1 | tail -20'
```

The new terminal belongs to your project and has no tab: follow it with `screen <id>`, or pass `--open` to show it to the user.

**Hear of changes as they happen.** For an event, use `watch`, not a short loop:

```sh
"$JUN_BIN" tool watch | jq --unbuffered -c 'select(.to == "waiting")'
```

It starts with a line for each agent, `from` empty. `idle` comes after it has held for 3 seconds, the rest at once. When Jun quits or the reader falls behind, it ends with status 1: run it again.

**Check in on a schedule.**

```sh
tick='[ "$("$JUN_BIN" tool terminals | jq -r ".[] | select(.id==7) | .status")" = idle ] && "$JUN_BIN" tool send 7 "Report your progress and anything blocking you."'
"$JUN_BIN" tool loop 10m "$tick"
"$JUN_BIN" tool loop                # the loops
"$JUN_BIN" tool loop cancel 12      # end one
```

The command checks that the agent is `idle` first, since `send` does not look: see the pitfalls. It needs `jq`.

A check a script can make needs no agent: have the loop tell the user itself, which spends no tokens. Here the link leads to the loop's terminal, which shows each run:

```sh
"$JUN_BIN" tool loop 1h --dir ~/work/api 'make test || "$JUN_BIN" tool message --title "Tests fail in api" --kind warn --link "terminal:$JUN_TERMINAL"'
```

## Pitfalls

- **Quoting.** The words of a command are joined by spaces into one command line, as `ssh` joins them. To keep quotes inside it, quote the whole command, as the loop above does.
- **Loops run as scripts.** A loop runs its command with `$SHELL -l -c`: login files apply, `.zshrc` does not. Write absolute paths, or `"$JUN_BIN"`. A run waits for the one before to end.
- **Loops cost.** Each run that sends wakes an agent and spends its tokens. Send only what needs an agent's judgement, choose the interval for the task, and cancel loops you no longer need: they last until cancelled or Jun quits.
- **Only agents take `send`.** A terminal running no agent refuses it, since the text would run as a command.
- **`send` does not look.** It pastes and presses Enter whatever the agent shows: on a `waiting` question, that Enter can answer it. Read the `screen` before you send, and have a scheduled send check the status.
- **Drafts.** `send` goes into the agent's input box as is. Text already there, such as the user's draft or a prompt Claude Code put back after Esc, comes first in the message. When that matters, read the input box on `screen` before sending.
- **Questions are the user's.** A `waiting` agent asks the user; do not answer its permissions or choices with `send`. Tell the user with `message --kind ask --link terminal:<id>`: a click peeks it.
- **A file shown outside the project does not follow its changes.** After writing to it, `open file:` again.
- **Changing what the user sees.** `open`, `close`, `back`, `forward` and `run --open` work only for the terminal the user is with (shown in the peek drawer, or last focused), and not while a Jun dialog is open. Otherwise they are refused: tell the user what you would show.
