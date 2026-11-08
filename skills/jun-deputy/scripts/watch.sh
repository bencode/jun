#!/bin/bash
# Passes what needs a look on to the deputy in terminal $1: agents that wait
# for the user, stopped waiting, finished or went while working, and those
# that start working with an ask still open, most likely answered. Each goes
# once the deputy is idle.
# Run from a loop, which starts it again when the watch ends.
self=$1
events='select(.id != $self and (.to == "waiting" or .from == "waiting"
    or (.from == "working" and (.to == "idle" or .to == "")) or (.from == "idle" and .to == "working")))
  | "\(.id) \(.project) \(.agent) \(if .to == "" then "gone" elif .from == "idle" and .to == "working" then "working, ask open" else .to end)"'

deputy_status() {
  "$JUN_BIN" tool terminals | jq -r --argjson self "$self" '.[] | select(.id == $self) | .status // empty'
}

# open_ask tells whether a message about the agent in terminal $1 still asks.
open_ask() {
  "$JUN_BIN" tool message | jq -e --arg t "terminal:$1" --arg d "diff:$1" \
    'any(.[]; .kind == "ask" and (.link == $t or .link == $d))' >/dev/null
}

"$JUN_BIN" tool watch | jq --unbuffered -r --argjson self "$self" "$events" | while IFS= read -r event; do
  if [[ $event == *", ask open" ]] && ! open_ask "${event%% *}"; then
    continue # the user gave it new work
  fi
  status=$(deputy_status)
  while [ "$status" != idle ]; do
    [ -n "$status" ] || exit 1 # the deputy is gone
    sleep 2
    status=$(deputy_status)
  done
  "$JUN_BIN" tool send "$self" "[jun-tick] $event: look, as the jun-deputy skill says."
done
