#!/usr/bin/bash

WORKDIR=$(pwd)

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
if ! source "${SCRIPT_DIR}/utils.sh"; then
    command echo "ERROR: Failed to load utils.sh"
    exit 1
fi

PROTONPATH=${WORKDIR}/${PROTON_VER}
LOG_DIR="${WORKDIR}/logs"
LOG_FILE="${LOG_DIR}/installer.log"


# Parse script arguments
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

# Initialize/Rotate log file
init_log_file

# CLI Header
echo "${COLOR_BOLD_BLUE}
===============================================================
|        Wine-Proton Orchestator Installer for Bellum         |
===============================================================
${COLOR_RESET}"

#======================================#
#  Source and execute precheck script  #
#======================================#
source "$(dirname "$0")/precheck.sh"
if [ $? -ne 0 ]; then
    exit 1
fi

#======================================#
#            Installer logic           #
#======================================#
install_vkd3d() {
    local archive="packages/vkd3d-${VKD3D_VER}.tar.gz"
    local tmp_dir
    local install_dir

    log info "Installing VKD3D..."
    if [ ! -f "$archive" ]; then
        log error "VKD3D archive not found: $archive"
        return 1
    fi

    tmp_dir="$(extract_package_to_tmp "$archive" "vkd3d")"
    if [ $? -ne 0 ] || [ -z "$tmp_dir" ]; then
        log error "Failed to extract VKD3D archive"
        return 1
    fi

    install_dir="$tmp_dir"
    if [ ! -f "$install_dir/setup_vkd3d_proton.sh" ]; then
        install_dir="$(find "$tmp_dir" -mindepth 1 -maxdepth 1 -type d | head -n 1)"
    fi
    if [ -z "$install_dir" ] || [ ! -f "$install_dir/setup_vkd3d_proton.sh" ]; then
        log error "VKD3D setup script not found after extraction."
        rm -rf "$tmp_dir"
        return 1
    fi

    run_command true "$install_dir/setup_vkd3d_proton.sh" install
    local rc=$?
    if [ $rc -ne 0 ]; then
        log error "VKD3D installation failed."
        rm -rf "$tmp_dir"
        return $rc
    fi

    rm -rf "$tmp_dir"
    log info "✓ VKD3D installed"
}

install_dxvk() {
    local archive="packages/dxvk-${DXVK_VER}.tar.gz"
    local tmp_dir
    local install_dir

    log info "Installing DXVK..."
    if [ ! -f "$archive" ]; then
        log error "DXVK archive not found: $archive"
        return 1
    fi

    tmp_dir="$(extract_package_to_tmp "$archive" "dxvk")"
    if [ $? -ne 0 ] || [ -z "$tmp_dir" ]; then
        log error "Failed to extract DXVK archive"
        return 1
    fi

    install_dir="$tmp_dir"
    if [ ! -f "$install_dir/dxvk_setup.sh" ]; then
        install_dir="$(find "$tmp_dir" -mindepth 1 -maxdepth 1 -type d | head -n 1)"
    fi
    if [ -z "$install_dir" ] || [ ! -f "$install_dir/dxvk_setup.sh" ]; then
        log error "DXVK setup script not found after extraction."
        rm -rf "$tmp_dir"
        return 1
    fi

    run_command true "$install_dir/dxvk_setup.sh" install
    local rc=$?
    if [ $rc -ne 0 ]; then
        log error "DXVK installation failed."
        rm -rf "$tmp_dir"
        return $rc
    fi

    rm -rf "$tmp_dir"
    log info "✓ DXVK installed"
}

run_installer() {
    echo
    log info "Starting Installation"
    echo

    export env PROTONPATH="${PROTONPATH}"
    export env WINEPREFIX="${WINEPREFIX_ARG:-$WINEPREFIX}"

    local msidb="${MSIDB_BIN}"
    local wine="${WINE_BIN}"
    local wineboot="${WINEBOOT_BIN}"
    local umu_run="${UMU_RUN_BIN}"
    local winetricks="${WINETRICKS_BIN}"
    local launcher_installer="${LAUNCHER_INSTALLER_PATH}"

    if [ -z "$launcher_installer" ]; then
        if ! download_launcher_installer; then
            return 1
        fi
        launcher_installer="${LAUNCHER_INSTALLER_PATH}"
    fi

    # Initialize Wineprefix with Proton Base
    log info "Initializing WINEPREFIX with Proton base"
    
    run_command true "$umu_run" winetricks arch=64
    run_command true "$umu_run" "$msidb"
    run_command true "$wineboot" -u

    # Configure WINEPREFIX
    log info "Configuring WINEPREFIX with things Bellum likes"
    log info "Visual Studio C++ Redistributable (vcrun2022) install may pop up a GUI window; please follow the prompts."
    run_command true "$winetricks" vcrun2022
    if [ $? -ne 0 ]; then
        log error "vcrun2022 installation failed."
        return 1
    fi
    
    run_command true "$winetricks" grabfullscreen=y windowmanagerdecorated=n
    if [ $? -ne 0 ]; then
        log error "Winetricks configuration failed."
        return 1
    fi
    
    run_command true "$winetricks" remove_mono
    if [ $? -ne 0 ]; then
        log error "Mono removal failed."
        return 1
    fi

    # Install Launcher
    log info "Time to install the launcher! Follow the on screen prompts once the GUI pops up."
    run_command true "$umu_run" "$launcher_installer" >/dev/null 2>&1
    if [ $? -ne 0 ]; then
        log error "Launcher installation failed."
        return 1
    fi

    if ! install_dxvk; then
        return 1
    fi

    if ! install_vkd3d; then
        return 1
    fi


    cleanup_launcher_installer
}

#======================================#
#    Print Pre-Installation Summary    #
#======================================#
print_installer_summary

# Prompt user to proceed with installation
echo
if ! confirm_proceed; then
    exit 0
fi

# Start installation
run_installer

# #### REMOVE ME ####
# echo "NUKING ${WINEPREFIX} FOR TESTING PURPOSES"
# rm -rf ${WINEPREFIX}
