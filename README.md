# Bellum Linux Installer

Welcome to the Bellum Linux Installer and Uninstaller. This is a linux Proton/Wine based install orchestrator for Bellum.
Please note, this is NOT official support or an official native release. I am just a supporter of the game and am not affiliated with Astarte Industries.
That said, my goal is making sure none of my fellow linux gamers have to see the light of Windows to play this awesome game.

## Download & Install

### Unpack the Release Package

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


**Optional FSR 4.1 for AMD Users:**

By default, this will install the current FSR 4.0.0 level.

If you want the leaked FSR 4.1.0 level, use the **--fsr41** flag 

```bash
./installer --fsr41
```


2. Select the directory where you want to install Bellum and confirm the install summary.

The WINEPREFIX named `Bellum` will be created in the selected directory.
   
<img width="800" alt="image" src="https://github.com/user-attachments/assets/826d7e36-1471-4cd2-9c61-8440252456aa" />
<img width="800" alt="image" src="https://github.com/user-attachments/assets/5347c5bd-c44d-4f37-b89b-cdbf4e137ae9" />


3. Let the installer do its thing until the AstarteLauncher pops up.
4. Install AstarteLauncher by following the instructions in the launcher.
<img width="800"  alt="image" src="https://github.com/user-attachments/assets/5ba0340b-9d2a-45f4-954c-83befd331534" />

5. Let the installer complete some post launcher install steps and you're done!

<img width="800" alt="image" src="https://github.com/user-attachments/assets/aab9f336-9307-4e2a-b7da-f5f8655eb92b" />


**Note:** `WINEPREFIX` environment variable can also be used to install Bellum:

```bash
export WINEPREFIX=/path/to/wineprefix
./installer
```

## Playing the Game

### Option 1 - Desktop Shortcut
This guy will be added to your Desktop once install is complete:
<img width="106" height="117" alt="image" src="https://github.com/user-attachments/assets/1305bf0b-994f-4726-a75f-64e5f5bbac7e" />

### Option 2 - Application Menu
This guy will be added under the **Games** category in your Application Menu:
<img width="537" height="67" alt="image" src="https://github.com/user-attachments/assets/d6acdfed-7569-415d-8e42-dac896d7bce9" />

### Option 3 - Terminaal
Just open a terminal anywhere, and run the `Bellum` command.
<img width="800" alt="image" src="https://github.com/user-attachments/assets/e24b60bc-7aaa-4fc8-99ff-26ed24fbe7e7" />

## Uninstallation

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

- Supports FSR 4.0 and FSR 4.1 (with --fsr41)
- Supports DLSS and Nvidia Framegen (5000 series users see driver note above)
- All scripts are Go binaries with no external dependencies
- Packages are bundled in the release tarball, not statically embedded
- The installer detects GPU type and configures accordingly
- All logging is written to `logs/installer.log`
- The uninstaller removes all launcher files and optionally the WINEPREFIX
