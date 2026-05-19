package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"bellum-installer/pkg/config"
	"bellum-installer/pkg/core"
	"bellum-installer/pkg/gui"
	"bellum-installer/pkg/packages"
)

// UninstallConfig holds configuration for uninstallation
type UninstallConfig struct {
	WINEPREFIX string
	GPUType    string
}

// RunUninstallation runs the uninstallation workflow
func RunUninstallation(config UninstallConfig, logger *core.Logger) error {

	logger.Info(fmt.Sprintf("GPU Type: %s", config.GPUType))
	logger.Info("Starting uninstallation phase...")
	fmt.Println()

	// Validate WINEPREFIX
	if config.WINEPREFIX == "" {
		return core.LogAndReturn(fmt.Errorf("WINEPREFIX is required. Use --wineprefix <path> or set WINEPREFIX environment variable."), core.ErrorLevelCritical, logger)
	}

	// Check if WINEPREFIX exists
	wineprefixExists := core.IsDir(config.WINEPREFIX)
	if !wineprefixExists {
		logger.Warn(fmt.Sprintf("WINEPREFIX directory not found: %s", core.Colorize(config.WINEPREFIX, core.ColorBoldYellow)))
	}

	logger.Info(fmt.Sprintf("WINEPREFIX: %s", core.Colorize(config.WINEPREFIX, core.ColorBoldYellow)))

	// Ask for confirmation before removing anything
	fmt.Println()
	if !core.AskBool("Are you sure you want to uninstall Bellum? This action cannot be undone. (Y/n): ") {
		logger.Info("Uninstallation cancelled by user.")
		return nil
	}

	fmt.Println()
	logger.Info("Proceeding with uninstallation...")
	fmt.Println()

	// Remove launcher binaries
	if err := removeLauncherBinaries(config.GPUType, logger); err != nil {
		return err
	}

	// Remove desktop entries
	if err := removeDesktopEntries(config.GPUType, logger); err != nil {
		return err
	}

	// Remove icon
	if err := removeIcon(logger); err != nil {
		return err
	}

	// Remove Proton directory
	if err := removeProton(config.WINEPREFIX, config.GPUType, logger); err != nil {
		return err
	}

	// Remove Wine package
	if err := removeWinePackage(logger); err != nil {
		return err
	}

	// Remove WINEPREFIX if it exists
	if wineprefixExists {
		if err := removeWINEPREFIX(config.WINEPREFIX, logger); err != nil {
			return err
		}
	} else {
		logger.Info(fmt.Sprintf("Skipping WINEPREFIX removal: %s does not exist", core.Colorize(config.WINEPREFIX, core.ColorBoldYellow)))
	}

	logger.Info("[OK] Uninstallation complete!")
	fmt.Println()

	return nil
}

// removeLauncherBinaries removes the launcher wrapper script.
// If writing to the launcher bin directory and not running as root, it uses pkexec to elevate privileges.
func removeLauncherBinaries(gpuType string, logger *core.Logger) error {
	logger.Info("Removing launcher binaries...")

	bellumPath := config.LauncherBinaryPath
	if _, err := os.Stat(bellumPath); err == nil {
		if err := os.Remove(bellumPath); err != nil {
			// If permission denied and path is under /usr/local/bin, try with pkexec
			if os.IsPermission(err) && strings.HasPrefix(bellumPath, config.LauncherBinDir+"/") {
				if removeWithSudo(bellumPath, logger) != nil {
					return fmt.Errorf("failed to remove %s: %w", bellumPath, err)
				}
				logger.Info(fmt.Sprintf("[OK] Removed %s", bellumPath))
				return nil
			}
			return fmt.Errorf("failed to remove %s: %w", bellumPath, err)
		}
		logger.Info(fmt.Sprintf("[OK] Removed %s", bellumPath))
	}

	return nil
}

// removeWithSudo uses pkexec to remove a file with elevated privileges.
func removeWithSudo(path string, logger *core.Logger) error {
	cmd := []string{"pkexec", "rm", path}
	if err := core.RunCommand(core.RunModeSilent, cmd, logger, "", nil, nil); err != nil {
		return fmt.Errorf("failed to remove %s with pkexec: %w", path, err)
	}
	return nil
}

// removeDesktopEntries removes the .desktop files.
func removeDesktopEntries(gpuType string, logger *core.Logger) error {
	logger.Info("Removing desktop entries...")

	homeDir := os.Getenv("HOME")
	paths := config.GetLauncherPaths("", homeDir)

	desktopPath := paths.DesktopPath
	if _, err := os.Stat(desktopPath); err == nil {
		if err := os.Remove(desktopPath); err != nil {
			logger.Warn(fmt.Sprintf("Failed to remove %s: %v", desktopPath, err))
		} else {
			logger.Info(fmt.Sprintf("[OK] Removed %s", desktopPath))
		}
	}

	desktopDest := config.DesktopLauncherPath(homeDir)
	if _, err := os.Stat(desktopDest); err == nil {
		if err := os.Remove(desktopDest); err != nil {
			logger.Warn(fmt.Sprintf("Failed to remove %s: %v", desktopDest, err))
		} else {
			logger.Info(fmt.Sprintf("[OK] Removed %s", desktopDest))
		}
	}

	userAppsDir := filepath.Join(homeDir, config.UserApplicationsDir)
	if _, err := os.Stat(userAppsDir); err == nil {
		core.RunCommand(core.RunModeSilent, []string{"update-desktop-database", userAppsDir}, logger, "", nil, nil)
	}

	return nil
}

