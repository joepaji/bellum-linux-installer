#!/usr/bin/env bash
# Core utilities for Bellum Installer
# Includes logging, colors, and echo wrapper

# Do not redefine SCRIPT_DIR - it's defined in the main script

# Color codes
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

# Echo wrapper with WINEPREFIX highlighting
echo() {
    local flags=""
    local message=""
    
    while [[ "$1" == -* ]]; do
        flags="$flags $1"
        shift
    done
    
    message="$*"
    message="${message//WINEPREFIX/${COLOR_BOLD_CYAN}WINEPREFIX${COLOR_RESET}}"
    command echo $flags -e "$message"
}

init_log_file() {
    mkdir -p "${LOG_DIR}"
    
    if [ -f "${LOG_FILE}" ]; then
        mv "${LOG_FILE}" "${LOG_FILE}.1"
        rm -f "${LOG_FILE}.2" "${LOG_FILE}".[0-9]*
    fi
}

confirm_proceed() {
    local prompt_message="${1:-Do you want to proceed with the installation? (Y/n): }"
    local proceed_confirmed=false

    while [ "$proceed_confirmed" = false ]; do
        read -p "$prompt_message" PROCEED_INPUT
        
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

${COLOR_BOLD_CYAN}         GPU TYPE${COLOR_RESET}:   ${COLOR_BOLD}${GPU_TYPE}${COLOR_RESET}

${COLOR_BOLD_YELLOW}NOTE:${COLOR_RESET} The game will be installed into the specified WINEPREFIX path.
EOM
)"
}

log() {
    local level="$1"
    local message="$2"
    
    # Handle empty messages
    if [ -z "$message" ]; then
        return
    fi
    
    case "$level" in
        error|info|warn|cmd)
            ;;
        *)
            message="$level"
            level="info"
            ;;
    esac
    
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
    
    local prefix_base="${COLOR_GRAY_BOLD}[Bellum-Linux-Installer]${COLOR_RESET}:${level_str}"
    local indent_base_raw="[Bellum-Linux-Installer]:[ERROR]"
    local indent_len=${#indent_base_raw}
    local indent=$(printf "%${indent_len}s")
    
    message="${message//WINEPREFIX/${COLOR_BOLD_CYAN}WINEPREFIX${COLOR_RESET}${message_color}}"
    
    local first_line=true
    while IFS= read -r display_line; do
        if [ "${#display_line}" -ge 125 ]; then
            display_line="${display_line:0:122}..."
        fi
        if [ "$first_line" = true ]; then
            echo "${prefix_base}  ${message_color}${display_line}${COLOR_RESET}"
            command echo -e "${prefix_base}  ${message_color}${display_line}${COLOR_RESET}" >> "${LOG_FILE}"
            first_line=false
        else
            echo "${indent}  ${message_color}${display_line}${COLOR_RESET}"
            command echo -e "${indent}  ${message_color}${display_line}${COLOR_RESET}" >> "${LOG_FILE}"
        fi
    done <<< "$(printf "%b" "$message")"
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

require_installer_env() {
    if [ -z "$WORKDIR" ] || [ -z "$LOG_DIR" ] || [ -z "$LOG_FILE" ]; then
        echo "ERROR: Required variables not set. Make sure this script is sourced from installer.sh"
        return 1
    fi
}

# Detect GPU type using glxinfo
# Returns the GPU type directly (e.g., "NVIDIA", "AMD", "Intel")
# Returns empty string and exits with status 1 if detection fails
detect_gpu_type() {
    local gpu_output
    gpu_output=$(glxinfo 2>/dev/null | grep -i "OpenGL renderer" | awk -F': ' '{print $2}' | awk '{print $1}')

    if [ -z "$gpu_output" ]; then
        log error "Failed to detect GPU type"
        return 1
    fi

    echo "$gpu_output"
    # echo "NVIDIA"
    return 0
}
