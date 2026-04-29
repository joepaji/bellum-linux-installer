#!/usr/bin/env bash
# Package extraction and download helpers for Bellum Installer

# Do not redefine SCRIPT_DIR - it's defined in the main script

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

# get_proton_url() {
#     if [ -z "${PROTON_BASE_URL:-}" ] || [ -z "${PROTON_VER:-}" ]; then
#         return 1
#     fi
#     echo "${PROTON_BASE_URL}/${PROTON_VER}/${PROTON_VER}.tar.gz"
# }

get_amd_proton_url() {
    if [ -z "${AMD_PROTON_BASE_URL:-}" ] || [ -z "${AMD_PROTON_VER:-}" ]; then
        return 1
    fi
    prefix=$(echo "$AMD_PROTON_VER" | sed -E 's/^proton-//; s/-x86_64$//')
    echo "${AMD_PROTON_BASE_URL}/${prefix}/${AMD_PROTON_VER}.tar.xz"
}

get_nv_proton_url() {
    if [ -z "${NV_PROTON_BASE_URL:-}" ] || [ -z "${NV_PROTON_VER:-}" ]; then
        return 1
    fi
    prefix=$(echo "$NV_PROTON_VER" | sed -E 's/^proton-//; s/-x86_64$//')
    echo "${NV_PROTON_BASE_URL}/${NV_PROTON_VER}.tar.gz"
}

get_nv_local_proton_path() {
    local local_proton_path="${WORKDIR}/packages/${NV_PROTON_VER}.tar.gz"
    if [ ! -f "$local_proton_path" ]; then
        log error "Local Nvidia Proton package not found: $local_proton_path"
        return 1
    fi
    echo "$local_proton_path"
}


