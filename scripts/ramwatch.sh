#!/usr/bin/env bash
#
# ramwatch.sh — RAM usage profiler for songer.
#
# Samples the RSS of songer + its mpv child while a song plays, then reports
# the peak / minimum / average and compares it to a normal YouTube browser tab.
#
# Usage:
#   scripts/ramwatch.sh "song query" [seconds]
#   scripts/ramwatch.sh                 # defaults: "surf curse freak" 30s
#
set -euo pipefail

cd "$(dirname "$0")/.."

QUERY="${1:-surf curse freak}"
DUR="${2:-30}"
BIN=./songer

if [[ ! -x "$BIN" ]]; then
  echo "▸ building songer..." >&2
  go build -o songer .
fi

rss_kb() {
  local pid="$1"
  if [[ -z "$pid" ]] || [[ ! -r "/proc/$pid/status" ]]; then
    echo 0
    return
  fi
  awk '/^VmRSS:/{print $2}' "/proc/$pid/status"
}

echo "▸ launching songer for ${DUR}s — playing: ${QUERY}"
"$BIN" --source "$QUERY" --limit 1 >/dev/null 2>&1 &
SONGER_PID=$!

# wait for the process tree to come up (search + mpv spawn)
for _ in $(seq 1 40); do
  kill -0 "$SONGER_PID" 2>/dev/null || break
  if pgrep -P "$SONGER_PID" >/dev/null 2>&1; then
    break
  fi
  sleep 0.25
done

mpv_kb=0
peak_total=0
min_total=999999999
sum_total=0
samples=0
time_alive=0

printf "%-6s %-14s %-12s %-10s\n" "sec" "songer(kb)" "mpv(kb)" "total(kb)"

for ((t=1; t<=DUR; t++)); do
  if ! kill -0 "$SONGER_PID" 2>/dev/null; then
    break
  fi
  mpv_pid=$(pgrep -P "$SONGER_PID" 2>/dev/null | head -1 || true)
  s_kb=$(rss_kb "$SONGER_PID")
  m_kb=$(rss_kb "$mpv_pid")
  total=$((s_kb + m_kb))
  printf "%-6d %-14s %-12s %-10s\n" "$t" "$s_kb" "$m_kb" "$total"

  sum_total=$((sum_total + total))
  samples=$((samples + 1))
  if (( total > peak_total )); then peak_total=$total; fi
  if (( total < min_total )); then min_total=$total; fi
  sleep 1
done

# clean up the process tree
pkill -TERM -P "$SONGER_PID" 2>/dev/null || true
kill -TERM "$SONGER_PID" 2>/dev/null || true
sleep 0.3
pkill -KILL -P "$SONGER_PID" 2>/dev/null || true
kill -KILL "$SONGER_PID" 2>/dev/null || true

echo
echo "==============================  RAM PROFILE  =============================="
if (( samples == 0 )); then
  echo "no live samples — songer exited before playing (check query / network)"
  exit 1
fi
avg_total=$((sum_total / samples))
peak_mb=$((peak_total / 1024))
min_mb=$((min_total / 1024))
avg_mb=$((avg_total / 1024))

printf "  samples          : %d over %d s\n" "$samples" "$DUR"
printf "  peak RAM         : %d KB  (~%d MB)\n" "$peak_total" "$peak_mb"
printf "  min RAM          : %d KB  (~%d MB)\n" "$min_total" "$min_mb"
printf "  average RAM      : %d KB  (~%d MB)\n" "$avg_total" "$avg_mb"
echo "-------------------------------------------------------------------------"
echo "  for comparison, a normal YouTube browser tab typically uses:"
echo "    Chrome tab      : ~600 - 1500 MB"
echo "    Firefox tab     : ~400 - 900 MB"
echo "    YouTube mobile  : ~250 - 500 MB"
echo "  songer peak is therefore ~$(awk "BEGIN{printf \"%.0f\", (700*1024)/$peak_total}")x smaller than a Chrome tab."
echo "=========================================================================="
