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

// PrecheckResult holds the result of precheck validation
type PrecheckResult struct {
	WINEPREFIX        string
	WINEPREFIXSource  string
	UseExisting       bool
	ForceWineVersion  bool
	LauncherInstaller string
	GPUType           string
	IsAMDGPU          bool
	ProtonVer         string
	ProtonPath        string
	WinetricksPath    string
	WinetricksTmpDir  string
	WineVer           string
	WineBinPath       string
	EACRuntimePath    string
}

// ValidateWINEPREFIX validates the WINEPREFIX path
// Uses GUI directory picker if no argument is provided
// WINEPREFIX is set to the same value as INSTALL_DIR
func ValidateWINEPREFIX(wineprefixArg string, logger *core.Logger) (string, string, bool, error) {
	WINEPREFIX := ""
	WINEPREFIXSource := ""
	UseExisting := false

	if wineprefixArg != "" {
		// If user provides a path directly, use it as the WINEPREFIX
		WINEPREFIX = strings.TrimSuffix(wineprefixArg, "/")
		WINEPREFIXSource = "argument"
		logger.Info(fmt.Sprintf("WINEPREFIX: %s%s%s", core.ColorBoldYellow, WINEPREFIX, core.ColorReset))
	} else if envPrefix := os.Getenv("WINEPREFIX"); envPrefix != "" {
		// Legacy: WINEPREFIX env var directly
		WINEPREFIX = envPrefix
		WINEPREFIXSource = "environment variable (legacy)"
		logger.Info(fmt.Sprintf("WINEPREFIX is already set to: %s%s%s", core.ColorBoldYellow, WINEPREFIX, core.ColorReset))
		if core.AskBool("Do you want to use this path? (Y/n): ") {
			UseExisting = true
		} else {
			logger.Info("Enter the desired WINEPREFIX path (e.g. /path/to/wineprefix):")
			reader := core.NewReader()
			input, err := reader.ReadString('\n')
			if err != nil {
				return "", "", false, fmt.Errorf("failed to read WINEPREFIX: %w", err)
			}
			WINEPREFIX = strings.TrimSpace(input)
			WINEPREFIXSource = "user input"
		}
	} else if envDir := os.Getenv("INSTALL_DIR"); envDir != "" {
		// New: INSTALL_DIR env var - WINEPREFIX = INSTALL_DIR
		WINEPREFIX = strings.TrimSuffix(envDir, "/")
		WINEPREFIXSource = "environment variable (INSTALL_DIR)"
		logger.Info(fmt.Sprintf("INSTALL_DIR is set to: %s%s%s", core.ColorBoldYellow, WINEPREFIX, core.ColorReset))
		if core.AskBool("Do you want to use this path? (Y/n): ") {
			UseExisting = true
		} else {
			logger.Info("Enter the desired WINEPREFIX path (e.g. /path/to/wineprefix):")
			reader := core.NewReader()
			input, err := reader.ReadString('\n')
			if err != nil {
				return "", "", false, fmt.Errorf("failed to read WINEPREFIX: %w", err)
			}
			WINEPREFIX = strings.TrimSpace(input)
			WINEPREFIXSource = "user input"
		}
	} else {
		// Use GUI directory picker for INSTALL_DIR
		logger.Info("Select the directory where you want to install Bellum...")
		fmt.Println()

		result, err := gui.PickDirectory("")
		if err != nil {
			return "", "", false, fmt.Errorf("failed to pick directory: %w", err)
		}

		if !result.Success {
			return "", "", false, fmt.Errorf("directory selection cancelled or failed: %v", result.Error)
		}

		// WINEPREFIX is the same as INSTALL_DIR
		selectedPath := strings.TrimSuffix(result.Path, "/")
		WINEPREFIX = selectedPath
		WINEPREFIXSource = "GUI picker"

		logger.Info(fmt.Sprintf("WINEPREFIX: %s%s%s", core.ColorBoldYellow, WINEPREFIX, core.ColorReset))
	}

	// Normalize path
	WINEPREFIX = strings.TrimSuffix(WINEPREFIX, "/")

	// Validate absolute path
	if !strings.HasPrefix(WINEPREFIX, "/") {
		return "", "", false, fmt.Errorf("WINEPREFIX must be an absolute path (starting with /): %s", WINEPREFIX)
	}

	// Check parent directory exists and is writable
	WINEPREFIXParent := WINEPREFIX
	for !core.IsDir(WINEPREFIXParent) && WINEPREFIXParent != "/" {
		WINEPREFIXParent = filepath.Dir(WINEPREFIXParent)
	}

	if !core.IsDir(WINEPREFIXParent) {
		return "", "", false, fmt.Errorf("WINEPREFIX path is not on a valid mounted filesystem: %s", WINEPREFIX)
	}

	// Check if WINEPREFIX exists
	exists := core.IsDir(WINEPREFIX)
	wineprefixExists := false

	if exists {
		entries, err := os.ReadDir(WINEPREFIX)
		if err == nil && len(entries) != 0 {
			wineprefixExists = true
		}
	}

	if wineprefixExists {
		logger.Info(fmt.Sprintf("WINEPREFIX directory already exists at %s", WINEPREFIX))
		logger.Warn("If you want to reinstall, please uninstall the existing installation first.")
		return "", "", false, fmt.Errorf("WINEPREFIX directory '%s' already exists", WINEPREFIX)
	}

	if !core.IsWritable(WINEPREFIXParent) {
		return "", "", false, fmt.Errorf("WINEPREFIX parent directory is not writable: %s", WINEPREFIXParent)
	}

	logger.Info("[OK] WINEPREFIX path is valid and writable")

	// Check if SSD - use the parent directory (where Bellum is located)
	if isSSD(WINEPREFIXParent, logger) {
		logger.Info("[OK] WINEPREFIX device is an SSD/NVME (optimal performance)")
	} else {
		logger.Warn("WINEPREFIX device is NOT an SSD/NVME (may have performance issues)")
		if !core.AskBool("Astarte Developers strongly recommend using NVMe or SSD for the game. Are you sure you want to proceed? (Y/n): ") {
			return "", "", false, fmt.Errorf("installation cancelled by user")
		}
	}

	return WINEPREFIX, WINEPREFIXSource, UseExisting, nil
}

