#!/usr/bin/env bash
# Installation phase for Bellum Installer
# Handles package installation and game setup

PHASE_SCRIPT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source configuration
source "${PHASE_SCRIPT}/../config/versions.env"

# Source utilities
source "${PHASE_SCRIPT}/../lib/core.sh"
source "${PHASE_SCRIPT}/../lib/commands.sh"
source "${PHASE_SCRIPT}/../lib/launcher.sh"

install_dxvk() {
    local gpu_type="${1:-}"

    # Only install DXVK for AMD GPUs
    if [[ "$gpu_type" != *"AMD"* ]] && [[ "$gpu_type" != *"Radeon"* ]]; then
        log info "Skipping DXVK installation for non-AMD GPU: $gpu_type"
        return 0
    fi

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

    run_command silent "$install_dir/dxvk_setup.sh" install
    local rc=$?
    if [ $rc -ne 0 ]; then
        log error "DXVK installation failed."
        rm -rf "$tmp_dir"
        return $rc
    fi

    run_command silent cp "$install_dir/dxvk.conf" $WINEPREFIX/
    local rc=$?
    if [ $rc -ne 0 ]; then
        log error "Failed to copy dxvk.conf."
        rm -rf "$tmp_dir"
        return $rc
    fi

    rm -rf "$tmp_dir"
    cleanup_packages_tmp_root "$archive"
    log info "✓ DXVK installed"
}

run_installer() {
    log info "Starting Installation"
    echo

    export PROTONPATH="${PROTONPATH}"
    export WINEPREFIX="${WINEPREFIX_ARG:-$WINEPREFIX}"
    export WINEARCH="win64"
    export STEAM_APP_PATH="${WINEPREFIX}"
    export STEAM_APPID="1"
    export STEAM_COMPAT_DATA_PATH="${WINEPREFIX}"
    export STEAM_COMPAT_CLIENT_INSTALL_PATH="${HOME}/.steam/steam"
    export GAMEID="1"

    local msidb="${MSIDB_BIN}"
    local wine="${WINE_BIN}"
    local wineboot="${WINEBOOT_BIN}"
    local winecfg="${WINECFG_BIN}"
    local wineserver="${WINESERVER_BIN}"
    local umu_run="${UMU_RUN_BIN}"
    local winetricks="${WINETRICKS_BIN}"
    local launcher_installer="${LAUNCHER_INSTALLER_PATH}"
    local proton="${PROTONPATH}/proton"
    local FSR_PATH="${WORKDIR}/packages/fsr4"

    if [ -z "$launcher_installer" ]; then
        if ! download_launcher_installer; then
            log error "Failed to download launcher installer"
            return 1
        fi
        launcher_installer="${LAUNCHER_INSTALLER_PATH}"
    else
        if ! cleanup_launcher_installer; then
            log warn "Failed to clean up previous launcher installer"
        fi
    fi

    log info "Initializing WINEPREFIX with Proton base"
    run_command silent "$umu_run" "/usr/bin/msidb"
    # "$umu_run" "/usr/bin/msidb"
    run_command silent "$wineboot" --init

    log info "Installing required winedlls"
    while IFS= read -r dll; do
        [[ -z "$dll" ]] && continue
        run_command silent $winetricks -q "$dll"
        if [ $? -ne 0 ]; then
            log error "Failed to install $dll"
            return 1
        else 
            log info "✓ $dll"
        fi
    done < <(
        echo "vcrun2026"
        echo "d3dcompiler_43"
        echo "d3dcompiler_47"
        echo "faudio"
        echo "msls31"
        echo "dotnet9"
        echo "dotnetdesktop9"
        echo "mfc140"
    )

    echo
    log info "Time to install the launcher! Follow the on screen prompts once the GUI pops up."
    run_command silent "$wineserver" -k
    run_command silent "$proton" run "$launcher_installer"
    if [ $? -ne 0 ]; then
        log error "Launcher installation failed."
        return 1
    fi

    log info "Astarte Launcher install completed successfully! Few more steps to go..."
    log warn "I'm not done! Don't launch game or close this script just yet"
    run_command silent $winetricks win11

    echo
    if ! install_dxvk "$GPU_TYPE"; then
        return 1
    fi

    log info "Configuring WINEPREFIX with things Bellum likes"
    run_command silent $winetricks grabfullscreen=y windowmanagerdecorated=n mwo=disabled
    if [ $? -ne 0 ]; then
        log error "Winetricks configuration failed."
        return 1
    fi

    if [[ $GPU_TYPE = "AMD" ]]; then
        run_command silent $winetricks "remove_mono"
        if [ $? -ne 0 ]; then
            log error "Mono removal failed."
            return 1
        fi
    fi

    generate_launcher_executable "${WINEPREFIX_ARG:-$WINEPREFIX}" "${PROTONPATH}"
    cleanup_launcher_installer

    run_command silent "$wine" reg add 'HKCU\Software\Wine\DirectInput' /v RawInput /t REG_DWORD /d 1 /f
    run_command silent $wineboot --end-session

    return 0
}
