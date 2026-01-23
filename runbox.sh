#!/usr/bin/env bash
set -euo pipefail

# Pure-bash streaming "output box" (no deps)
#
# - Fixed-width background box (adjust BOX_WIDTH easily)
# - Truncates long lines with "..." (ASCII only)
# - Header/footer are inside the box, no left padding for those lines
# - Command output lines have 2-space left+right padding inside the box
#
# Usage:
#   ./stream_box.sh test
#   ./stream_box.sh -- your_command arg1 arg2
#
# Quick width test:
#   BOX_WIDTH=60 ./stream_box.sh test
#   BOX_WIDTH=120 ./stream_box.sh test

BOX_WIDTH="${BOX_WIDTH:-100}"   # <— adjust here or via env
PAD="${PAD:-2}"

# Subtle gray "overlay" look (best effort)
BG="${BG:-48;5;236}"   # dark gray background
FG="${FG:-38;5;145}"   # light gray text

style=$'\e['"${FG}"';'"${BG}"'m'
reset=$'\e[0m'

# Truncate a string to fit max chars, appending "..." if truncated.
# Byte-based (ASCII-safe).
_trunc_dots() {
  local s="$1"
  local max="$2"

  if (( max <= 0 )); then
    printf ''
    return
  fi

  if (( ${#s} <= max )); then
    printf '%s' "$s"
    return
  fi

  # If we can't even fit "...", just hard cut.
  if (( max <= 3 )); then
    printf '%s' "${s:0:max}"
    return
  fi

  printf '%s...' "${s:0:max-3}"
}

# Print a boxed line that fills exactly BOX_WIDTH columns with background color.
# pad_left/pad_right control inner padding (0 for header/footer).
_print_box_line() {
  local content="$1"
  local pad_left="${2:-$PAD}"
  local pad_right="${3:-$PAD}"

  local usable=$(( BOX_WIDTH - pad_left - pad_right ))
  (( usable < 0 )) && usable=0

  content="$(_trunc_dots "$content" "$usable")"

  printf '%s%*s%-*.*s%*s%s\n' \
    "$style" \
    "$pad_left" "" \
    "$usable" "$usable" "$content" \
    "$pad_right" "" \
    "$reset"
}

run_box_stream() {
  _print_box_line '--- Command Output Start ---' 0 0
  _print_box_line '' 0 0

  # If the "command" is actually a bash function, run it directly.
  # (stdbuf can't execute shell functions)
  if declare -F "${1:-}" >/dev/null 2>&1; then
    "$@" 2>&1 | while IFS= read -r line; do
      _print_box_line "$line" "$PAD" "$PAD"
    done
  else
    # Merge stderr into stdout, stream line-by-line
    if command -v stdbuf >/dev/null 2>&1; then
      stdbuf -oL -eL "$@" 2>&1 | while IFS= read -r line; do
        _print_box_line "$line" "$PAD" "$PAD"
      done
    else
      "$@" 2>&1 | while IFS= read -r line; do
        _print_box_line "$line" "$PAD" "$PAD"
      done
    fi
  fi

  _print_box_line '' 0 0
  _print_box_line '--- Command Output End ---' 0 0
}

########################################
# Test: long-running streaming output
########################################
test_streaming_command() {
  bash -lc '
    for i in {1..120}; do
      ts="$(date +%H:%M:%S)"
      echo "[$ts] step $i: building something; details=$(printf "%0.sabcdef0123456789-" {1..10})"
      if (( i % 13 == 0 )); then
        echo "[$ts] WARN step $i: simulated warning on stderr; payload=$(printf "%0.sWARN-" {1..35})" 1>&2
      fi
      if (( i % 17 == 0 )); then
        echo "[$ts] info: path=/very/long/path/that/keeps/going/and/going/file_$i.log?query=$(printf "%0.sQ" {1..160})"
      fi
      sleep 0.05
    done
  '
}

########################################
# Entrypoint
########################################
case "${1:-}" in
  test)
    shift || true
    run_box_stream test_streaming_command
    ;;
  --)
    shift
    run_box_stream "$@"
    ;;
  "")
    cat <<'EOF'
Usage:
  ./stream_box.sh test
  ./stream_box.sh -- <command> [args...]

Tuning (env vars):
  BOX_WIDTH=100   total box width (background fill), default 100
  PAD=2           left/right padding for output lines, default 2
  BG="48;5;236"   background SGR
  FG="38;5;252"   foreground SGR

Examples:
  BOX_WIDTH=60 ./stream_box.sh test
  BOX_WIDTH=120 PAD=4 ./stream_box.sh test
EOF
    ;;
  *)
    run_box_stream "$@"
    ;;
esac

