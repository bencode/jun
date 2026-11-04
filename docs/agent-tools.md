# Agent Tools

Programs in Jun's terminals, agents above all, can ask the Jun that runs them about its terminals with `jun tool`. An agent you work with, in a project of its own, can then see what the other agents are doing and read their screens. Jun gives the tools only; how an agent uses them, as a coordinator or otherwise, is up to the instructions or skill you give it.

`"$JUN_BIN" tool help` prints what this page says, for an agent to read.

## Commands

| Command | Output |
|---|---|
| `jun tool help` | The commands, fields, environment and exit status |
| `jun tool terminals` | Jun's running terminals, as a JSON array, by id |
| `jun tool screen <id> [--lines N]` | The last N lines of the terminal's text, as plain text |

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
- **Outside Jun's terminals**: the read-only commands work from anywhere with `JUN_API` set, for instance to debug, and then `self` is `false` everywhere.

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
