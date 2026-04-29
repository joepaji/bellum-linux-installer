#!/usr/bin/env bash

# =============================================================================
# Bellum Launcher Cleanup Script
# =============================================================================
# Removes the desktop launcher executable, icon, and .desktop entry
# that were generated outside the WINEPREFIX by the installer.
# =============================================================================

set -eo pipefail

# Load versions from config (WORKDIR is defined there)
source config/versions.env

# Setup script directory and source color utilities
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/lib/core.sh"

SYSTEM_BIN="/usr/local/bin/Bellum"
SYSTEM_BIN_PROTON="/usr/local/bin/Bellum-Proton"
DESKTOP_DIR="$HOME/Desktop"
APPS_DIR="$HOME/.local/share/applications"
ICON_DIR="$HOME/.local/share/icons/hicolor/256x256/apps"

DESKTOP_FILE="${APPS_DIR}/Bellum.desktop"
DESKTOP_COPY="${DESKTOP_DIR}/Bellum.desktop"
DESKTOP_FILE_PROTON="${APPS_DIR}/Bellum-Proton.desktop"
DESKTOP_COPY_PROTON="${DESKTOP_DIR}/Bellum-Proton.desktop"
APP_ICON="${ICON_DIR}/bellum.png"

REMOVED=0

# Validate WINEPREFIX - must be provided via argument or env var
if [ -z "$WINEPREFIX" ]; then
    echo "ERROR: WINEPREFIX is required. Set via argument or WINEPREFIX environment variable."
    exit 1
fi

# Detect GPU type and determine which Proton version to remove
GPU_TYPE=$(detect_gpu_type)
if [ $? -ne 0 ] || [ -z "$GPU_TYPE" ]; then
    echo "WARNING: Failed to detect GPU type. Using default Proton version for removal."
    PROTON_VER_TO_REMOVE="${PROTON_VER:-}"
else
    echo "Detected GPU: ${GPU_TYPE}"
    # Determine which Proton version to remove based on GPU type
    if [[ "$GPU_TYPE" == *"AMD"* ]] || [[ "$GPU_TYPE" == *"Radeon"* ]]; then
        PROTON_VER_TO_REMOVE="${AMD_PROTON_VER}"
    else
        PROTON_VER_TO_REMOVE="${NV_PROTON_VER}"
    fi
fi

# Warn if PROTON_VER_TO_REMOVE is empty
if [ -z "$PROTON_VER_TO_REMOVE" ]; then
    echo "WARNING: Proton version is empty. Skipping Proton directory removal."
fi

remove_path() {
    local path="$1"
    local label="$2"

    if [ -e "$path" ]; then
        sudo rm -rf "$path"
        echo "  ${COLOR_BOLD_GREEN}✓${COLOR_RESET} Removed ${label}: ${path}"
        REMOVED=$((REMOVED + 1))
    else
        echo "  ${COLOR_GRAY_BOLD}-${COLOR_RESET} Not found: ${label} (${path})"
    fi
}

echo "${COLOR_BOLD}${COLOR_BOLD_BLUE}Bellum Uninstaller${COLOR_RESET}"
echo "=================="
echo
echo "Target WINEPREFIX: ${COLOR_BOLD_YELLOW}${WINEPREFIX}${COLOR_RESET}"
echo

# Ask for confirmation before removing WINEPREFIX
read -p "Are you sure you want to remove the entire WINEPREFIX? (y/N): " CONFIRM

case "$CONFIRM" in
    [yY]*) ;;  # proceed
    *) echo "Uninstall cancelled by user."; exit 0 ;;
esac

remove_path "$SYSTEM_BIN" "System launcher"
remove_path "$DESKTOP_FILE" "Applications entry"
remove_path "$DESKTOP_COPY" "Desktop entry"
remove_path "$APP_ICON" "Application icon"

# Remove Proton launcher and desktop entry (NVIDIA only)
if [ "$GPU_TYPE" = "NVIDIA" ]; then
    remove_path "$SYSTEM_BIN_PROTON" "Proton launcher"
    remove_path "$DESKTOP_FILE_PROTON" "Proton applications entry"
    remove_path "$DESKTOP_COPY_PROTON" "Proton desktop entry"
fi

remove_path "$WINEPREFIX" "WINEPREFIX"
if [ -n "$PROTON_VER_TO_REMOVE" ]; then
    remove_path "${WORKDIR}/${PROTON_VER_TO_REMOVE}" "Proton directory"
fi

# Refresh the desktop database if available
if [ $REMOVED -gt 0 ] && command -v update-desktop-database &>/dev/null; then
    update-desktop-database "$APPS_DIR" 2>/dev/null || true
    echo "  ✓ Refreshed desktop database"
fi

echo
if [ $REMOVED -eq 0 ]; then
    echo "Nothing to clean up."
else
    echo "Cleaned up ${REMOVED} file(s)."
fi
