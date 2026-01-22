#!/usr/bin/bash

# Load shared versions
UTILS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
if ! source "${UTILS_DIR}/versions.env"; then
    echo "ERROR: Failed to load versions.env"
    return 1
fi

# Color codes for output
COLOR_BOLD='\033[1m'
COLOR_RED='\033[31m'
COLOR_BOLD_RED='\033[1;31m'
COLOR_YELLOW='\033[33m'
COLOR_BOLD_YELLOW='\033[1;33m'
COLOR_GREEN='\033[32m'
COLOR_BOLD_GREEN='\033[1;32m'
COLOR_BLUE='\033[34m'
COLOR_BOLD_BLUE='\033[1;34m'
COLOR_CYAN='\033[36m'
COLOR_BOLD_CYAN='\033[1;36m'
COLOR_GRAY_BOLD='\033[90m'
COLOR_CMD_DIM='\033[2m'
COLOR_CMD_OUT='\033[90m'
COLOR_CMD_BG='\033[48;5;236m'
COLOR_RESET='\033[0m'

# Required Wine binaries (absolute paths)
REQUIRED_WINE_BINARIES=("/usr/bin/msidb" "/usr/bin/wine" "/usr/bin/wineboot")

# Echo wrapper function that highlights WINEPREFIX with CYAN BOLD color
echo() {
    local flags=""
    local message=""
    
    # Extract echo flags (like -n, -e, -E)
    while [[ "$1" == -* ]]; do
        flags="$flags $1"
        shift
    done
    
    # Remaining arguments are the message
    message="$*"
    
    # Replace WINEPREFIX with colored version
    message="${message//WINEPREFIX/${COLOR_BOLD_CYAN}WINEPREFIX${COLOR_RESET}}"
    
    # Call the real echo with flags and message
    command echo $flags -e "$message"
}

init_log_file() {
    # Create logs directory if it doesn't exist
    mkdir -p "${LOG_DIR}"

    # Log rotation: backup current log and remove old backups
    if [ -f "${LOG_FILE}" ]; then
        mv "${LOG_FILE}" "${LOG_FILE}.1"
        rm -f "${LOG_FILE}.2" "${LOG_FILE}."[0-9]*
    fi
}

confirm_proceed() {
    local prompt_message="${1:-Do you want to proceed with the installation? (Y/n): }"
    local proceed_confirmed=false

    while [ "$proceed_confirmed" = false ]; do
        read -p "$prompt_message" PROCEED_INPUT
        
        # Default to yes if empty
        if [ -z "$PROCEED_INPUT" ]; then
            proceed_confirmed=true
        elif [[ "$PROCEED_INPUT" =~ ^[Yy]$ ]]; then
            proceed_confirmed=true
        elif [[ "$PROCEED_INPUT" =~ ^[Nn]$ ]]; then
            log warn "Installation cancelled by user"
            return 1
        else
            echo "${COLOR_BOLD_RED}Invalid input: '$PROCEED_INPUT'${COLOR_RESET}"
            echo "Please enter 'y' or 'Y' to proceed, 'n' or 'N' to cancel, or press Enter to proceed (default: yes)"
        fi
    done

    return 0
}

print_installer_summary() {
    echo "$(cat <<- EOM 
${COLOR_BOLD_CYAN}===================================================
#    Wine-Proton Orchestator Installer Summary    #
===================================================${COLOR_RESET}
EOM
)"

    local launcher_summary
    if [ -n "${LAUNCHER_INSTALLER_PATH}" ]; then
        launcher_summary="${LAUNCHER_INSTALLER_PATH}"
    else
        launcher_summary="(will be downloaded)"
    fi

    echo "$(cat <<- EOM
${COLOR_BOLD_CYAN}   Proton Version${COLOR_RESET}:   ${COLOR_BOLD}${PROTON_VER}${COLOR_RESET}
${COLOR_BOLD_CYAN}     Wine Version${COLOR_RESET}:   ${COLOR_BOLD}${WINE_VER} (Stable)${COLOR_RESET}
${COLOR_BOLD_CYAN}   Winetricks Ver${COLOR_RESET}:   ${COLOR_BOLD}${WINETRICKS_VER}${COLOR_RESET}
${COLOR_BOLD_CYAN}        VKD3D Ver${COLOR_RESET}:   ${COLOR_BOLD}${VKD3D_VER}${COLOR_RESET}
${COLOR_BOLD_CYAN}         DXVK Ver${COLOR_RESET}:   ${COLOR_BOLD}${DXVK_VER}${COLOR_RESET}

