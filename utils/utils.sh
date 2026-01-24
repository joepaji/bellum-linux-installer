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
        local display_line="$line"
        if [ "${#display_line}" -ge 125 ]; then
            display_line="${display_line:0:122}..."
        fi
        if [ "$first_line" = true ]; then
            echo "${prefix_base}  ${message_color}${display_line}${COLOR_RESET}"
            command echo -e "${prefix_base}  ${message_color}${line}${COLOR_RESET}" >> "${LOG_FILE}"
            first_line=false
        else
            echo "${indent}  ${message_color}${display_line}${COLOR_RESET}"
            command echo -e "${indent}  ${message_color}${line}${COLOR_RESET}" >> "${LOG_FILE}"
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

    local rc=0
    if [ "$log_to_console" = "true" ]; then
        cmd_streamer "$@"
        rc=$?
    else
        command echo "--- Command Output Start ---" >> "${LOG_FILE}"
        "$@" >> "${LOG_FILE}" 2>&1
        rc=$?
        command echo "--- Command Output End ---" >> "${LOG_FILE}"
        command echo >> "${LOG_FILE}"
    fi
    if [ $rc -ne 0 ]; then
        log error "Command failed (${rc}): $*"
    fi

    echo

    return $rc
}

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
    local content="$1"
    local pad_left="${2:-$pad}"
    local pad_right="${3:-$pad}"
    local usable=$(( width - pad_left - pad_right ))
    (( usable < 0 )) && usable=0

    # Strip leading/trailing invisible whitespace only (spaces/tabs/newlines)
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
        "$@" 2>&1 | while IFS= read -r line; do
            _cmd_streamer_print_line "$line" "$pad" "$pad"
        done
        rc=${PIPESTATUS[0]}
    else
        if command -v stdbuf >/dev/null 2>&1; then
            stdbuf -oL -eL "$@" 2>&1 | while IFS= read -r line; do
                _cmd_streamer_print_line "$line" "$pad" "$pad"
            done
            rc=${PIPESTATUS[0]}
        else
            "$@" 2>&1 | while IFS= read -r line; do
                _cmd_streamer_print_line "$line" "$pad" "$pad"
            done
            rc=${PIPESTATUS[0]}
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

get_proton_ge_url() {
    if [ -z "${PROTON_GE_BASE_URL:-}" ] || [ -z "${PROTON_VER:-}" ]; then
        return 1
    fi
    echo "${PROTON_GE_BASE_URL}/${PROTON_VER}/${PROTON_VER}.tar.gz"
}

patch_proton_user_settings() {
    local settings_file="$1"
    if [ -z "$settings_file" ] || [ ! -f "$settings_file" ]; then
        log error "Proton user settings file not found: $settings_file"
        return 1
    fi

    local tmp_file
    tmp_file="$(mktemp)"
    if [ -z "$tmp_file" ]; then
        log error "Failed to create temp file for Proton settings patch"
        return 1
    fi

    awk -v tmp_file="$tmp_file" '
        BEGIN {
            desired["PROTON_NO_ESYNC"]="1"
            desired["PROTON_NO_FSYNC"]="1"
            desired["PROTON_USE_NTSYNC"]="1"
            desired["PROTON_FSR3_UPGRADE"]="1"
            desired["PROTON_FSR4_RDNA3_UPGRADE"]="1"
            desired["PROTON_FSR4_UPGRADE"]="4.0.1"
        }
        {
            line=$0
            if (line ~ /^[[:space:]]*#/) {
                print line >> tmp_file
                next
            }
            matched=0
            for (key in desired) {
                pattern="^[[:space:]]*\"" key "\"[[:space:]]*:"
                if (line ~ pattern) {
                    indent=match(line, /^[[:space:]]*/)
                    pad=substr(line, 1, RLENGTH)
                    print pad "\"" key "\": \"" desired[key] "\"," >> tmp_file
                    found[key]=1
                    matched=1
                    break
                }
            }
            if (matched == 1) {
                next
            }
            if (line ~ /^[[:space:]]*}[[:space:]]*$/) {
                for (key in desired) {
                    if (!(key in found)) {
                        print "    \"" key "\": \"" desired[key] "\"," >> tmp_file
                    }
                }
            }
            print line >> tmp_file
        }
    ' "$settings_file"

    if [ ! -s "$tmp_file" ]; then
        log error "Failed to write patched Proton settings"
        rm -f "$tmp_file"
        return 1
    fi

    mv "$tmp_file" "$settings_file"
    return 0
}

ensure_proton_ge() {
    local proton_dir="${WORKDIR}/${PROTON_VER}"
    local settings_sample="${proton_dir}/user_settings.sample.py"
    local settings_file="${proton_dir}/user_settings.py"

    if [ ! -d "$proton_dir" ]; then
        local proton_url
        proton_url="$(get_proton_ge_url)"
        if [ -z "$proton_url" ]; then
            log error "Unable to determine Proton GE download URL"
            return 1
        fi

        local tmp_dir
        tmp_dir="$(mktemp -d -t "proton-ge.XXXXXX")"
        if [ -z "$tmp_dir" ] || [ ! -d "$tmp_dir" ]; then
            log error "Failed to create temp directory for Proton GE download"
            return 1
        fi

        local archive="${tmp_dir}/${PROTON_VER}.tar.gz"
        log info "Downloading Proton GE ${PROTON_VER}..."
        if ! wget -O "$archive" "$proton_url" >> "${LOG_FILE}" 2>&1; then
            rm -rf "$tmp_dir"
            return 1
        fi

        log info "Extracting Proton GE to ${WORKDIR}..."
        if ! tar -xzf "$archive" -C "$WORKDIR" >> "${LOG_FILE}" 2>&1; then
            rm -rf "$tmp_dir"
            return 1
        fi

        rm -rf "$tmp_dir"
    fi

    if [ -f "$settings_sample" ] && [ ! -f "$settings_file" ]; then
        mv "$settings_sample" "$settings_file"
    fi

    if [ -f "$settings_file" ]; then
        patch_proton_user_settings "$settings_file"
    else
        log error "Proton user settings file missing after setup: $settings_file"
        return 1
    fi

    return 0
}

download_launcher_installer() {
    local download_dir="${WORKDIR}/installer-cache"
    local filename="AstarteLauncher-amd64-installer.exe"
    local dest="${download_dir}/${filename}"

    mkdir -p "$download_dir"
    log info "Downloading launcher installer to: ${dest}"
    if ! wget -O "$dest" "https://auto-updater.astarte.industries/astartelauncher/windows-amd64/AstarteLauncher-amd64-installer.exe" >> "${LOG_FILE}" 2>&1; then
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

cleanup_packages_tmp_root() {
    local archive_path="$1"
    local tmp_root
    tmp_root="$(get_packages_tmp_root "$archive_path")"
    if [ -n "$tmp_root" ] && [ -d "$tmp_root" ]; then
        rmdir "$tmp_root" 2>/dev/null || true
    fi
}
