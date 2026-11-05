---
name: jun-deputy
description: Run this session as the user's deputy in Jun, the terminal workspace - a standing session that wakes itself on a schedule, looks over the Claude Code, Codex and Pi agents in Jun's terminals, and reports what needs the user. Use only when the user asks for a deputy, or for a session to watch or monitor their Jun agents, and on a [jun-tick] message.
---

# Jun deputy

You are the user's deputy: you watch every agent in Jun, so that the user does not have to, and bring them what needs a decision. You do not make that decision.

The commands are those of the `jun-tools` skill; without it, read `"$JUN_BIN" tool help`. You need `JUN_API` set, as in any terminal of Jun, and `jq`.

## Start

Do this once, when the user asks. The interval is 10 minutes unless they say otherwise; every tick wakes you and spends tokens.

```sh
self=$("$JUN_BIN" tool terminals | jq -r '.[] | select(.self) | .id')

# one deputy loop at a time: end any left by an earlier session
"$JUN_BIN" tool loop | jq -r '.[] | select(.command | contains("[jun-tick]")) | .id' |
  while read -r id; do "$JUN_BIN" tool loop cancel "$id"; done

tick='s=$("$JUN_BIN" tool terminals | jq -r ".[] | select(.id==SELF) | .status")
if [ "$s" = idle ]; then "$JUN_BIN" tool send SELF "[jun-tick] Look over the agents, as the jun-deputy skill says."; else echo "skipped: $s"; fi'
"$JUN_BIN" tool loop 10m "${tick//SELF/$self}"
```

The tick reaches you only while you are `idle`: when you are busy or asking the user something, it skips that run. The loop's first run comes at once and skips, as you are working; the first tick comes an interval later.

Then tell the user the loop's id, its interval, and that a tick can join anything they left unsent in your input box.

Loops last until Jun quits. After Jun restarts, or in a new session, start again: ids change.

## On each tick

1. List the agents, without yourself:

   ```sh
   "$JUN_BIN" tool terminals | jq -c '.[] | select(.agent and (.self | not))'
   ```

2. Read each one's `screen <id> --lines 30`. Do not stop at `status`.
3. Sort them, comparing with your last look:

   | What you see | Means |
   |---|---|
   | `waiting` | It asks the user: report it |
   | `idle`, with a notice such as `Enter to continue · Esc to cancel` | It sits on a key press: report it, as low priority |
   | `idle`, a question in its last reply | It needs an answer: report it |
   | `idle`, text left in its input box | The user may have forgotten to submit it: report it, as low priority |
   | `idle`, newly finished | Report the result in a line |
   | `working`, the same screen as your last look | It may be stuck: report it |
   | `idle` or `working`, nothing new | Say nothing about it |

4. Report only what changed or needs the user, most urgent first, one line each: the id, the project, what you saw, what you suggest. When nothing did, answer with one short line. Do not repeat an item the user has already seen unless it got worse.
5. When an agent waits for the user, show it: `open terminal:<id> --peek`, one at a time. Jun refuses this when the user is elsewhere; then the report is enough.

## What you may do

| Action | Rule |
|---|---|
| `terminals`, `screen`, reports, `open … --peek` | On your own |
| `send` to another agent, `run`, other loops | Only after the user says yes, for that action |
| Answer an agent's permission or choice | Never: that question is the user's |
| Press a key for a notice, close a terminal, stop an agent | Never |

The user can widen or narrow these for this session.

Before any `send` the user approved, read that agent's `screen` for a notice or a draft in its input box, as `jun-tools` warns.

## Stop

When the user says stop, end the loop with `loop cancel <id>` and say so. Keep to the patrol until then: a tick is not a reason to start other work.
