# Bellum Linux Installer

This repo provides a guided Bash installer to set up Bellum on Linux using Wine (11.0 stable), Proton GE, DXVK, VKD3D, and winetricks. It includes prechecks, dependency guidance, and a post-install Lutris configuration reference.

## What's Included
- `installer.sh`: Main installer entrypoint.
- `utils/precheck.sh`: Dependency checks and environment validation.
- `utils/utils.sh`: Shared helpers (logging, command execution, package extraction, downloads).
- `utils/versions.env`: Single source of truth for tool versions.
- `packages/`: Local archives for winetricks, DXVK, and VKD3D.
- `GE-Proton10-28/`: Bundled Proton GE build (used or re-downloaded if missing).
- `scripts/streamer_test.sh`: Command output streamer test harness.

## Requirements
Required:
- `bash`
- Wine 11.0 stable (`wine`, `wineboot`, `msidb`)
- `umu-run` (umu-launcher 1.3.0)
- `winetricks` (installer can install from `packages/` if missing; requires `sudo` + `make`)
- `wget`
- `tar`
- `awk`, `sed`, `df`, `lsblk`, `mktemp` (coreutils/util-linux)
- `cabextract` (required by winetricks)

Optional:
- `stdbuf` (from coreutils) for smoother streamed output

Network access is required only if Proton GE or the launcher installer needs to be downloaded.

## Unpack the Release Package
If you downloaded the release tarball, unpack it into a directory and run the installer from there:
```bash
tar -xzf bellum-linux-installer-v1.0.0.tar.gz
cd bellum-linux-installer
```

## Quick Start
```bash
./installer.sh --wineprefix /path/to/WINEPREFIX --launcher-installer /path/to/AstarteLauncher-amd64-installer.exe
```

You can also set `WINEPREFIX` in the environment and omit `--wineprefix`, or just export it before running the installer:
```bash
export WINEPREFIX=/path/to/WINEPREFIX
./installer.sh --launcher-installer /path/to/AstarteLauncher-amd64-installer.exe
```

If `--launcher-installer` is not provided, the installer will download it (requires `wget`).

## Usage
```bash
./installer.sh [--force-wine-version] [--wineprefix PATH] [--launcher-installer PATH]
```

Positional args are also accepted:
```bash
./installer.sh [WINEPREFIX] [LAUNCHER_INSTALLER_PATH]
```

## Notes
- `WINEPREFIX` must be an absolute path and must not already exist. The precheck flow will prompt if a value is missing.
- Proton GE is downloaded and extracted if missing, and `user_settings.py` is patched automatically.
- Logs are written to `logs/installer.log` when the installer runs.
- `packages/.tmp/` is used for temporary extraction during installs.

## Post-Install: Lutris Setup
At the end of the install, the script prints the exact Lutris settings and writes them to `lutris_profile.txt` in the project root for later reference.

Use the following configuration in Lutris (the installer prints the resolved `WINEPREFIX` and executable path):

## Lutris Game Profile

### [Game Info]
---
**Name:** Bellum  
**Runner:** Wine

### [Game Options Tab]
---
**Executable:** $WINEPREFIX/drive_c/users/steamuser/AppData/Local/Astarte Industries/Astarte Launcher/AstarteLauncher.exe
**Wine Prefix:** $WINEPREFIX
**Prefix architecture:** 64 bit

### [Runner Options Tab]
---
**Wine Version:** System (11.0)  
**Enable DXVK:** Toggle On  
**DXVK Version:** Manual  
**Enable VKD3D:** Toggle On  
**VKD3D Version:** Manual  
**Enable D3D Extras:** Toggle On  
**D3D Extras Version:** v2 (default)  
**Enable DXVK-NVAPI / DLSS:** Toggle On  
**DXVK NVAPI Version:** v0.9.0 (default)  

### [System Options]
**Disable Lutris Runtime:** Toggle On  
**Prefer System Libraries:** Toggle On  
**Enable Gamemode:** On (unless you use falcond)

### Notes:
- Gamemode is optional but highly recommended.
- Settings not mentioned here can be left default or customized at will.
