#!/bin/bash
# Passes what needs a look on to the deputy in terminal $1: agents that wait
# for the user, stopped waiting, finished or went while working, each once the
# deputy is idle.
# Run from a loop, which starts it again when the watch ends.
self=$1
events='select(.id != $self and (.to == "waiting" or .from == "waiting" or (.from == "working" and (.to == "idle" or .to == ""))))
  | "\(.id) \(.project) \(.agent) \(if .to == "" then "gone" else .to end)"'

deputy_status() {
  "$JUN_BIN" tool terminals | jq -r --argjson self "$self" '.[] | select(.id == $self) | .status // empty'
}

"$JUN_BIN" tool watch | jq --unbuffered -r --argjson self "$self" "$events" | while IFS= read -r event; do
  status=$(deputy_status)
  while [ "$status" != idle ]; do
    [ -n "$status" ] || exit 1 # the deputy is gone
    sleep 2
    status=$(deputy_status)
  done
  "$JUN_BIN" tool send "$self" "[jun-tick] $event: look, as the jun-deputy skill says."
done
