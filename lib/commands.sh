#!/usr/bin/env bash
# Command execution helpers for Bellum Installer

# Do not redefine SCRIPT_DIR - it's defined in the main script

# Command execution with logging
run_command() {
    local log_to_console="false"
    local log_to_stderr="false"

    if [ "$#" -eq 0 ]; then
        return 1
    fi

    if [ "$1" = "true" ] || [ "$1" = "false" ] || [ "$1" = "silent" ]; then
        log_to_console="$1"
        shift
    fi

    if [ "$#" -eq 0 ]; then
        return 1
    fi
    
    local rc=0
    if [ "$log_to_console" = "true" ]; then
        log_command "$@"
        cmd_streamer "$@"
        rc=$?
    elif [ "$log_to_console" = "silent" ]; then
        command echo "--- Command Output Start ---" >> "${LOG_FILE}"
        "$@" >> "${LOG_FILE}" 2>&1
        rc=$?
        command echo "--- Command Output End ---" >> "${LOG_FILE}"
        command echo >> "${LOG_FILE}"
    elif [ "$log_to_console" = "false" ]; then
        command echo "--- Command Output Start ---" >> "${LOG_FILE}"
        log_command "$@"
        "$@" >> "${LOG_FILE}" 2>&1
        rc=$?
        command echo "--- Command Output End ---" >> "${LOG_FILE}"
        command echo >> "${LOG_FILE}"
    else
        log error "Invalid run_command arg: $log_to_console"
        rc=1
    fi

    if [ $rc -ne 0 ]; then
        log error "Command failed (${rc}): $*"
    fi

    if [ "$log_to_console" = "true" ]; then
        echo
    fi

    return $rc
}

# Live command output streaming
_cmd_streamer_trunc() {
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

    if (( max <= 3 )); then
        printf '%s' "${s:0:max}"
        return
    fi

    printf '%s...' "${s:0:max-3}"
}

_cmd_streamer_print_line() {
    # Extract streamer settings from environment or use defaults
    local width="${CMD_STREAMER_WIDTH:-120}"
    local pad="${CMD_STREAMER_PAD:-2}"
    local bg="${CMD_STREAMER_BG:-48;5;236}"
    local fg="${CMD_STREAMER_FG:-38;5;145}"
    local style reset
    
    style=$'\e['"${fg}"';'"${bg}"'m'
    reset=$'\e[0m'
    
    local content="$1"
    local pad_left="${2:-$pad}"
    local pad_right="${3:-$pad}"
    local usable=$(( width - pad_left - pad_right ))
    (( usable < 0 )) && usable=0

    content="${content#"${content%%[!$' \t\r\n']*}"}"
    content="${content%"${content##*[!$' \t\r\n']}"}"

    content="$(_cmd_streamer_trunc "$content" "$usable")"

    printf '%s%*s%-*.*s%*s%s\n' \
        "$style" \
        "$pad_left" "" \
        "$usable" "$usable" "$content" \
        "$pad_right" "" \
        "$reset"
}

_cmd_streamer_run() {
    local rc=0

    if declare -F "${1:-}" >/dev/null 2>&1; then
        {
            while IFS= read -r line; do
                _cmd_streamer_print_line "$line" "$pad" "$pad"
            done
        } < <("$@" 2>&1)
        rc=$?
    else
        if command -v stdbuf >/dev/null 2>&1; then
            {
                while IFS= read -r line; do
                    _cmd_streamer_print_line "$line" "$pad" "$pad"
                done
            } < <(stdbuf -oL -eL "$@" 2>&1)
            rc=$?
        else
            {
                while IFS= read -r line; do
                    _cmd_streamer_print_line "$line" "$pad" "$pad"
                done
            } < <("$@" 2>&1)
            rc=$?
        fi
    fi

    return "$rc"
}

cmd_streamer() {
    local width="${CMD_STREAMER_WIDTH:-120}"
    local pad="${CMD_STREAMER_PAD:-2}"
    local bg="${CMD_STREAMER_BG:-48;5;236}"
    local fg="${CMD_STREAMER_FG:-38;5;145}"
    local header="${CMD_STREAMER_HEADER:--- Command Output Start ---}"
    local footer="${CMD_STREAMER_FOOTER:--- Command Output End ---}"
    local style reset

    style=$'\e['"${fg}"';'"${bg}"'m'
    reset=$'\e[0m'

    _cmd_streamer_print_line "$header" 0 0
    _cmd_streamer_print_line "" 0 0

    _cmd_streamer_run "$@"
    local rc=$?

    _cmd_streamer_print_line "" 0 0
    _cmd_streamer_print_line "$footer" 0 0

    return "$rc"
}
