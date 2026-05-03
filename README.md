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

4. Run the installer:

```bash
./installer
```

5. 

### Uninstallation

```bash
./bellum-uninstaller -uninstall -wineprefix /path/to/wineprefix
```

## Release Tarball

The release tarball (`bellum-installer-linux-amd64-v2.0.0.tar.gz`) contains:

```
bellum-installer-linux-amd64-v2.0.0.tar.gz
├── bellum-installer          # Installer binary
├── bellum-uninstaller        # Uninstaller binary
└── packages/                 # All bundled packages
```

End users extract and run:

```bash
tar -xzf bellum-installer-linux-amd64-v2.0.0.tar.gz
./bellum-installer -install -wineprefix /path/to/wineprefix
# OR
./bellum-uninstaller -uninstall -wineprefix /path/to/wineprefix
```

## Building

```bash
# Build both binaries
make all

# Create release tarball
make release

# Clean build artifacts
make clean
```


## Implementation Notes

- All scripts are Go binaries with no external dependencies
- Packages are bundled in the release tarball, not statically embedded
- The installer detects GPU type and configures accordingly
- All logging is written to `logs/installer.log`
- The uninstaller removes all launcher files and optionally the WINEPREFIX
