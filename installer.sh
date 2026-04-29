#!/usr/bin/env bash

WORKDIR=$(pwd)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROTONPATH=${WORKDIR}/${PROTON_VER}
LOG_DIR="${WORKDIR}/logs"
LOG_FILE="${LOG_DIR}/installer.log"

FORCE_WINE_VERSION=false
WINEPREFIX_ARG=""
LAUNCHER_INSTALLER_PATH=""

print_usage() {
    echo "Usage: $0 [--force-wine-version] [--wineprefix PATH] [--launcher-installer PATH]"
    echo "Positional args: [WINEPREFIX] [LAUNCHER_INSTALLER_PATH]"
    echo
    echo "Notes:"
    echo "  - If provided, --wineprefix/positional WINEPREFIX overrides the WINEPREFIX env var."
    echo "  - If launcher installer path is omitted, it will be downloaded (requires wget)."
    echo "  - WINEPREFIX can also be set via the WINEPREFIX environment variable."
}

while [[ $# -gt 0 ]]; do
    case "$1" in
        --force-wine-version)
            FORCE_WINE_VERSION=true
            shift
            ;;
        --wineprefix)
            WINEPREFIX_ARG="$2"
            shift 2
            ;;
        --launcher-installer)
            LAUNCHER_INSTALLER_PATH="$2"
            shift 2
            ;;
        -h|--help)
            print_usage
            exit 0
            ;;
        *)
            if [ -z "$WINEPREFIX_ARG" ]; then
                WINEPREFIX_ARG="$1"
            elif [ -z "$LAUNCHER_INSTALLER_PATH" ]; then
                LAUNCHER_INSTALLER_PATH="$1"
            else
                log error "Unexpected argument: $1"
                exit 1
            fi
            shift
            ;;
    esac
done

# Source configuration (needed for env vars)
source "${SCRIPT_DIR}/config/versions.env"

# Source core utilities (defines init_log_file and other functions)
source "${SCRIPT_DIR}/lib/core.sh"

# Initialize logging
init_log_file

# Source command helpers
source "${SCRIPT_DIR}/lib/commands.sh"

# Source package helpers
source "${SCRIPT_DIR}/lib/packages.sh"

# Source launcher generator
source "${SCRIPT_DIR}/lib/launcher.sh"

echo "${COLOR_BOLD_BLUE}
==============================================================
|        Wine-Proton Orchestrator Installer for Bellum       |
==============================================================${COLOR_RESET}"

# Source workflow
source "${SCRIPT_DIR}/workflow/precheck.sh"
if [ $? -ne 0 ]; then
    exit 1
fi

# Run installation workflow
source "${SCRIPT_DIR}/workflow/install.sh"
run_installer

# Run configuration workflow
source "${SCRIPT_DIR}/workflow/configure.sh"
run_configuration || exit 1
