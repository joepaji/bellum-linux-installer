# Bellum Linux Installer

This directory contains the Go implementation of the Bellum Linux Installer and Uninstaller.

## Usage

### Unpack the Release Packagec

1. Download the release tarball from the [Releases Page](https://github.com/joepaji/bellum-linux-installer/releases/latest)

2. Open a terminal in the directory where you downloaded the release tarball

3. Extract the release tarball & access extracted directory:

```bash
tar -xzf bellum-installer-linux-amd64-v2.0.0.tar.gz 
cd bellum-installer-linux-amd64-v2.0.0
```

### Install Game

1. Run the installer:

```bash
./installer
```

2. Select the directory where you want to install Bellum. The WINEPREFIX named `Bellum` will be created in the selected directory.

3. Let the installer do its thing until the AstarteLauncher pops up.

4. Install AstarteLauncher by following the instructions in the launcher.

5. Let the installer complete some post launcher install steps and you're done!

**Note:** `WINEPREFIX` environment variable can also be used to install Bellum:

```bash
export WINEPREFIX=/path/to/wineprefix
./installer
```

### Uninstallation

Set the `WINEPREFIX` env var to the one used to install the game. Then run unintsaller script.
```bash
export WINEPREFIX=/path/to/wineprefix
./uninstaller
```

## Release Tarball Structure

The release tarball (`bellum-installer-linux-amd64-v2.0.0.tar.gz`) contains:

```
bellum-installer-linux-amd64-v2.0.0.tar.gz
├── installer          # Installer binary
├── uninstaller        # Uninstaller binary
└── packages/          # All bundled packages
```

##  ** ONLY Nvidia Blackwell 5000 Series GPUs **
If you have an RTX 5000 series GPU running driver level `595`, you will need to downgrade to `590` before installing Bellum.

The driver is just plain broken for UE5 on wine/proton and it will fail to load shaders every time.

`590` is the latest driver level that is confirmed to be working for these GPUs.

## Implementation Notes

- All scripts are Go binaries with no external dependencies
- Packages are bundled in the release tarball, not statically embedded
- The installer detects GPU type and configures accordingly
- All logging is written to `logs/installer.log`
- The uninstaller removes all launcher files and optionally the WINEPREFIX