// SelectInstallDir prompts the user to select an installation directory using GUI picker
// Returns the selected install directory path (which is also the WINEPREFIX)
func SelectInstallDir(logger *core.Logger) (string, error) {
	fmt.Println()
	logger.Info("Select the directory where you want to install Bellum...")

	result, err := gui.PickDirectory("")
	if err != nil {
		return "", fmt.Errorf("failed to pick directory: %w", err)
	}

	if !result.Success {
		return "", fmt.Errorf("directory selection cancelled or failed: %v", result.Error)
	}

	selectedPath := result.Path
	logger.Info(fmt.Sprintf("Selected installation directory: %s", core.Colorize(selectedPath, core.ColorBoldYellow)))
	fmt.Println()

	// Validate the selected directory
	valid, errMsg := gui.ValidateDirectory(selectedPath, logger)
	if !valid {
		return "", fmt.Errorf("directory validation failed: %s", errMsg)
	}

	logger.Info("[OK] Directory validation passed")
	fmt.Println()

	// Create the directory if it doesn't exist
	if !core.IsDir(selectedPath) {
		logger.Info(fmt.Sprintf("Creating directory at %s...", selectedPath))
		if err := os.MkdirAll(selectedPath, 0755); err != nil {
			return "", fmt.Errorf("failed to create directory %s: %w", selectedPath, err)
		}
		logger.Info("[OK] Directory created successfully")
		fmt.Println()
	}

	return selectedPath, nil
}

// ValidateWINEPREFIXWithGUI prompts user to select a directory using GUI picker and validates it
// This function handles the complete workflow of:
// 1. Opening GUI directory picker
// 2. Validating the selected directory
// 3. Creating the directory
// Returns the WINEPREFIX path (which is the same as INSTALL_DIR)
func ValidateWINEPREFIXWithGUI(logger *core.Logger) (string, error) {
	// Open GUI directory picker
	fmt.Println()
	logger.Info("Select the directory where you want to install Bellum...")

	result, err := gui.PickDirectory("")
	if err != nil {
		return "", fmt.Errorf("failed to pick directory: %w", err)
	}

	if !result.Success {
		return "", fmt.Errorf("directory selection cancelled or failed: %v", result.Error)
	}

	selectedPath := result.Path
	logger.Info(fmt.Sprintf("Selected directory: %s", core.Colorize(selectedPath, core.ColorBoldYellow)))
	fmt.Println()

	// WINEPREFIX is the same as the selected directory
	wineprefixPath := selectedPath

	// Validate the selected directory
	valid, errMsg := gui.ValidateDirectory(wineprefixPath, logger)
	if !valid {
		return "", fmt.Errorf("directory validation failed: %s", errMsg)
	}

	logger.Info("[OK] Directory validation passed")
	fmt.Println()

	// Create the directory if it doesn't exist
	if !core.IsDir(wineprefixPath) {
		logger.Info(fmt.Sprintf("Creating directory at %s...", wineprefixPath))
		if err := os.MkdirAll(wineprefixPath, 0755); err != nil {
			return "", fmt.Errorf("failed to create directory %s: %w", wineprefixPath, err)
		}
		logger.Info("[OK] Directory created successfully")
		fmt.Println()
	}

	return wineprefixPath, nil
}

