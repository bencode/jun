#!/bin/bash
# Tells the lead in terminal $1 of the agents it leads whose screen has not
# moved since the last run while they work: they may be stuck. A screen is
# told once. Run from a loop, as often as a screen should move.
set -o pipefail
self=$1
state="${TMPDIR:-/tmp}/jun-lead-$self" # a line a member: id, screen, told
touch "$state" || exit 1
terminals=$("$JUN_BIN" tool terminals) || exit 1
lead_idle=$(jq --argjson self "$self" 'any(.[]; .id == $self and .status == "idle")' <<<"$terminals")

# The screen without its last lines and digits, where the spinner, timers
# and status bar change on their own.
screen_sum() {
  "$JUN_BIN" tool screen "$1" --lines 40 |
    awk '{ line[NR] = $0 } END { for (i = 1; i <= NR - 8; i++) print line[i] }' |
    tr -d '0-9' | cksum | cut -d ' ' -f 1
}

next="" stuck=""
while read -r id name; do
  sum=$(screen_sum "$id") || continue
  read -r _ was told < <(grep "^$id " "$state")
  [ "$sum" = "$was" ] || told=0
  if [ "$sum" = "$was" ] && [ "$told" != 1 ] && [ "$lead_idle" = true ]; then
    stuck="$stuck #$id $name," told=1
  fi
  next+="$id $sum $told"$'\n'
done < <(jq -r --argjson self "$self" '.[] | select(.lead == $self and .status == "working") | "\(.id) \(.name // .agent)"' <<<"$terminals")
printf '%s' "$next" >"$state"

[ -z "$stuck" ] || "$JUN_BIN" tool send "$self" "[jun-tick] Screen unchanged while working:${stuck%,}: look, as the jun-lead skill says."
