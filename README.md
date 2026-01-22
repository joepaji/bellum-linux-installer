# Bellum Linux Installer

This workspace contains a Bash-based installer and precheck flow for running Bellum on Linux with Wine/Proton.

## What's Included
- `installer.sh`: Main installer entrypoint.
- `precheck.sh`: Dependency checks and environment validation.
- `utils.sh`: Shared helpers (logging, command execution, package extraction, downloads).
- `versions.env`: Single source of truth for tool versions.
- `packages/`: Local archives for winetricks, dxvk, and vkd3d.

## Quick Start
```bash
./installer.sh --wineprefix /path/to/WINEPREFIX --launcher-installer /path/to/AstarteLauncher-amd64-installer.exe
```

You can also set `WINEPREFIX` in the environment and omit `--wineprefix`.
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
- Keep large binaries (e.g., Proton builds) out of Git and download them separately.
- Archives expected in `packages/`:
  - `winetricks-${WINETRICKS_VER}.tar.gz`
  - `dxvk-${DXVK_VER}.tar.gz`
  - `vkd3d-${VKD3D_VER}.tar.gz`

## License
Add your license information here.