// CheckWinePackage ensures Wine package is downloaded and available
func CheckWinePackage(logger *core.Logger) (string, string, error) {
	wineVer := config.DefaultVersions.WineVer

	wineDir, err := packages.EnsureWine(wineVer, logger)
	if err != nil {
		return "", "", err
	}

	wineBinPath := packages.GetWineBinPath(wineDir)
	return wineVer, wineBinPath, nil
}

// CheckUMURun verifies umu-run is available
func CheckUMURun(logger *core.Logger) error {
	if core.LookPath("umu-run") == "" {
		return fmt.Errorf("umu-run not found")
	}

	logger.Info("[OK] umu-run binary found: " + core.LookPath("umu-run"))
	return nil
}

// CheckLauncherInstaller checks if launcher installer is available
func CheckLauncherInstaller(launcherInstallerPath string, logger *core.Logger) error {
	if launcherInstallerPath != "" {
		if _, err := os.Stat(launcherInstallerPath); os.IsNotExist(err) {
			return fmt.Errorf("launcher installer not found: %s", launcherInstallerPath)
		}
		logger.Info(fmt.Sprintf("[OK] Launcher installer found: %s", launcherInstallerPath))
		return nil
	}

	if core.LookPath("wget") == "" {
		return fmt.Errorf("launcher installer not provided and wget not available")
	}

	logger.Info("[OK] wget found for launcher installer download")
	return nil
}

// CheckWinetricks extracts winetricks to a temporary location and returns its path
func CheckWinetricks(workdir string, logger *core.Logger) (string, string, error) {
	logger.Info("Extracting winetricks from local archive...")

	winetricksArchive := filepath.Join(workdir, "packages", "winetricks-"+config.DefaultVersions.WinetricksVer+".tar.gz")
	if _, err := os.Stat(winetricksArchive); os.IsNotExist(err) {
		return "", "", fmt.Errorf("winetricks archive not found")
	}

	logger.Info(fmt.Sprintf("Extracting %s into packages/.tmp/winetricks/...", winetricksArchive))
	tmpDir, err := packages.ExtractPackage(winetricksArchive, "winetricks")
	if err != nil {
		return "", "", err
	}

	if !core.IsDir(tmpDir) {
		return "", "", fmt.Errorf("winetricks extraction failed: directory %s not found", tmpDir)
	}

	// Find the winetricks binary in the extracted directory (may be in src/ subdirectory)
	winetricksBinary := filepath.Join(tmpDir, "winetricks")
	if _, err := os.Stat(winetricksBinary); os.IsNotExist(err) {
		// Try src/ subdirectory
		winetricksBinary = filepath.Join(tmpDir, "src", "winetricks")
		if _, err := os.Stat(winetricksBinary); os.IsNotExist(err) {
			return "", "", fmt.Errorf("winetricks binary not found in extracted directory")
		}
	}
	core.RunCommand(core.RunModeSilent, []string{winetricksBinary, "--self-update"}, logger, "", nil, nil)
	logger.Info(fmt.Sprintf("Using winetricks from: %s", winetricksBinary))
	return winetricksBinary, tmpDir, nil
}

// CheckProton ensures Proton is available
func CheckProton(packageRoot string, gpuType string, isFSR41 bool, logger *core.Logger) (string, string, error) {
	isAMD := strings.Contains(strings.ToLower(gpuType), "amd") || strings.Contains(strings.ToLower(gpuType), "radeon")

	if core.LookPath("wget") == "" {
		return "", "", core.LogAndReturn(fmt.Errorf("proton missing and wget not available"), core.ErrorLevelCritical, logger)
	}
	logger.Info("[OK] wget found for Proton download")

	// Use proton-cachyos for all GPUs (AMD and NVIDIA)
	protonVer := config.DefaultVersions.ProtonVer
	_ = packages.GetProtonURL(protonVer, config.DefaultVersions.ProtonBaseURL)

	// Get the actual proton install path
	protonDir, err := packages.GetProtonInstallPath(protonVer)
	if err != nil {
		return "", "", err
	}

	if err := packages.EnsureProton(protonDir, protonVer, isAMD, isFSR41, logger); err != nil {
		return "", "", err
	}

	return protonVer, protonDir, nil
}