${COLOR_BOLD_CYAN}       WINEPREFIX${COLOR_RESET}:   ${COLOR_BOLD}${WINEPREFIX}${COLOR_RESET}
${COLOR_BOLD_CYAN} Bellum Installer${COLOR_RESET}:   ${COLOR_BOLD}${launcher_summary}${COLOR_RESET}

${COLOR_BOLD_YELLOW}NOTE:${COLOR_RESET} The game will be installed into the specified WINEPREFIX path.
EOM
)"
}

log() {
    local level="$1"
    local message="$2"
    
    # Validate log level
    case "$level" in
        error|info|warn|cmd)
            ;;
        *)
            # If no valid level provided, treat first arg as message with default level
            message="$level"
            level="info"
            ;;
    esac
    
    # Format log level with right alignment, spacing, and colors
    # Max level width is 7 (for [ERROR])
    local level_str=""
    local message_color="${COLOR_RESET}"
    case "$level" in
        error)
            level_str="${COLOR_BOLD_RED}[ERROR]${COLOR_RESET}"
            ;;
        warn)
            level_str="${COLOR_BOLD_YELLOW} [WARN]${COLOR_RESET}"
            message_color="${COLOR_YELLOW}"
            ;;
        info)
            level_str="${COLOR_BOLD_GREEN} [INFO]${COLOR_RESET}"
            ;;
        cmd)
            level_str="${COLOR_BOLD_BLUE}  [CMD]${COLOR_RESET}"
            message_color="${COLOR_BLUE}"
            ;;
    esac
    
    # Calculate indent for multi-line messages (without trailing spaces)
    # Note: indent_len accounts for the raw length minus color codes
    local prefix_base="${COLOR_GRAY_BOLD}[Bellum-Linux-Installer]${COLOR_RESET}:${level_str}"
    local indent_base_raw="[Bellum-Linux-Installer]:[ERROR]"  # Use ERROR as reference for consistent length
    local indent_len=${#indent_base_raw}
    local indent=$(printf "%${indent_len}s")
    
    # Highlight WINEPREFIX in the message
    message="${message//WINEPREFIX/${COLOR_BOLD_CYAN}WINEPREFIX${COLOR_RESET}${message_color}}"
    
    # Convert escape sequences and process multi-line message
    local first_line=true
    while IFS= read -r line; do
        while [ "${#line}" -gt 150 ]; do
            local chunk="${line:0:150}"
            line="${line:150}"
            if [ "$first_line" = true ]; then
                echo "${prefix_base}  ${message_color}${chunk}${COLOR_RESET}" | tee -a "${LOG_FILE}"
                first_line=false
            else
                echo "${indent}  ${message_color}${chunk}${COLOR_RESET}" | tee -a "${LOG_FILE}"
            fi
        done
        if [ "$first_line" = true ]; then
            echo "${prefix_base}  ${message_color}${line}${COLOR_RESET}" | tee -a "${LOG_FILE}"
            first_line=false
        else
            echo "${indent}  ${message_color}${line}${COLOR_RESET}" | tee -a "${LOG_FILE}"
        fi
    done <<< "$(printf "%b" "$message ")"
}

log_command() {
    if [ "$#" -eq 0 ]; then
        return 1
    fi

    local joined=""
    local arg
    for arg in "$@"; do
        if [ -n "$joined" ]; then
            joined+=" "
        fi
        joined+="$arg"
    done

    log cmd "${joined}"
}

get_cmd_output_width() {
    local width="${CMD_OUTPUT_MAX_WIDTH:-150}"

    if [ -n "${COLUMNS:-}" ]; then
        if [ "${COLUMNS}" -lt "${width}" ]; then
            width="${COLUMNS}"
        fi
    else
        local term_cols
        term_cols="$(tput cols 2>/dev/null || echo "")"
        if [ -n "$term_cols" ] && [ "$term_cols" -lt "$width" ]; then
            width="$term_cols"
        fi
    fi

    if [ -z "$width" ] || [ "$width" -le 0 ]; then
        width=150
    fi

    echo "$width"
}

strip_ansi() {
    # Remove ANSI escape sequences for accurate width calculations.
    printf "%s" "$1" | sed -E 's/\x1B\[[0-9;]*[[:alpha:]]//g'
}

expand_tabs() {
    local input="$1"
    local output=""
    local col=0
    local ch=""

    while IFS= read -r -n1 ch; do
        if [ "$ch" = $'\t' ]; then
            local spaces=$((8 - (col % 8)))
            output+=$(printf "%${spaces}s" "")
            col=$((col + spaces))
        else
            output+="$ch"
            col=$((col + 1))
        fi
    done < <(printf "%s" "$input")

    printf "%s" "$output"
}

