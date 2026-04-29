#!/usr/bin/env bash
# Configuration phase for Bellum Installer
# Post-install: DLL overrides, GPU-specific config, launch vars

PHASE_SCRIPT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Set paths
MSIDB_BIN="${MSIDB_BIN:-/usr/bin/msidb}"
WINE_BIN="${WINE_BIN:-/usr/bin/wine}"
WINEBOOT_BIN="${WINEBOOT_BIN:-/usr/bin/wineboot}"
WINECFG_BIN="${WINECFG_BIN:-/usr/bin/winecfg}"
WINESERVER_BIN="${WINESERVER_BIN:-/usr/bin/wineserver}"
UMU_RUN_BIN="${UMU_RUN_BIN:-}"
WINETRICKS_BIN="${WINETRICKS_BIN:-}"
FSR_PATH="${FSR_PATH:-}"

# Source configuration
source "${PHASE_SCRIPT}/../config/versions.env"

# Source utilities
source "${PHASE_SCRIPT}/../lib/core.sh"
source "${PHASE_SCRIPT}/../lib/commands.sh"

# Detect GPU type if not already set (should be done in precheck phase)
if [ -z "$GPU_TYPE" ]; then
    log error "GPU_TYPE is not set. Run the precheck phase first."
    return 1
fi

# Create launch vars file for NVIDIA
create_launch_vars_file_nvidia() {
    local wineprefix="$1"
    local protonpath="$2"
    local launch_vars="${wineprefix}/launch_vars.env"

    log info "Creating launch environment file: ${launch_vars}"

    cat > "$launch_vars" <<EOF
PROTONPATH="${protonpath}"
WINEPREFIX="${wineprefix}"
STEAM_COMPAT_DATA_PATH="${wineprefix}"
STEAM_COMPAT_SHADER_PATH="shadercache"
STEAM_COMPAT_CLIENT_INSTALL_PATH=""
PROTON_ENABLE_NGX_UPDATER="1"
PROTON_ENABLE_NVAPI="1"
PROTON_VKD3D_HEAP="1"
PROTON_DXVK_D3D8="1"
PROTON_NVIDIA_LIBS="1"
PROTON_DLSS_UPGRADE="1"
MALLOC_ARENA_MAX="1"
VKD3D_CONFIG="descriptor_heap"
WINEESYNC="1"
WINEFSYNC="1"
DXVK_NVAPI="1"
DXVK_ENABLE_NVAPI="1"
DXVK_NVAPIHACK="0"
WEBKIT_DISABLE_DMABUF_RENDERER="1"
WINE_LARGE_ADDRESS_AWARE="1"
CUDA_DISABLE_PERF_BOOST="1"
EOF

    log info "✓ Launch environment file created"
}

# Create launch vars file for AMD
create_launch_vars_file_amd() {
    local wineprefix="$1"
    local protonpath="$2"
    local launch_vars="${wineprefix}/launch_vars.env"

    log info "Creating launch environment file: ${launch_vars}"

    cat > "$launch_vars" <<EOF
PROTONPATH="${protonpath}"
WINEPREFIX="${wineprefix}"
EOF

    log info "✓ Launch environment file created"
}

