# Agent Tools

Programs in Jun's terminals, agents above all, can ask the Jun that runs them about its terminals, and change what it shows, with `jun tool`. An agent you work with, in a project of its own, can then see what the other agents are doing, read their screens, bring you a terminal or a branch's changes to look at, pass a message on to another agent, and run commands in terminals of their own, once or again and again. Jun gives the tools only; how an agent uses them, as a coordinator or otherwise, is up to the instructions or skill you give it.

`"$JUN_BIN" tool help` prints what this page says, for an agent to read.

## As a skill

Two skills build on these tools, in the [Agent Skills](https://agentskills.io) format, which Claude Code, Codex and Pi read:

- [`jun-tools`](../skills/jun-tools/SKILL.md) teaches an agent the tools: when to use them, the common work, and the pitfalls.
- [`jun-deputy`](../skills/jun-deputy/SKILL.md) makes a session your deputy: it hears of your agents as they change, looks them over on a schedule, and reports what needs you.

Install them for your user with the [skills](https://github.com/vercel-labs/skills) CLI:

```sh
npx skills add bencode/jun -g
```

Or copy the folders in `skills/` to `~/.claude/skills/` for Claude Code (`$CLAUDE_CONFIG_DIR/skills/` when that is set), or to `~/.agents/skills/` for Codex and Pi.

## Commands

| Command | Output |
|---|---|
| `jun tool help` | The commands, fields, environment and exit status |
| `jun tool terminals` | Jun's running terminals, as a JSON array, by id |
| `jun tool terminals update <id> [--name <name>] [--every <every>] [--at <HH:MM>] [--text <text>] [--no-timer]` | Names the terminal, or drops its name for `''`; sets or drops its timer. Prints nothing. |
| `jun tool roles` / `roles --mine` / `roles <name> [--limit <n>]` | The [roles](#roles), as a JSON array, by name / the role of the caller's session / that role with its latest sessions |
| `jun tool watch` | Agents' status as it changes, as JSON, a line each, until it ends |
| `jun tool screen <id> [--lines N]` | The last N lines of the terminal's text, as plain text |
| `jun tool open <target> [--in <perspective>] [--peek]` | Shows the target. Prints nothing. |
| `jun tool close <target>` | Closes the target. Prints nothing, or what it leaves pending. |
| `jun tool back` / `jun tool forward` | One step back or forward through where you have been, as ⌘[ and ⌘] go. Prints nothing. |
| `jun tool send <id> <text>` / `jun tool send <id> -` | Submits the text to the agent in the terminal, or holds it while the user is typing there. Prints nothing, `sent, not started yet` or `held: …`. |
| `jun tool message [--title <title>] [--kind <kind>] [--link <resource>] <text>`, or `-` for the text | Shows the text to the user, in [Messages](agents-view.md#messages). Prints the message's id. |
| `jun tool message update <id> [--kind <kind>] [<text>]`, or `-` for the text | Changes the message's kind, adds a reply, or both. Prints nothing. |
| `jun tool message` | The messages, as a JSON array, the latest first |
| `jun tool run [--project <name>] [--dir <path>] [--open] <command…>` | Starts a terminal running the command. Prints its id. |
| `jun tool join <id>` | Makes the agent in that terminal the caller's lead. Prints nothing. |
| `jun tool leave` | Drops the caller's lead. Prints nothing. |
| `jun tool loop <every> [--dir <path>] <command…>` | Runs the command every interval in a terminal of its own. Prints the loop's id. |
| `jun tool loop` | The loops, as a JSON array |
| `jun tool loop cancel <id>` | Ends the loop and its terminal. Prints nothing. |

Run them as `"$JUN_BIN" tool …`, which works whatever `PATH` and aliases say.

### terminals

```json
[
  {
    "id": 3,
    "name": "Euler",
    "project": "web",
    "dir": "/path/to/web",
    "title": "Fix the login form",
    "agent": "claude",
    "status": "waiting",
    "self": false,
    "lead": 2,
    "timer": { "every": "30m", "next": "2026-11-20T12:30:00+08:00", "text": "Look for new P1 and P2 tickets" }
  }
]
```

| Field | Meaning |
|---|---|
| `id` | The terminal's id, for `screen`. Ids hold while Jun runs; a new run numbers its terminals anew. |
| `name` | What agents call it, when it has [one](#terminals-update); absent otherwise |
| `project` | The project the terminal belongs to |
| `dir` | Its shell's directory |
| `title` | Its title, as the Agents view or its tab shows it |
| `agent` | `claude`, `codex` or `pi`, when one runs in it; absent otherwise |
| `status` | `waiting`, `working` or `idle`, as the [Agents](agents-view.md) view shows it; with `agent` only |
| `self` | `true` for the terminal the caller runs in |
| `lead` | The terminal of the agent that [leads](#join-and-leave) it, when one does |
| `timer` | Its [timer](#timers), when it has one: `every`, absent for one that fires once; `next`, when it fires next; `text`, absent for none |

Terminals with closed tabs are listed too: their shells keep running. Terminals whose shell has ended are not.

### terminals update

`terminals update <id>` changes a terminal: its name, its timer or both. Each option comes once, in any order. Any terminal may change any running one.

`--name <name>` names a terminal, for you and agents to call it by, as in "hand this to Euler" or "what is Euler doing?". An agent finds the id in `terminals` by the name, then uses the id: the other commands take ids only.

- **Who**: an agent names itself, which makes the name a [role](#roles); a lead may also name the terminals it runs. A new name takes the place of the one before; `--name ''` drops it.
- **A name** is one line of at most 64 bytes, not a number, and no other running terminal's, whatever its case.
- **Shown** in [Agents](agents-view.md#order-and-status), beside the terminal's number, and given as `name` by `terminals` and `watch`.
- **It ends** with the terminal, or when Jun quits; a role stays.

#### Roles

A name is a role: an entity that lasts, and the sessions that took it are its memory. When an agent names its own terminal, Jun records the agent's session as the role's: Claude Code, Codex and Pi give their commands its id, in `CLAUDE_CODE_SESSION_ID`, `CODEX_THREAD_ID` and `PI_SESSION_ID`, and `jun tool` sends it along. A name given to another agent's terminal records no session.

- **Kept** beside Jun's configuration, a file for each role that changes are only added to, so a role with many sessions costs no more to change. A role lasts while it has a session, across Jun's runs.
- **Renamed**, the agent's session moves to the new role; `--name ''` takes it out. A role with no session left goes, and its name with it.
- **Taken on**: naming a terminal a role's name that no running terminal has makes the agent's session part of that role. Do so to carry a role on, not by chance: look at `roles` first.
- **Resumed**, a session keeps its id: `roles --mine` finds its role, for the agent to take the name back.

`roles` lists each role with `name`, `terminal` (the running one of that name, if any), `count` (its sessions) and `last`, its latest session: `agent`, `id`, `dir`, and `at`, when it took the role. `roles <name>` gives that role with its latest `sessions`, the latest first, 20 unless `--limit` says, and `roles --mine` the same for the role of the caller's session, or an empty array.

Jun does not read the sessions: the [jun-tools](#as-a-skill) skill tells an agent where each agent keeps them and how much of them to read.

#### Timers

A timer wakes the agent in a terminal on a schedule, as a lead that looks for new tickets every 30 minutes. Jun only signals it; what it does then is its own to know, from its skill and context.

```sh
"$JUN_BIN" tool terminals update 7 --every 30m --text 'Look for new P1 and P2 tickets'
"$JUN_BIN" tool terminals update 7 --at 15:00 --text 'Check the PR for reviews'
```

- **`--every <every>`** signals the agent every interval, written as `loop`'s: `90s`, `5m` or `1h`, at least `1m`. The first comes one interval on, or at `--at`.
- **`--at <HH:MM>`** signals it once, at the next time the clock reads so, local time: today, or tomorrow once past. With `--every`, it is the first of them.
- **`--text <text>`** follows the signal's tag, one line of at most 500 bytes. The signal is `[jun-timer 30m] <text>`, or `[jun-timer 15:00] <text>` for one that fires once.
- **`--no-timer`** drops the timer. A new timer takes the place of the one before: a terminal has one at most.
- **When**: once it is due, and the agent idle with nothing in its input. A signal missed while the agent works or you type in its input comes once, when it can; then the next is an interval on. One that fires once goes.
- **Needs an agent** in the terminal to be set. Jun shows it in [Agents](agents-view.md#layout), and forgets it with the terminal, or when Jun quits.

### watch

Runs until Jun quits, printing a line as soon as there is news of an agent:

```json
{"id":3,"name":"Euler","project":"web","agent":"claude","title":"Fix the login form","from":"working","to":"waiting"}
```

| Field | Meaning |
|---|---|
| `id`, `name`, `project`, `agent`, `title`, `lead` | As `terminals` has them. A new name alone is no news: it comes with the agent's next line. |
| `from` | The status before; empty for an agent that is new |
| `to` | The status now; empty for an agent that is gone, or whose terminal closed |

- **It starts** with a line for each agent, `from` empty, so that a change just before it began is not lost.
- **When Jun tells**: Jun reads the agents' status every second. `waiting`, `working`, a new agent and one gone are told at once. `idle` is told once it has held for 3 seconds: agents go idle for a moment between steps, and that is no news.
- **It ends** with status 1 when Jun quits, or when the reader falls too far behind, for Jun never waits for a reader. Run it again to catch up, from a loop for instance.
- **Filter** with `jq`, which needs `--unbuffered` in a pipeline to pass each line on as it comes:

```sh
"$JUN_BIN" tool watch | jq --unbuffered -c 'select(.to == "waiting")'
```

### screen

- The terminal's text with its scrollback, after the blank lines below the cursor: 40 lines by default, 1 to 500 with `--lines`.
- A line the terminal wraps on screen counts as one.
- An agent's text includes what it draws around its messages: frames, status bars and its input box.

### Resources

A resource names something in Jun. `open` and `message --link` take these forms, and `close` all but `project:`. `open` and `close` also take the `perspective:` and `peek` of the table below, which are no resources: they depend on where the user is.

| Resource | What it is |
|---|---|
| `terminal:<id>` | A terminal, with the `id` that `terminals` and `watch` give |
| `diff:<id>` | The changes of the branch that terminal is on: the same id |
| `file:<path>` | A file. A relative path starts from the caller's directory |
| `project:<name>` | A project, whatever the case of its name |

### open and close

| Target | `open` | `close` |
|---|---|---|
| `terminal:<id>` | Shows the terminal's tab, switching project and perspective as needed, and reopens its tab when it was closed. With `--peek`, an agent's terminal shows in the [peek drawer](agents-view.md#peek) instead, or its tab when that is on screen. | Closes its tab in the current perspective. The shell keeps running. |
| `diff:<id>` | Opens the changes of the branch the terminal is on, as its agent's Review Changes does, in the current perspective. With `--in <perspective>`, first switches to that perspective of the terminal's project, made when there is none. | Closes the Diff of that branch in the current perspective. |
| `file:<path>` | Opens the file in a tab of its own, read-only, wherever it is: see [Files](files.md#files-outside-the-project). Opened again, the file is read again. | Closes its tab in the current perspective |
| `project:<name>` | Switches to the project | — |
| `perspective:<name>` | Switches to the perspective of the current project, made when there is none | Deletes it, as Delete Perspective… does: with terminals in it, Jun asks you to confirm, and prints `waiting for the user to confirm`. A project keeps at least one perspective. |
| `peek` | — | Closes the peek drawer |

- Names match whatever their case.
- The first time a project's repository is asked for, Jun reads its branches and worktrees; `diff:` waits a few seconds for them. When the terminal is in none of the worktrees read, as when its agent made one since, Jun reads them again once.

A review, for instance, takes two calls: `open diff:7 --in Review`, and once you are done, `back`.

### When Jun changes what it shows

`open`, `close`, `back` and `forward` change what you see, so Jun runs them only:

- **For the terminal you are with**: the one shown in the peek drawer, or else the terminal you last focused. Otherwise: `the user is not with you`.
- **While no dialog waits for you**: the command palette, a form, a confirmation, an error or the shortcuts panel. Otherwise: `Jun is waiting for the user`.
- **On another terminal**: a terminal cannot open or close itself (`that is your own terminal`).

Opening another terminal gives it the focus, so you are then with that terminal: the caller's next change is refused until you come back to it.

### send

- The text goes in as a paste, then Enter, as when you paste a message and submit it: its line breaks stay in one message. Checked with Claude Code 2.1.318, Codex 0.166.0 and Pi 1.5.0.
- With `-`, the text is standard input, for long or multi-line messages. Line breaks at its end go; at most 1 MiB.
- Any agent takes it, whatever it is doing, the caller's own terminal too. A terminal running no agent does not: text there would run as a command.
- Once sent, Jun waits up to 5 seconds for the agent to work. If it does not by then, `send` prints `sent, not started yet`, still with status 0.
- While the user has typed something in the agent's input and not submitted it, Enter would submit it with the text. Jun holds the text instead, and submits it once the input is empty, as the user submits or clears it; `send` prints `held: …` at once, with status 0. Texts sent to an agent while others are held follow them, in order. Held texts go if the agent or Jun quits first. Placeholders and suggested prompts, which agents show dim, are not the user's. While texts are held, the agent's terminal shows how many at its bottom right, as `✉ 2 waiting`, and so does its row in [Agents](agents-view.md#layout).

### message

`message` tells the user something without disturbing them: it appears in [Messages](agents-view.md#messages), in Global.

```sh
"$JUN_BIN" tool message --title 'Asks to open a PR' --kind ask --link terminal:7 'I suggest yes.'
```

- **Who may send**: any terminal of Jun, at any time, whatever the user is looking at. Outside Jun's terminals it is refused.
- **`--title`**: one line, at most 200 bytes. With a title, the text may go.
- **`--kind`**: `ask` when the user has something to do, `warn`, `done`, or `info`, the default. It gives the message its icon and color.
- **`--link`**: the [resource](#resources) it is about, where a click takes the user. Jun tells the project from it, so a loop reporting on every agent gives only this. Without it, the message is plain news: no tag, and a click does nothing.
- **A resource that does not exist** is status 1, and shows nothing: the terminal, file or project, or a terminal in no worktree for `diff:`. `peek` and `perspective:` are not for messages.
- **The text**: with `-`, standard input. Line breaks at its end go; at most 4 KiB.
- **How long it stays**: Jun keeps the last 500, until it quits. A terminal's id lasts as long, so a message's link never points at another terminal.
- **Its id**: `message` prints it. Jun shows it as `M12`; the commands take `12`.

#### Updating a message

`message update` changes what the user sees of a message: its kind, a reply below it, or both, at least one of them.

```sh
"$JUN_BIN" tool message update 12 --kind done 'You agreed; it goes on.'
"$JUN_BIN" tool message update 12 'PR #53 is open, the tests pass.'
"$JUN_BIN" tool message update 12 --kind done
```

- **`ask` means the user has something to do**, so `ask` becoming `done` tells them it is handled. Jun does not do this itself: whoever watches the agent decides.
- **A reply** has its own time, and moves the message to the top. A new kind alone leaves the message where it is.
- **The text**: as for `message`. A message keeps its last 50 replies.
- **Who may update**: any terminal of Jun, whoever sent the message. Outside Jun's terminals it is refused.
- **A message that does not exist** is status 1, as is one Jun no longer keeps.

#### Reading messages back

`message` with nothing after it lists them, for instance to find what still waits for the user:

```sh
"$JUN_BIN" tool message | jq -c '.[] | select(.kind == "ask")'
```

```json
[
  {
    "id": 12,
    "time": "2026-11-05T19:30:00+08:00",
    "kind": "done",
    "title": "Asks to open a PR",
    "text": "I suggest yes.",
    "project": "web",
    "link": "terminal:7",
    "replies": [
      { "time": "2026-11-05T19:32:00+08:00", "text": "You agreed; it goes on." }
    ]
  }
]
```

`link` is as `--link` takes it; `title`, `text`, `project`, `link` and `replies` are absent when there are none.

### run and loop

Both run a command in a terminal of the caller's project, in `--dir`, by default the caller's current directory. The terminal has no tab: `open terminal:<id>` shows it, and `terminals` lists it.

- **`run --project <name>`** starts the terminal in that project instead, whatever the case of its name, in the project's directory unless `--dir` says otherwise. A project that does not exist is status 1.
- **A terminal an agent runs** has that agent as its [lead](#join-and-leave). One that a shell or a loop runs has none.

- **The command**: the words after the options, joined by spaces into one command line, as `ssh` joins them. Quote a command whose own quotes must stay: `run 'git commit -m "a b"'`.
- **run** types the command into a new shell, as if you had, so your shell's configuration applies; the shell stays once the command ends. `run codex` starts Codex there. With `--open`, the terminal shows in a tab, focused, under the same rules as `open`.
- **loop** runs the command again and again, each run the interval after the one before ends, so that runs never overlap. The interval is written as `90s`, `5m` or `1h`, at least `1m`.
  - It runs with `$SHELL -l -c`, as a script runs: your login files apply, not `.zshrc`. Write absolute paths, or `"$JUN_BIN"`.
  - Each run shows in the loop's terminal with its time and exit status.
  - A loop's id is its terminal's, for `screen` and `open` too.
- **Ending a loop**: `loop cancel <id>`, Control-C in its terminal, or quitting Jun. Loops are not saved.

Jun runs what it is asked: a loop may run `run`, or start another loop. A loop that sends to an agent wakes it as `send` does:

```sh
"$JUN_BIN" tool loop 10m "$JUN_BIN" tool send 7 "'How is it going? Report anything blocking you.'"
```

`send` presses Enter whatever the agent shows, so a loop that may meet a question should check the agent's `status` first. To wake an agent on a schedule, a [timer](#timers) does that, idle only; a loop is for commands.

### loop

```json
[
  {
    "id": 12,
    "every": "10m",
    "command": "/path/to/jun tool send 7 'How is it going?'",
    "project": "web",
    "dir": "/path/to/web"
  }
]
```

### join and leave

An agent can lead others: hand them work, follow them and answer them. Jun records which agent leads which terminal, and shows it in [Agents](agents-view.md#order-and-status); what leading means is up to the agents.

- **A lead comes** from `run`, for the terminal an agent starts, or from `join <id>`, which a terminal calls to be led by the agent in terminal `<id>`, as a session you started yourself and hand over does.
- **One lead at a time**: a join takes the place of the lead before.
- **Refused**: a terminal running no agent, the caller's own, and a terminal the caller leads, however far down.
- **It ends** with `leave`, or when either terminal ends. The terminals the ended one led then have no lead, not the one above it. Leads last until Jun quits.
- **A lead follows its own** with `lead` in `terminals` and `watch`:

```sh
"$JUN_BIN" tool watch | jq --unbuffered -c "select(.lead == $JUN_TERMINAL)"
```

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

A tool command without `tool`, such as `jun open file:x.md`, exits with status 2 and prints the command to run instead, unless a directory or file of that name is in the working directory.