print_cmd_output_line() {
    local line="$1"
    local indent="${2:-0}"
    local width
    local visible
    local padding
    local stripped
    local clean_line
    local expanded_line
    local max_len=150
    local indent_str=""
    local display_line

    width="$(get_cmd_output_width)"
    clean_line="${line//$'\r'/}"
    expanded_line="$(expand_tabs "$clean_line")"
    stripped="$(strip_ansi "$expanded_line")"
    visible=${#stripped}

    if [ "$visible" -gt "$max_len" ]; then
        expanded_line="${expanded_line:0:$((max_len - 3))}..."
        stripped="$(strip_ansi "$expanded_line")"
        visible=${#stripped}
    fi

    if [ "$indent" -gt 0 ]; then
        indent_str="$(printf "%${indent}s" "")"
        display_line="${indent_str}${expanded_line}"
    else
        display_line="${expanded_line}"
    fi

    stripped="$(strip_ansi "$display_line")"
    visible=${#stripped}

    if [ "$visible" -gt "$width" ]; then
        display_line="${display_line:0:$width}"
        visible=$width
    fi

    padding=$((width - visible))
    printf "%b%s%*s%b\n" "${COLOR_CMD_OUT}${COLOR_CMD_BG}" "$display_line" "$padding" "" "${COLOR_RESET}"
}

colorize_cmd_output() {
    local line
    while IFS= read -r line || [ -n "$line" ]; do
        print_cmd_output_line "$line" 2
    done
}

run_command() {
    local log_to_console="false"

    if [ "$#" -eq 0 ]; then
        return 1
    fi

    if [ "$1" = "true" ] || [ "$1" = "false" ]; then
        log_to_console="$1"
        shift
    fi

    if [ "$#" -eq 0 ]; then
        return 1
    fi

    if [ "$log_to_console" = "true" ]; then
        log_command "$@"
    else
        log_command "$@" >/dev/null
    fi
    print_cmd_output_line "--- Command Output Start ---" 0
    "$@" 2> >(colorize_cmd_output) | colorize_cmd_output
    local rc=${PIPESTATUS[0]}
    print_cmd_output_line "--- Command Output End ---" 0
    if [ $rc -ne 0 ]; then
        log error "Command failed (${rc}): $*"
    fi

    echo

    return $rc
}

download_launcher_installer() {
    local download_dir="${WORKDIR}/installer-cache"
    local filename="AstarteLauncher-amd64-installer.exe"
    local dest="${download_dir}/${filename}"

    mkdir -p "$download_dir"
    log info "Downloading launcher installer to: ${dest}"
    if ! run_command true wget -O "$dest" "https://auto-updater.astarte.industries/astartelauncher/windows-amd64/AstarteLauncher-amd64-installer.exe"; then
        return 1
    fi

    LAUNCHER_INSTALLER_PATH="$dest"
    LAUNCHER_INSTALLER_DOWNLOADED="true"
    LAUNCHER_INSTALLER_DIR="$download_dir"
}

cleanup_launcher_installer() {
    if [ "${LAUNCHER_INSTALLER_DOWNLOADED}" = "true" ] && [ -n "$LAUNCHER_INSTALLER_PATH" ]; then
        log info "Cleaning up downloaded launcher installer..."
        rm -f "$LAUNCHER_INSTALLER_PATH"
        if [ -n "$LAUNCHER_INSTALLER_DIR" ]; then
            rmdir "$LAUNCHER_INSTALLER_DIR" 2>/dev/null || true
        fi
    fi
}

require_installer_env() {
    if [ -z "$WORKDIR" ] || [ -z "$LOG_DIR" ] || [ -z "$LOG_FILE" ]; then
        echo "ERROR: Required variables not set. Make sure this script is sourced from installer.sh"
        return 1
    fi
}

get_packages_tmp_root() {
    local archive_path="$1"
    local packages_dir
    packages_dir="$(cd "$(dirname "$archive_path")" && pwd)"
    echo "${packages_dir}/.tmp"
}

extract_package_to_tmp() {
    local archive_path="$1"
    local dest_name="$2"
    local tmp_root
    local dest_dir

    tmp_root="$(get_packages_tmp_root "$archive_path")"
    mkdir -p "$tmp_root"
    dest_dir="$(mktemp -d -p "$tmp_root" "${dest_name}.XXXXXX")"
    if [ $? -ne 0 ] || [ -z "$dest_dir" ]; then
        return 1
    fi
    tar -xzf "$archive_path" -C "$dest_dir"
    if [ $? -ne 0 ]; then
        rm -rf "$dest_dir"
        return 1
    fi

    echo "$dest_dir"
}