// DetectGPU detects the GPU type
func DetectGPU(logger *core.Logger) (string, error) {
	gpuType, err := core.DetectGPU()
	if err != nil {
		return "", core.LogAndReturn(fmt.Errorf("failed to detect GPU type: %w", err), core.ErrorLevelCritical, logger)
	}

	logger.Info(fmt.Sprintf("GPU Vendor: %s", gpuType))
	return gpuType, nil
}

// RunPrechecks runs all precheck validations
func RunPrechecks(wineprefixArg string, launcherInstallerPath string, forceWineVersion bool, fsr41 bool, logger *core.Logger) (*PrecheckResult, error) {
	logger.Info("Starting precheck phase...")
	fmt.Println()

	// Detect GPU type
	gpuType, err := DetectGPU(logger)
	if err != nil {
		return nil, err
	}

	isAMD := strings.Contains(strings.ToLower(gpuType), "amd") || strings.Contains(strings.ToLower(gpuType), "radeon")

	// Validate WINEPREFIX
	wineprefix, _, _, err := ValidateWINEPREFIX(wineprefixArg, logger)
	if err != nil {
		return nil, err
	}

	// Check Wine package (download and extract packaged Wine)
	wineVer, wineBinPath, err := CheckWinePackage(logger)
	if err != nil {
		return nil, err
	}

	// Check xdotool (optional - commented out per bash version)
	// if err := CheckXdotool(logger); err != nil {
	// 	return nil, err
	// }

	// Check umu-run
	if err := CheckUMURun(logger); err != nil {
		return nil, err
	}

	// Check launcher installer
	if err := CheckLauncherInstaller(launcherInstallerPath, logger); err != nil {
		return nil, err
	}

	// Get executable path and workdir
	exePath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("failed to get executable path: %w", err)
	}
	workdir := filepath.Dir(exePath)
	packageRoot := filepath.Join(workdir, "packages")
	if _, err := os.Stat(packageRoot); os.IsNotExist(err) {
		return nil, fmt.Errorf("packages directory not found: %s", packageRoot)
	}

	// Check winetricks
	winetricksPath, winetricksTmpDir, err := CheckWinetricks(workdir, logger)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(packageRoot); os.IsNotExist(err) {
		return nil, fmt.Errorf("packages directory not found: %s", packageRoot)
	}

	// Check Proton
	protonVer, protonPath, err := CheckProton(packageRoot, gpuType, fsr41, logger)
	if err != nil {
		return nil, err
	}

	// Check EAC Runtime
	eacRuntimePath, err := CheckEACRuntime(workdir, logger)
	if err != nil {
		return nil, err
	}

	logger.Info("[OK] All prechecks passed!")
	fmt.Println()

	return &PrecheckResult{
		WINEPREFIX:        wineprefix,
		GPUType:           gpuType,
		IsAMDGPU:          isAMD,
		ProtonVer:         protonVer,
		ProtonPath:        protonPath,
		ForceWineVersion:  forceWineVersion,
		LauncherInstaller: launcherInstallerPath,
		WinetricksPath:    winetricksPath,
		WinetricksTmpDir:  winetricksTmpDir,
		WineVer:           wineVer,
		WineBinPath:       wineBinPath,
		EACRuntimePath:    eacRuntimePath,
	}, nil
}

// CheckEACRuntime ensures EAC Runtime is installed
func CheckEACRuntime(workdir string, logger *core.Logger) (string, error) {
	if err := packages.EnsureEACRuntime(workdir, logger); err != nil {
		return "", err
	}

	path, err := packages.GetEACRuntimeInstallPath()
	if err != nil {
		return "", err
	}

	return path, nil
}

// Helper functions

// isSSD checks if the filesystem containing the given path is an SSD/NVMe device.
// Uses lsblk to check rotational status, falling back to device name detection.
func isSSD(path string, logger *core.Logger) bool {
	// Try lsblk first
	var output string
	if err := core.RunCommand(core.RunModeCapture, []string{"lsblk", "-no", "rota", filepath.Dir(path)}, nil, "", nil, &output); err == nil {
		rotational := strings.TrimSpace(output)
		return rotational == "0"
	}

	// Fallback to checking device name
	var device string
	if err := core.RunCommand(core.RunModeCapture, []string{"df", "-P", path}, nil, "", nil, &device); err != nil {
		return false
	}

	// Parse device name from df output
	lines := strings.Split(device, "\n")
	if len(lines) >= 2 {
		fields := strings.Fields(lines[1])
		if len(fields) >= 1 {
			deviceName := filepath.Base(fields[0])
			return strings.HasPrefix(deviceName, "nvme") || strings.HasPrefix(deviceName, "sd") || strings.HasPrefix(deviceName, "vd")
		}
	}

	return false
}
