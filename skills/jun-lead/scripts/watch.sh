#!/bin/bash
# Tells the lead in terminal $1 of the agents it leads that start waiting
# for the user, stop waiting, or go while working, each once the lead is
# idle. Run from a loop, which starts it again when the watch ends.
self=$1
events='select(.lead == $self and (.to == "waiting" or .from == "waiting" or (.from == "working" and .to == "")))
  | "#\(.id) \(.name // .agent) \(if .to == "" then "gone while working" elif .to == "waiting" then "waiting" else "stopped waiting" end)"'

lead_status() {
  "$JUN_BIN" tool terminals | jq -r --argjson self "$self" '.[] | select(.id == $self) | .status // empty'
}

"$JUN_BIN" tool watch | jq --unbuffered -r --argjson self "$self" "$events" | while IFS= read -r event; do
  status=$(lead_status)
  while [ "$status" != idle ]; do
    [ -n "$status" ] || exit 1 # the lead is gone
    sleep 2
    status=$(lead_status)
  done
  "$JUN_BIN" tool send "$self" "[jun-tick] $event: look, as the jun-lead skill says."
done