# Upgrade FSR to 4.1.0
upgrade_fsr() {
    log info "Upgrading to FSR 4.1.0"

    local fg_dll="amd_fidelityfx_framegeneration_dx12.dll"
    local d3d_dll="D3D12Core.dll"

    local fg_target_dir="${WINEPREFIX}/drive_c/Program Files/Astarte Industries/Bellum/Project_Bellum/Plugins/AMD/FSR/Source/fidelityfx-sdk/Kits/FidelityFX/signedbin/"
    local d3d_target_dir="${WINEPREFIX}/drive_c/Program Files/Astarte Industries/Bellum/Project_Bellum/Binaries/Win64/D3D12/x64/"

    local fg_target="$fg_target_dir/$fg_dll"
    local d3d_target="$d3d_target_dir/$d3d_dll"

    local fg_source="${FSR_PATH}/$fg_dll"
    local d3d_source="${FSR_PATH}/$d3d_dll"

    run_command silent mkdir -p "$fg_target_dir" "$d3d_target_dir"
    if [ $? -ne 0 ]; then
        log error "Failed to create dir $fg_target_dir"
    fi
    run_command silent cp -v "$fg_source" "$fg_target"
    if [ $? -ne 0 ]; then
        log error "Failed to copy $fg_source -> $fg_target"
    fi
    run_command silent cp -v "$d3d_source" "$d3d_target"
    if [ $? -ne 0 ]; then
        log error "Failed to create copy $d3d_source -> $d3d_target"
    fi

    run_command silent "$WINE_BIN" reg add 'HKEY_CURRENT_USER\Software\Wine\DllOverrides' \
            /v "amdxcffx64" \
            /d "native" \
            /f
    if [ $? -ne 0 ]; then
        log error "Failed to register amdxcffx64 DLL override"
    fi

    # prevent override
    log info "FSR 4.1.0 Upgrade Complete!"
}

# Update DLL overrides
update_dlls() {
    local dll

    log info "Setting DLL overrides"
    # AMD GPU-specific overrides (only for AMD users)
    # if [ "$GPU_TYPE" = "AMD" ]; then
    # System-wide overrides
    for dll in \
        d3d12 \
        d3d12core \
        d3d10core \
        d3d9 \
        d3d8
    do
        run_command silent "$WINE_BIN" reg add 'HKEY_CURRENT_USER\Software\Wine\DllOverrides' \
            /v "$dll" \
            /d "native,builtin" \
            /f
        if [ $? -ne 0 ]; then
            log error "Failed to set override for $dll"
        fi
    done
   
    for dll in \
        d3d11 \
        dxgi
    do
        run_command silent "$WINE_BIN" reg add 'HKCU\Software\Wine\AppDefaults\AstarteLauncher.exe\DllOverrides' \
            /v "$dll" \
            /d "builtin" \
            /f
        if [ $? -ne 0 ]; then
            log error "Failed to set override for $dll (launcher)"
        fi

        run_command silent "$WINE_BIN" reg add 'HKCU\Software\Wine\AppDefaults\Bellum-Win64-Shipping.exe\DllOverrides' \
            /v "$dll" \
            /d "native" \
            /f
        if [ $? -ne 0 ]; then
            log error "Failed to set override for $dll (game)"
        fi
    done

    # # Disabled overrides
    # for dll in \
    #     openvr_api \
    #     openxr_loader \
    #     vrclient \
    #     vrclient_x64 \
    #     wineopenxr
    # do
    #     run_command silent "$WINE_BIN" reg add 'HKEY_CURRENT_USER\Software\Wine\DllOverrides' \
    #         /v "$dll" \
    #         /d "" \
    #         /f
    #     if [ $? -ne 0 ]; then
    #         log error "Failed to disable override for $dll"
    #     fi
    # done
}

run_configuration() {
    echo
    log info "Starting configuration phase..."

    export PROTONPATH="${PROTONPATH}"
    export WINEPREFIX="${WINEPREFIX_ARG:-$WINEPREFIX}"

    # Update DLL overrides
    update_dlls

    # GPU-specific configuration
    if [ "$GPU_TYPE" = "NVIDIA" ]; then
        create_launch_vars_file_nvidia "${WINEPREFIX}" "${PROTONPATH}"
        # log info "Detected NVIDIA GPU - Installing dxvk_nvapi"
        # run_command silent "$WINETRICKS_BIN" -q dxvk_nvapi

    elif [ "$GPU_TYPE" = "AMD" ]; then
        upgrade_fsr
        create_launch_vars_file_amd "${WINEPREFIX}" "${PROTONPATH}"

    else
        log warn "Unknown or unsupported GPU type: $GPU_TYPE"
    fi

    log info "✓ Configuration phase complete!"
    echo

    return 0
}
