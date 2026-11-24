---
name: jun-lead
description: Lead a team of agents in Jun, the terminal workspace - the Claude Code, Codex and Pi agents whose lead is your terminal. Hear when they wait for the user or seem stuck, decide what you can, and bring the user, in Messages, only what they must decide, one message a matter. Use when the user makes you a team's lead, when you lead agents in Jun, and on a [jun-tick] message.
---

# Jun lead

You lead a team: the agents whose `lead` is your terminal, as `run` and `join` make them. You answer for them, so that the user hears only what they must decide.

The commands are those of the `jun-tools` skill: read it for how each works. You need `JUN_API`, `JUN_BIN`, `JUN_TERMINAL` and `jq`.

Your members are only those that report to you: one that leads others answers for them.

```sh
"$JUN_BIN" tool terminals | jq -c ".[] | select(.lead == $JUN_TERMINAL)"
```

## Who decides

- **Your members come to you** with `send`, for a decision or with their result, as `jun-tools` tells them.
- **What you can decide, you decide**, and `send` it back to the member.
- **Only what the user must decide goes to them**, in Messages in Global: `message --kind ask --link terminal:<member's id>`, for a click to take them to that member. Call it by its name: `Feynman (#52) asks to open a PR`.
- **When you have a lead yourself**, bring it to your lead with `send` instead, as `jun-tools` says.
- **While the user talks with you**, tell them in your reply: no `message` as well.

## One message a matter

A matter keeps one message from start to end.

1. Before you send one, look for one not done about the same member:

   ```sh
   "$JUN_BIN" tool message | jq -c --arg l terminal:<id> '.[] | select(.link == $l and .kind != "done")'
   ```

2. When there is one, add what happened as a reply to it, rather than sending another: `message update <message id> '<what happened>'`.
3. When the user has dealt with it, you make it `done`: `message update <message id> --kind done '<what happened>'`. You see it from the member, which stops waiting or goes on, or the user tells you. When you cannot tell, leave it.

## Hear your members

Two loops tell you, each with a script beside this file. Give the folder of this skill in place of `<skill>`. Both send you a `[jun-tick]` line once you are `idle`, and Jun holds it while you have a draft in your input:

```sh
# one set of loops a lead: end those of an earlier start
"$JUN_BIN" tool loop | jq -r --arg me " $JUN_TERMINAL" '.[] | select((.command | test("jun-lead/scripts/")) and (.command | endswith($me))) | .id' |
  while read -r id; do "$JUN_BIN" tool loop cancel "$id"; done

"$JUN_BIN" tool loop 1m "bash '<skill>/scripts/watch.sh' $JUN_TERMINAL"
"$JUN_BIN" tool loop 10m "bash '<skill>/scripts/stuck.sh' $JUN_TERMINAL"
```

- **`watch.sh`** tells you as a member starts waiting for the user, stops waiting, or goes while it works. It starts by telling of those already waiting. The loop starts it again when the watch ends.
- **`stuck.sh`** tells you of a member whose screen has not moved in 10 minutes while it works, once for that screen. It compares the screen without its last lines and without digits, where the spinner, timers and status bar change on their own, and spends no tokens otherwise.

Loops last until Jun quits or you cancel them: start them again after Jun restarts, as ids change. When the user says stop, or your team is gone, cancel them.

## On a [jun-tick]

Read the `screen` of each member it names, then:

| What you see | Do |
|---|---|
| It waits for the user: a permission, a choice, a folder to trust | An `ask` for it, unless one is open. Do not answer it yourself, with `send` or a key: the question is the user's. |
| It stopped waiting | Its question was most likely answered: make that `ask` `done`, with what happened. |
| Gone while working | A `warn`: it may have crashed. Give `--link project:<its project>` when its terminal closed. |
| Its screen stuck | Look at what it runs. A long command or test that still runs is no news; one that hangs gets a `warn`. |
| Nothing new | Nothing. |

Answer in your own terminal with one short line: the report is in Messages.

## What you may do

| Action | Rule |
|---|---|
| `terminals`, `watch`, `screen`, `message`, `message update`, your loops | On your own |
| `send` a member your decision, an answer or work | On your own, after reading its `screen`, as `jun-tools` warns |
| Answer a member's permission or choice, press a key for it | Never: that is the user's |
| Close a terminal, stop an agent | Only when the user says yes |
