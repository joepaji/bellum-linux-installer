#!/usr/bin/bash

# =============================================================================
# Bellum Precheck Script - Validates all dependencies before installation
# =============================================================================
# This script validates:
#   - WINEPREFIX path validity
#   - Required Wine binaries
#   - Wine 11.0 stable version
#   - umu-run binary
#   - winetricks binary
# =============================================================================

# Source configuration from parent script
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
if ! source "${SCRIPT_DIR}/utils.sh"; then
    echo "ERROR: Failed to load utils.sh"
    return 1
fi

if ! require_installer_env; then
    return 1
fi

: '
=============================================================
#                 PRECHECK PHASE - START                    #
=============================================================
  This phase validates all required dependencies and 
  configuration before proceeding with installation:
  
  - WINEPREFIX path validation
  - Wine required binaries check
  - Wine 11.0 stable version check
  - umu-run binary availability
  - winetricks binary availability/installation
============================================================
'

log info "Starting precheck phase..."
echo

precheck_validate_wineprefix() {
    # ============================================================
    # PRECHECK 1: Validate WINEPREFIX Path
    # ============================================================
    log info "Validating WINEPREFIX path..."

    local WINEPREFIX_SOURCE=""
    local USE_EXISTING=""

    if [ -n "$WINEPREFIX_ARG" ]; then
        WINEPREFIX="$WINEPREFIX_ARG"
        WINEPREFIX_SOURCE="$(basename "$0") argument"
    elif [ -n "$WINEPREFIX" ]; then
        WINEPREFIX_SOURCE="environment variable"
        log info "WINEPREFIX is already set to: ${COLOR_BOLD_YELLOW}$WINEPREFIX${COLOR_RESET}"
        echo
        read -p "Do you want to use this path? (Y/n): " USE_EXISTING

        if [ -n "$USE_EXISTING" ] && [[ ! "$USE_EXISTING" =~ ^[Yy]$ ]]; then
            echo -n "Enter the desired WINEPREFIX path (e.g. /mx500/games/Bellum): "
            read WINEPREFIX
            WINEPREFIX_SOURCE="user input"

            if [ -z "$WINEPREFIX" ]; then
                log error "WINEPREFIX path cannot be empty"
                return 1
            fi
        fi
    else
        log error "WINEPREFIX not provided."
        if command -v print_usage &> /dev/null; then
            echo
            print_usage
        else
            echo
            echo "Usage: $0 [--force-wine-version] [--wineprefix PATH] [--launcher-installer PATH]"
            echo "Positional args: [WINEPREFIX] [LAUNCHER_INSTALLER_PATH]"
            echo
            echo "Notes:"
            echo "  - If provided, --wineprefix/positional WINEPREFIX overrides the WINEPREFIX env var."
            echo "  - If launcher installer path is omitted, it will be downloaded (requires wget)."
            echo "  - WINEPREFIX can also be set via the WINEPREFIX environment variable."
        fi
        return 1
    fi

    # Trim trailing forward slash from WINEPREFIX
    WINEPREFIX="${WINEPREFIX%/}"
    # Log the final WINEPREFIX being used
    log info "Using WINEPREFIX: ${WINEPREFIX:-<not set>} (source: ${WINEPREFIX_SOURCE})"

    # Validate that WINEPREFIX is an absolute path
    if [[ ! "$WINEPREFIX" =~ ^/ ]]; then
        log error "WINEPREFIX must be an absolute path (starting with /): $WINEPREFIX"
        return 1
    fi

    # Validate that WINEPREFIX path is on a valid mounted filesystem
    # Find the closest existing parent directory
    local WINEPREFIX_PARENT="$WINEPREFIX"
    while [ ! -d "$WINEPREFIX_PARENT" ] && [ "$WINEPREFIX_PARENT" != "/" ]; do
        WINEPREFIX_PARENT=$(dirname "$WINEPREFIX_PARENT")
    done

    # Check if the parent exists and is writable
    if [ ! -d "$WINEPREFIX_PARENT" ]; then
        log error "WINEPREFIX path is not on a valid mounted filesystem: $WINEPREFIX"
        return 1
    fi

    if [ ! -w "$WINEPREFIX_PARENT" ]; then
        log error "WINEPREFIX parent directory is not writable: $WINEPREFIX_PARENT"
        return 1
    fi

    log info "✓ WINEPREFIX path is valid and writable"

    # Verify WINEPREFIX directory does not exist
    if [ -d "$WINEPREFIX" ]; then
        log error "WINEPREFIX directory already exists at $WINEPREFIX \nThe WINEPREFIX must be a directory that does not yet exist."
        return 1
    fi

    # Check if the drive containing WINEPREFIX is SSD/NVME
    # Use the parent directory that we know exists
    local WINEPREFIX_DEVICE
    local DEVICE_NAME
    local DEVICE_BASE
    local IS_SSD=0
    local SYSFS_PATH
    local ROTATIONAL

    WINEPREFIX_DEVICE=$(df "$WINEPREFIX_PARENT" 2>/dev/null | tail -1 | awk '{print $1}')
    DEVICE_NAME=$(basename "$WINEPREFIX_DEVICE")

    if command -v lsblk &> /dev/null; then
        ROTATIONAL=$(lsblk -no rota "$WINEPREFIX_DEVICE" 2>/dev/null | head -n 1)
        if [ "$ROTATIONAL" = "0" ]; then
            IS_SSD=1
        fi
    else
        # Remove partition number (e.g., sda1 -> sda, nvme0n1p1 -> nvme0n1)
        if [[ "$DEVICE_NAME" == nvme* ]]; then
            DEVICE_BASE=$(echo "$DEVICE_NAME" | sed 's/p[0-9]*$//')
        else
            DEVICE_BASE=$(echo "$DEVICE_NAME" | sed 's/[0-9]*$//')
        fi

        SYSFS_PATH="/sys/block/${DEVICE_BASE}/queue/rotational"
        if [ -f "$SYSFS_PATH" ]; then
            ROTATIONAL=$(cat "$SYSFS_PATH" 2>/dev/null)
            if [ "$ROTATIONAL" = "0" ]; then
                IS_SSD=1
            fi
        fi
    fi

    if [ $IS_SSD -eq 1 ]; then
        log info "✓ WINEPREFIX device is an SSD/NVME (optimal performance)"
    else
        log warn "WINEPREFIX device is NOT an SSD/NVME (may have performance issues)"
        echo
        if ! confirm_proceed "Astarte Developers strongly recommend using NVMe or SSD for the game. Are you sure you want to proceed? (Y/n): "; then
            return 1
        fi
    fi
}