patch_proton_user_settings() {
    local settings_file="$1"
    if [ -z "$settings_file" ] || [ ! -f "$settings_file" ]; then
        log error "Proton user settings file not found: $settings_file"
        return 1
    fi

    # Determine which FSR settings to apply based on GPU type
    local gpu_type="${GPU_TYPE:-}"
    local is_amd_gpu=0
    if [[ "$gpu_type" == *"AMD"* ]] || [[ "$gpu_type" == *"Radeon"* ]]; then
        is_amd_gpu=1
    fi

    local tmp_file
    tmp_file="$(mktemp)"
    if [ -z "$tmp_file" ]; then
        log error "Failed to create temp file for Proton settings patch"
        return 1
    fi

    awk -v tmp_file="$tmp_file" -v is_amd="$is_amd_gpu" '
        BEGIN {
            if (is_amd == 1) {
                desired["PROTON_FSR4_UPGRADE"]="4.1.0"
                desired["PROTON_FSR4_RDNA3_UPGRADE"]="4.1.0"
            } else {
                desired["PROTON_ENABLE_NVAPI"]="1"
                desired["PROTON_ENABLE_NGX_UPDATER"]="1"
                desired["PROTON_DLSS_UPGRADE"]="1"
                desired["MALLOC_ARENA_MAX"]="1"
                desired["PROTON_VKD3D_HEAP"]="1"
                desired["PROTON_DXVK_D3D8"]="1"
                desired["PROTON_NVIDIA_LIBS"]="1"
            }
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

ensure_proton() {
    # Determine GPU type for URL selection
    local gpu_type="${GPU_TYPE:-}"
    local is_amd_gpu=0
    if [[ "$gpu_type" == *"AMD"* ]] || [[ "$gpu_type" == *"Radeon"* ]]; then
        is_amd_gpu=1
    fi

    local PROTON_VER=""
    local proton_url=""
    local archive_ext=""
    local local_proton_path=""

    # Set the appropriate version variable and URL getter based on GPU type
    if [ "$is_amd_gpu" -eq 1 ]; then
        PROTON_VER="${AMD_PROTON_VER}"
        proton_url="$(get_amd_proton_url)"
        archive_ext=".tar.xz"
    else
        PROTON_VER="${NV_PROTON_VER}"
        local_proton_path="$(get_nv_local_proton_path)"
        if [ $? -ne 0 ] || [ -z "$local_proton_path" ]; then
            log error "Failed to get local Nvidia Proton path"
            return 1
        fi
        archive_ext=".tar.gz"
    fi

    local proton_dir="${WORKDIR}/${PROTON_VER}"
    local settings_sample="${proton_dir}/user_settings.sample.py"
    local settings_file="${proton_dir}/user_settings.py"

    # For AMD GPUs, use the download URL; for Nvidia GPUs, use the local path
    if [ "$is_amd_gpu" -eq 1 ]; then
        if [ -z "$proton_url" ]; then
            log error "Unable to determine Proton download URL"
            return 1
        fi
    fi

    if [ ! -d "$proton_dir" ]; then
        log info "Proton directory not found, checking for cached version..."
        if [ -d "${WORKDIR}/proton-*" ]; then
            local cached_proton
            cached_proton=$(ls -d "${WORKDIR}/proton-*" 2>/dev/null | head -n1)
            log info "Found cached Proton in ${cached_proton}"
            return 0
        fi

        if [ "$is_amd_gpu" -eq 1 ]; then
            # AMD GPU: Download and extract
            local tmp_dir
            tmp_dir="$(mktemp -d -t "proton.XXXXXX")"
            if [ -z "$tmp_dir" ] || [ ! -d "$tmp_dir" ]; then
                log error "Failed to create temp directory for Proton download"
                return 1
            fi

            local archive="${tmp_dir}/${PROTON_VER}${archive_ext}"
            log info "Downloading Proton ${PROTON_VER}..."
            if ! wget -O "$archive" "$proton_url" >> "${LOG_FILE}" 2>&1; then
                rm -rf "$tmp_dir"
                return 1
            fi

            log info "Extracting Proton to ${WORKDIR}/${PROTON_VER}..."
            mkdir -p "${WORKDIR}/${PROTON_VER}"
            if ! tar -xvf "$archive" -C "${WORKDIR}/${PROTON_VER}" --strip-components=1 >> "${LOG_FILE}" 2>&1; then
                rm -rf "$tmp_dir" "${WORKDIR}/${PROTON_VER}"
                return 1
            fi
            rm -rf "$tmp_dir"
        else
            # Nvidia GPU: Extract from local package
            log info "Extracting local Nvidia Proton from ${local_proton_path}..."
            mkdir -p "${WORKDIR}/${PROTON_VER}"
            if ! tar -xzf "$local_proton_path" -C "${WORKDIR}/${PROTON_VER}" --strip-components=1 >> "${LOG_FILE}" 2>&1; then
                rm -rf "${WORKDIR}/${PROTON_VER}"
                return 1
            fi
        fi
    fi

    if [ -f "$settings_sample" ] && [ ! -f "$settings_file" ]; then
        mv "$settings_sample" "$settings_file"
    fi

    if [ -f "$settings_file" ]; then
        if ! patch_proton_user_settings "$settings_file"; then
            log error "Failed to patch Proton user settings, removing and re-downloading Proton"
            rm -rf "$proton_dir"
            return 1
        fi
    else
        log error "Proton user settings file missing after setup: $settings_file"
        log error "Please run the uninstall.sh script then re-run the installer."
        return 1
    fi

    export PROTONPATH="${WORKDIR}/${PROTON_VER}"

    return 0
}

download_launcher_installer() {
    local download_dir="${WORKDIR}/installer-cache"
    local filename="AstarteLauncher-amd64-installer.exe"
    local dest="${download_dir}/${filename}"

    mkdir -p "$download_dir"
    log info "Downloading launcher installer to: ${dest}"
    if ! wget -O "$dest" "https://auto-updater.astarte.industries/astartelauncher/windows-amd64/AstarteLauncher-amd64-installer.exe" >> "${LOG_FILE}" 2>&1; then
        log error "Failed to download launcher installer"
        return 1
    fi
    
    if [ ! -f "$dest" ]; then
        log error "Download verification failed: launcher installer not found"
        return 1
    fi

    LAUNCHER_INSTALLER_PATH="$dest"
    LAUNCHER_INSTALLER_DOWNLOADED="true"
    LAUNCHER_INSTALLER_DIR="$download_dir"
}

cleanup_launcher_installer() {
    if [ "${LAUNCHER_INSTALLER_DOWNLOADED}" = "true" ] && [ -n "$LAUNCHER_INSTALLER_PATH" ]; then
        if [ -f "$LAUNCHER_INSTALLER_PATH" ]; then
            log info "Cleaning up downloaded launcher installer..."
            rm -f "$LAUNCHER_INSTALLER_PATH"
        fi
        if [ -n "$LAUNCHER_INSTALLER_DIR" ]; then
            if ! rmdir "$LAUNCHER_INSTALLER_DIR" 2>/dev/null; then
                log warn "Failed to remove launcher installer directory: $LAUNCHER_INSTALLER_DIR (directory not empty or does not exist)"
            fi
        fi
    fi
}