// removeIcon removes the launcher icon from user-level icon directory
func removeIcon(logger *core.Logger) error {
	logger.Info("Removing launcher icon...")

	homeDir := os.Getenv("HOME")
	paths := config.GetLauncherPaths("", homeDir)
	iconPath := paths.IconPath
	if _, err := os.Stat(iconPath); err == nil {
		if err := os.Remove(iconPath); err != nil {
			logger.Warn(fmt.Sprintf("Failed to remove %s: %v", iconPath, err))
		} else {
			logger.Info(fmt.Sprintf("[OK] Removed %s", iconPath))
		}
	}

	return nil
}

// removeProton removes the Proton directory
func removeProton(wineprefix string, gpuType string, logger *core.Logger) error {
	logger.Info("Removing Proton directory...")

	// Use proton-cachyos for all GPUs (AMD and NVIDIA)
	protonVer := config.DefaultVersions.ProtonVer

	// Get the proton install path (even if wineprefix doesn't exist)
	protonPath, err := packages.GetProtonInstallPath(protonVer)
	if err != nil {
		logger.Warn(fmt.Sprintf("Failed to get proton install path: %v", err))
		return nil
	}
	if _, err := os.Stat(protonPath); err == nil {
		logger.Info(fmt.Sprintf("Removing Proton directory: %s", protonPath))
		if err := os.RemoveAll(protonPath); err != nil {
			return fmt.Errorf("failed to remove Proton directory %s: %w", protonPath, err)
		}
		logger.Info("[OK] Removed Bellum Proton directory")
	}

	// Check if the parent proton directory is now empty and remove it silently
	protonParentDir := filepath.Join(filepath.Dir(protonPath))
	if core.IsEmptyDir(protonParentDir) {
		os.RemoveAll(protonParentDir)
	}

	// Check if the bellum directory is now empty and remove it silently
	bellumDir := filepath.Join(filepath.Dir(filepath.Dir(protonParentDir)))
	if core.IsEmptyDir(bellumDir) {
		os.RemoveAll(bellumDir)
	}

	return nil
}

// removeWinePackage removes the packaged Wine directory
func removeWinePackage(logger *core.Logger) error {
	logger.Info("Removing Wine package...")

	wineVer := config.DefaultVersions.WineVer
	wineDir, err := packages.GetWineInstallPath(wineVer)
	if err != nil {
		logger.Warn(fmt.Sprintf("Failed to get Wine install path: %v", err))
		return nil
	}

	if _, err := os.Stat(wineDir); err == nil {
		logger.Info(fmt.Sprintf("Removing Wine directory: %s", wineDir))
		if err := os.RemoveAll(wineDir); err != nil {
			return fmt.Errorf("failed to remove Wine directory %s: %w", wineDir, err)
		}
		logger.Info("[OK] Removed Wine directory")
	}

	// Check if the bellum directory is now empty and remove it silently
	bellumDir := filepath.Join(filepath.Dir(wineDir))
	if core.IsEmptyDir(bellumDir) {
		os.RemoveAll(bellumDir)
	}

	return nil
}

// removeWINEPREFIX removes the WINEPREFIX directory with user confirmation
func removeWINEPREFIX(wineprefix string, logger *core.Logger) error {
	if err := os.RemoveAll(wineprefix); err != nil {
		return err
	}

	logger.Info(fmt.Sprintf("[OK] Removed WINEPREFIX: %s", core.Colorize(wineprefix, core.ColorBoldYellow)))
	return nil
}

// ValidateWINEPREFIXWithGUIForUninstall prompts user to select a WINEPREFIX using GUI picker
// This function handles the complete workflow of:
// 1. Opening GUI directory picker to select the Bellum WINEPREFIX directory directly
// 2. Validating the WINEPREFIX exists
// Returns the WINEPREFIX path (selected directory)
func ValidateWINEPREFIXWithGUIForUninstall(logger *core.Logger) (string, error) {
	// Open GUI directory picker for existing directory
	fmt.Println()
	logger.Info("Select the Bellum WINEPREFIX directory to uninstall...")
	logger.Info("This will uninstall Bellum from the selected location.")
	fmt.Println()

	result, err := gui.PickDirectoryExisting("")
	if err != nil {
		return "", fmt.Errorf("failed to pick directory: %w", err)
	}

	if !result.Success {
		return "", fmt.Errorf("directory selection cancelled or failed: %v", result.Error)
	}

	selectedPath := result.Path
	logger.Info(fmt.Sprintf("Selected directory: %s", core.Colorize(selectedPath, core.ColorBoldYellow)))
	fmt.Println()

	// Validate the WINEPREFIX exists
	if !core.IsDir(selectedPath) {
		return "", fmt.Errorf("selected directory does not exist: %s", selectedPath)
	}

	// Verify it's a valid Bellum installation by checking for typical files
	entries, err := os.ReadDir(selectedPath)
	if err != nil || len(entries) == 0 {
		logger.Warn(fmt.Sprintf("WINEPREFIX directory %s appears to be empty or invalid", selectedPath))
		if !core.AskBool("Are you sure you want to proceed with uninstallation? (Y/n): ") {
			return "", fmt.Errorf("uninstallation cancelled by user")
		}
	}

	logger.Info(fmt.Sprintf("[OK] WINEPREFIX found at %s", core.Colorize(selectedPath, core.ColorBoldYellow)))
	fmt.Println()

	return selectedPath, nil
}