precheck_required_wine_binaries() {
    # ============================================================
    # PRECHECK 2: Verify Required Wine Binaries
    # ============================================================
    log info "Checking required Wine binaries..."

    local MISSING_BINARIES=()

    for binary in "${REQUIRED_WINE_BINARIES[@]}"; do
        if [ ! -f "$binary" ]; then
            MISSING_BINARIES+=("$binary")
        fi
    done

    if [ ${#MISSING_BINARIES[@]} -gt 0 ]; then
        log error "Required Wine binaries not found:"
        for binary in "${MISSING_BINARIES[@]}"; do
            echo "  - $binary"
        done
        echo
        echo "${COLOR_BOLD_YELLOW}Please install Wine with the required binaries:${COLOR_RESET}"
        echo "  - Fedora: ${COLOR_BOLD}sudo dnf install wine${COLOR_RESET}"
        echo "  - Ubuntu/Debian: ${COLOR_BOLD}sudo apt install wine wine-tools${COLOR_RESET}"
        echo "  - Arch: ${COLOR_BOLD}sudo pacman -S wine${COLOR_RESET}"
        return 1
    fi

    MSIDB_BIN="/usr/bin/msidb"
    WINE_BIN="/usr/bin/wine"
    WINEBOOT_BIN="/usr/bin/wineboot"

    log info "✓ All required Wine binaries found"
}

precheck_wine_version() {
    # ============================================================
    # PRECHECK 3: Verify Wine 11.0 Stable Version
    # ============================================================
    log info "Verifying Wine version..."

    local INSTALLED_WINE
    INSTALLED_WINE=$(wine --version 2>/dev/null | grep -oP 'wine-\K[0-9.]+' || echo "")

    if [ -z "$INSTALLED_WINE" ]; then
        log error "Wine binary not found in PATH"
        return 1
    fi

    if [ "$INSTALLED_WINE" != "11.0" ]; then
        if [ "$FORCE_WINE_VERSION" = false ]; then
        log error "Wine version mismatch. Installed: wine-${INSTALLED_WINE}, Required: ${WINE_VER}"
            echo
            echo "${COLOR_BOLD_YELLOW}Please install Wine 11.0 stable release for your distribution:${COLOR_RESET}"
            echo "  - Fedora: ${COLOR_BOLD}sudo dnf install winehq-stable${COLOR_RESET}"
            echo "  - Ubuntu/Debian: ${COLOR_BOLD}sudo apt install --install-recommends winehq-stable${COLOR_RESET}"
            echo
            echo "Ensure you are using the stable release channel and NOT development or staging versions."
            echo "Bellum works best with the latest Wine 11.0-stable based on my testing."
            echo
            echo "${COLOR_BOLD_YELLOW}NOTE:${COLOR_RESET} If your upstream is pointing to the non-stable wine-11.0 version, please refer to the WineHQ documentation to switch to stable:"
            echo
            echo "${COLOR_BOLD}Wine downloads${COLOR_RESET}: https://www.winehq.org/download"
            echo
            echo "${COLOR_BOLD_YELLOW}To proceed with wine-${INSTALLED_WINE} (not recommended):${COLOR_RESET}"
            echo "  ${COLOR_BOLD}$0 --force-wine-version $WINEPREFIX_ARG${COLOR_RESET}"
            return 1
        else
            log warn "Wine version mismatch. Installed: wine-${INSTALLED_WINE}, Required: ${WINE_VER}"
            log warn "Proceeding with wine-${INSTALLED_WINE} due to --force-wine-version flag (not recommended)"
        fi
    else
    log info "✓ Wine ${WINE_VER} stable found"
    fi
}

precheck_umu_run() {
    # ============================================================
    # PRECHECK 4: Verify umu-run Binary
    # ============================================================
    log info "Checking umu-run binary..."

    if [ -z "$(command -v umu-run)" ]; then
        log error "umu-run binary not found in PATH.\nGrab latest umu-launcher-1.3.0 for your distro: https://github.com/Open-Wine-Components/umu-launcher/releases/tag/1.3.0"
        
        echo
        echo "${COLOR_BOLD_BLUE}Umu-Launcher Install Fedora 43 Example${COLOR_RESET}: 
    1) From umu-launcher releases page, download or copy the link to the appropriate .rpm for your system
    2) Install using one of the methods below:

      ${COLOR_BOLD}If using .rpm URL (direct release link)${COLOR_RESET}:
          $ sudo dnf install -y https://github.com/Open-Wine-Components/umu-launcher/releases/download/1.3.0/umu-launcher-1.3.0.fc43.rpm
      
      ${COLOR_BOLD}If using downloaded .rpm file${COLOR_RESET}:
          $ sudo dnf install -y /path/to/downloaded/umu-launcher-*.rpm          
      ${COLOR_BOLD}Verify installation${COLOR_RESET}:
          $ umu-run --version
          umu-launcher version 1.3.0 (3.14.2 (main, Dec  5 2025, 00:00:00) [GCC 15.2.1 20251111 (Red Hat 15.2.1-4)])"
        
        echo
        echo "${COLOR_BOLD}Once umu-launcher is installed and the umu-run binary is verified, re-run this script.${COLOR_RESET}"
        return 1
    fi

    UMU_RUN_BIN="$(command -v umu-run)"
    log info "✓ umu-run binary found: $(command -v umu-run)"
}

