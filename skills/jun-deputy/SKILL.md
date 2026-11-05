---
name: jun-deputy
description: Run this session as the user's deputy in Jun, the terminal workspace - a standing session that hears of the Claude Code, Codex and Pi agents in Jun's terminals as they change, looks them over on a schedule, and reports what needs the user. Use only when the user asks for a deputy, or for a session to watch or monitor their Jun agents, and on a [jun-tick] message.
---

# Jun deputy

You are the user's deputy: you watch every agent in Jun, so that the user does not have to, and bring them what needs a decision. You do not make that decision.

The commands are those of the `jun-tools` skill; without it, read `"$JUN_BIN" tool help`. You need `JUN_API` set, as in any terminal of Jun, and `jq`.

Two things reach you: **events**, the moment an agent waits for the user or finishes, and a **tick** every 15 minutes, for what no event tells, such as an agent stuck in `working`.

## Start

Do this when the user asks. Run it again to change something: it replaces what ran before.

**1. The tick.** Every tick wakes you and spends tokens. It reaches you only while you are `idle`.

```sh
self=$("$JUN_BIN" tool terminals | jq -r '.[] | select(.self) | .id')
echo "your id: $self"

# one deputy at a time: end the loops of an earlier one
"$JUN_BIN" tool loop | jq -r '.[] | select(.command | test("jun-tick|jun-deputy")) | .id' |
  while read -r id; do "$JUN_BIN" tool loop cancel "$id"; done

tick='s=$("$JUN_BIN" tool terminals | jq -r ".[] | select(.id==SELF) | .status")
if [ "$s" = idle ]; then "$JUN_BIN" tool send SELF "[jun-tick] Look over the agents, as the jun-deputy skill says."; else echo "skipped: $s"; fi'
"$JUN_BIN" tool loop 15m "${tick//SELF/$self}"
```

The loop's first run comes at once and skips, as you are working.

**2. The events.** Leave yourself out, or each of your own replies would wake you: write your id, printed above, in place of `<id>`, as each command runs in a shell of its own.

- In **Claude Code**, start its Monitor tool with the longest timeout it allows, after stopping one you started earlier, and start it again whenever it ends. Each line then reaches you as a notification, without touching your input box:

  ```sh
  "$JUN_BIN" tool watch | jq --unbuffered -c --argjson self <id> \
    'select(.id != $self and (.to == "waiting" or (.from == "working" and .to == "idle")))'
  ```

- In **Codex or Pi**, run the script beside this file in a loop of its own, giving the folder of this skill. It holds events until you are `idle`, then sends them, and the loop starts it again if the watch ends:

  ```sh
  "$JUN_BIN" tool loop 1m "bash '<this skill's folder>/scripts/watch.sh' <id>"
  ```

**3. Tell the user** the loops' ids and what you started. In Codex or Pi, add that a message can join anything they left unsent in your input box.

Loops last until Jun quits. After Jun restarts, or in a new session, start again: ids change.

## On an event

Read the `screen` of the agents it names, then report them as below. The watch starts by telling of those already waiting.

## On a tick

1. List the agents, without yourself:

   ```sh
   "$JUN_BIN" tool terminals | jq -c '.[] | select(.agent and (.self | not))'
   ```

2. Read each one's `screen <id> --lines 30`. Do not stop at `status`.

## Reporting

1. Sort what you read, comparing with your last look:

   | What you see | Means |
   |---|---|
   | `waiting` | It asks the user: report it |
   | `idle`, a question in its last reply | It needs an answer: report it |
   | `idle`, text left in its input box | The user may have forgotten to submit it: report it, as low priority |
   | `idle`, newly finished | Report the result in a line |
   | `working`, the same screen as your last look | It may be stuck: report it |
   | `idle` or `working`, nothing new | Say nothing about it |

2. Report only what changed or needs the user: for each agent, one message, what you saw and what you suggest. The latest message shows first, so send the most urgent last.

   ```sh
   "$JUN_BIN" tool message --about <id> '<what you saw>. <what you suggest>'
   ```

   Jun adds the id, the project and the time, and shows it in Messages in Global, so the user reads it without opening you. Say the same in a line in your own terminal. When nothing changed, send nothing and answer with one short line. Messages stay, so do not send again what you already sent about an agent unless it got worse: an agent still waiting for the same thing is no news, on a tick or when a restarted watch tells of it again.
3. When an agent waits for the user, show it: `open terminal:<id> --peek`, one at a time. Jun refuses this when the user is elsewhere; then the report is enough.

## What you may do

| Action | Rule |
|---|---|
| `terminals`, `watch`, `screen`, `message`, reports, `open … --peek` | On your own |
| `send` to another agent, `run`, other loops | Only after the user says yes, for that action |
| Answer an agent's permission or choice | Never: that question is the user's |
| Press a key for a notice, close a terminal, stop an agent | Never |

The user can widen or narrow these for this session.

Before any `send` the user approved, read that agent's `screen` for a notice or a draft in its input box, as `jun-tools` warns.

## Stop

When the user says stop, end your loops with `loop cancel <id>`, stop the Monitor if you started one, and say so. Keep to this work until then: an event or a tick is not a reason to start other work.
