package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// Launcher paths - centralized constants for all launcher-related file paths
const (
	// LauncherBinDir is the system-wide launcher binary directory
	LauncherBinDir = "/usr/local/bin"

	// LauncherBinaryPath is the system-wide launcher binary path
	LauncherBinaryPath = LauncherBinDir + "/Bellum"

	// LauncherDesktopAppName is the name of the desktop application
	LauncherDesktopAppName = "Bellum.desktop"

	// LauncherIconName is the name of the launcher icon file
	LauncherIconName = "bellum.png"

	// LauncherLaunchVarsFilename is the name of the launch variables file
	LauncherLaunchVarsFilename = "launch_vars.env"

	// LauncherLogFilename is the name of the launcher log file
	LauncherLogFilename = "launcher.log"

	// LauncherExeFilename is the name of the Astarte Launcher executable
	LauncherExeFilename = "AstarteLauncher.exe"

	// LauncherExeRelPath is the relative path to the launcher executable inside WINEPREFIX
	LauncherExeRelPath = "drive_c/users/steamuser/AppData/Local/Astarte Industries/Astarte Launcher/AstarteLauncher.exe"

	// UserApplicationsDir is the user-level applications directory
	UserApplicationsDir = ".local/share/applications"

	// UserIconsDir is the user-level icons directory
	UserIconsDir = ".local/share/icons/hicolor/256x256/apps"

	// DesktopDir is the user's Desktop directory
	DesktopDir = "Desktop"
)

// BellumInstallDir is the relative path for the Bellum installation directory
const BellumInstallDir = ".local/share/bellum"

// GetBellumInstallPath returns the full path to the Bellum install directory
func GetBellumInstallPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = os.Getenv("HOME")
	}
	if homeDir == "" {
		return "", fmt.Errorf("unable to determine home directory")
	}
	return filepath.Join(homeDir, BellumInstallDir), nil
}

// LauncherPaths returns a struct containing all launcher-related paths
type LauncherPaths struct {
	BinaryPath     string
	DesktopPath    string
	IconPath       string
	LaunchVarsPath string
	LogPath        string
	ExePath        string
}

// GetLauncherPaths constructs launcher paths for a given WINEPREFIX and home directory
func GetLauncherPaths(wineprefix, homeDir string) LauncherPaths {
	return LauncherPaths{
		BinaryPath:     LauncherBinaryPath,
		DesktopPath:    launcherDesktopPath(homeDir),
		IconPath:       launcherIconPath(homeDir),
		LaunchVarsPath: launcherLaunchVarsPath(wineprefix),
		LogPath:        launcherLogPath(wineprefix),
		ExePath:        launcherExePath(wineprefix),
	}
}

// launcherDesktopPath returns the full path to the .desktop file
func launcherDesktopPath(homeDir string) string {
	return launcherPath(homeDir, UserApplicationsDir, LauncherDesktopAppName)
}

// launcherIconPath returns the full path to the launcher icon
func launcherIconPath(homeDir string) string {
	return launcherPath(homeDir, UserIconsDir, LauncherIconName)
}

// launcherLaunchVarsPath returns the full path to the launch_vars.env file
func launcherLaunchVarsPath(wineprefix string) string {
	return launcherPath(wineprefix, "", LauncherLaunchVarsFilename)
}

// launcherLogPath returns the full path to the launcher.log file
func launcherLogPath(wineprefix string) string {
	return launcherPath(wineprefix, "", LauncherLogFilename)
}

// launcherExePath returns the full path to the AstarteLauncher.exe inside WINEPREFIX
func launcherExePath(wineprefix string) string {
	return launcherPath(wineprefix, "", LauncherExeRelPath)
}

// launcherPath constructs a path from base, subDir, and filename
func launcherPath(base, subDir, filename string) string {
	if subDir != "" {
		return filepath.Join(base, subDir, filename)
	}
	return filepath.Join(base, filename)
}

// DesktopLauncherPath returns the full path to the desktop launcher shortcut
func DesktopLauncherPath(homeDir string) string {
	return launcherPath(homeDir, DesktopDir, LauncherDesktopAppName)
}