precheck_launcher_installer() {
    # ============================================================
    # PRECHECK 5: Verify/Download Launcher Installer
    # ============================================================
    if [ -n "$LAUNCHER_INSTALLER_PATH" ]; then
        if [ ! -f "$LAUNCHER_INSTALLER_PATH" ]; then
            log error "Launcher installer not found at: $LAUNCHER_INSTALLER_PATH"
            return 1
        fi
        log info "✓ Launcher installer found: $LAUNCHER_INSTALLER_PATH"
        return 0
    fi

    if ! command -v wget &> /dev/null; then
        log error "Launcher installer path not provided and wget is not available."
        echo
        echo "${COLOR_BOLD_YELLOW}Provide the launcher installer path or install wget:${COLOR_RESET}"
        echo "  - Download: ${COLOR_BOLD}https://playbellum.com/download/${COLOR_RESET}"
        echo "  - Fedora: ${COLOR_BOLD}sudo dnf install wget${COLOR_RESET}"
        echo "  - Ubuntu/Debian: ${COLOR_BOLD}sudo apt install wget${COLOR_RESET}"
        echo "  - Arch: ${COLOR_BOLD}sudo pacman -S wget${COLOR_RESET}"
        return 1
    fi

    log info "✓ wget found for launcher installer download"
}

precheck_winetricks() {
    # ============================================================
    # PRECHECK 5: Verify/Install winetricks Binary
    # ============================================================
    log info "Checking winetricks binary..."

    if command -v winetricks &> /dev/null; then
        WINETRICKS_BIN="$(command -v winetricks)"
        log info "winetricks found, running self-update..."
        sudo winetricks --self-update &>/dev/null
        if [ $? -eq 0 ]; then
            log info "✓ winetricks updated successfully"
        else
            log warn "winetricks self-update failed, but binary exists"
        fi
    else
        log warn "winetricks binary not found, attempting to install from local archive..."
        
        local WINETRICKS_ARCHIVE="packages/winetricks-${WINETRICKS_VER}.tar.gz"
        local WINETRICKS_DIR
        
        if [ ! -f "$WINETRICKS_ARCHIVE" ]; then
            log error "winetricks binary not found in PATH and $WINETRICKS_ARCHIVE not found"
            return 1
        fi
        
        log info "Extracting $WINETRICKS_ARCHIVE into packages/.tmp/winetricks/..."
        WINETRICKS_DIR="$(extract_package_to_tmp "$WINETRICKS_ARCHIVE" "winetricks")"
        if [ $? -ne 0 ] || [ -z "$WINETRICKS_DIR" ]; then
            log error "Failed to extract $WINETRICKS_ARCHIVE"
            return 1
        fi
        
        if [ ! -d "$WINETRICKS_DIR" ]; then
            log error "Expected directory $WINETRICKS_DIR not found after extraction"
            return 1
        fi
        
        log info "Installing winetricks..."
        cd "$WINETRICKS_DIR"
        sudo make install
        if [ $? -ne 0 ]; then
            log error "Failed to install winetricks"
            cd "$WORKDIR"
            return 1
        fi
        cd "$WORKDIR"
        
        # Verify installation
        if ! command -v winetricks &> /dev/null; then
            log error "winetricks binary not found after installation"
            return 1
        fi
        
        log info "Running winetricks self-update..."
        sudo winetricks --self-update &>/dev/null
    if [ $? -eq 0 ]; then
        log info "✓ winetricks installed and updated successfully"
    else
        log error "winetricks installed but self-update failed"
        return 1
    fi

    WINETRICKS_BIN="$(command -v winetricks)"
    
        # Clean up extracted directory (keep the tar.gz)
        log info "Cleaning up extracted winetricks directory..."
        rm -rf "$WINETRICKS_DIR"
        cleanup_packages_tmp_root "$WINETRICKS_ARCHIVE"
    fi
}

precheck_proton_ge() {
    # ============================================================
    # PRECHECK 6: Ensure Proton GE is available and patched
    # ============================================================
    log info "Ensuring Proton GE ${PROTON_VER} is available..."

    local proton_dir="${WORKDIR}/${PROTON_VER}"
    if [ ! -d "$proton_dir" ]; then
        if ! command -v wget &> /dev/null; then
            log error "Proton GE is missing and wget is not available to download it."
            echo
            echo "${COLOR_BOLD_YELLOW}Install wget or place ${PROTON_VER} in the project root:${COLOR_RESET}"
            echo "  - Fedora: ${COLOR_BOLD}sudo dnf install wget${COLOR_RESET}"
            echo "  - Ubuntu/Debian: ${COLOR_BOLD}sudo apt install wget${COLOR_RESET}"
            echo "  - Arch: ${COLOR_BOLD}sudo pacman -S wget${COLOR_RESET}"
            return 1
        fi
        log info "✓ wget found for Proton GE download"
    fi

    ensure_proton_ge
}

run_prechecks() {
    precheck_validate_wineprefix || return 1
    precheck_required_wine_binaries || return 1
    precheck_wine_version || return 1
    precheck_umu_run || return 1
    precheck_launcher_installer || return 1
    precheck_winetricks || return 1
    precheck_proton_ge || return 1
}

if ! run_prechecks; then
    return 1
fi

: '
=============================================================
#                  PRECHECK PHASE - END                     #
=============================================================
'

log info "✓ All prechecks passed!"
echo

return 0
