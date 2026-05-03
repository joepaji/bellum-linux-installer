// Package main provides the entry point for the Bellum installer.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"bellum-installer/pkg/config"
	"bellum-installer/pkg/core"
	"bellum-installer/pkg/workflow"
)

func main() {
	// Print banner
	printInstallerBanner()

	// Parse command line arguments
	forceWineVersion := flag.Bool("force-wine-version", false, "Force Wine version check")
	wineprefix := flag.String("wineprefix", "", "Path to WINEPREFIX directory (optional if WINEPREFIX env var is set)")
	launcherInstaller := flag.String("launcher-installer", "", "Path to launcher installer executable")
	fsr41 := flag.Bool("fsr41", false, "Use FSR 4.1 upgrade path")
	help := flag.Bool("help", false, "Show help message")

	flag.Parse()

	if *help {
		fmt.Println("Bellum Linux Installer - Go Edition")
		fmt.Println()
		fmt.Println("Usage: bellum-installer [options]")
		fmt.Println()
		fmt.Println("Options:")
		fmt.Println("  --force-wine-version  Force Wine version check (not recommended)")
		fmt.Println("  --wineprefix PATH     Path to WINEPREFIX directory (optional if WINEPREFIX env var is set)")
		fmt.Println("  --launcher-installer PATH  Path to launcher installer executable")
		fmt.Println("  --fsr41               Use FSR 4.1 upgrade path")
		fmt.Println("  --help                Show this help message")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  bellum-installer --wineprefix /path/to/wineprefix")
		fmt.Println("  bellum-installer --wineprefix /path/to/wineprefix --launcher-installer /path/to/launcher.exe")
		os.Exit(0)
	}

	// Determine workdir (directory containing the binary)
	exePath, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to get executable path: %v\n", err)
		os.Exit(1)
	}
	workdir := filepath.Dir(exePath)

	// Create log directory and file
	logDir := filepath.Join(workdir, "logs")
	logFile := filepath.Join(logDir, "installer.log")

	// Initialize logger
	logger, err := core.NewLogger(logFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Close()

	logger.Info("Bellum Linux Installer - Go Edition")

	// Determine WINEPREFIX
	var selectedWINEPREFIX string
	if *wineprefix != "" {
		selectedWINEPREFIX = *wineprefix
		logger.Info(fmt.Sprintf("WINEPREFIX from flag: %s", core.Colorize(selectedWINEPREFIX, core.ColorBoldYellow)))
	} else if envPrefix := os.Getenv("WINEPREFIX"); envPrefix != "" {
		selectedWINEPREFIX = envPrefix
		logger.Info(fmt.Sprintf("WINEPREFIX from environment: %s", core.Colorize(selectedWINEPREFIX, core.ColorBoldYellow)))
	} else {
		// Use GUI-based WINEPREFIX selection
		selectedWINEPREFIX, err = workflow.ValidateWINEPREFIXWithGUI(logger)
		if err != nil {
			logger.Error(fmt.Sprintf("WINEPREFIX selection failed: %v", err))
			os.Exit(1)
		}
	}

	// Resolve WINEPREFIX to absolute path if needed
	if !filepath.IsAbs(selectedWINEPREFIX) {
		absWINEPREFIX, err := filepath.Abs(selectedWINEPREFIX)
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to resolve WINEPREFIX to absolute path: %v", err))
			os.Exit(1)
		}
		selectedWINEPREFIX = absWINEPREFIX
	}

	// Run prechecks with absolute paths
	result, err := workflow.RunPrechecks(selectedWINEPREFIX, *launcherInstaller, *forceWineVersion, *fsr41, logger)
	if err != nil {
		logger.Error(fmt.Sprintf("Prechecks failed: %v", err))
		os.Exit(1)
	}

	// Print installer summary
	core.PrintInstallerSummary(
		result.ProtonVer,
		config.DefaultVersions.WineVer,
		config.DefaultVersions.WinetricksVer,
		config.DefaultVersions.VKD3DVer,
		config.DefaultVersions.DXVKVer,
		result.WINEPREFIX,
		result.LauncherInstaller,
		result.GPUType,
		workdir,
	)

	// Confirm with user before proceeding
	if !core.ConfirmProceed() {
		fmt.Println("Installation cancelled.")
		os.Exit(0)
	}

	// Create InstallConfig
	installConfig := workflow.InstallConfig{
		WINEPREFIX:        result.WINEPREFIX,
		ProtonPath:        result.ProtonPath,
		GPUType:           result.GPUType,
		IsAMDGPU:          result.IsAMDGPU,
		LauncherInstaller: result.LauncherInstaller,
		Workdir:           workdir,
		IsFSR41:           *fsr41,
	}

	// Run FSR4.1 upgrade DLL copy before installation if --fsr41 flag is passed
	if *fsr41 && result.IsAMDGPU {
		logger.Info("Preparing FSR 4.1 upgrade DLL...")
		if err := copyFSR41UpgradeDLL(workdir, logger); err != nil {
			logger.Error(fmt.Sprintf("Failed to copy FSR 4.1 upgrade DLL: %v", err))
			os.Exit(1)
		}
	}

	// Run installation
	logger.Info("Starting installation phase...")
	if err := workflow.RunInstaller(installConfig, logger); err != nil {
		logger.Error(fmt.Sprintf("Installation failed: %v", err))
		os.Exit(1)
	}

	// Run configuration
	configureConfig := workflow.ConfigureConfig{
		WINEPREFIX: result.WINEPREFIX,
		ProtonPath: result.ProtonPath,
		GPUType:    result.GPUType,
		IsAMDGPU:   result.IsAMDGPU,
		Workdir:    workdir,
		IsFSR41:    *fsr41,
	}

	if err := workflow.RunConfiguration(configureConfig, logger); err != nil {
		logger.Error(fmt.Sprintf("Configuration failed: %v", err))
		os.Exit(1)
	}

	logger.Info("Installation complete!")
	fmt.Println()
	fmt.Printf("%sInstallation completed successfully!%s\n", core.ColorBoldGreen, core.ColorReset)
	fmt.Println()
	fmt.Println("You can now launch Bellum using the 'Bellum' using any of these:")
	fmt.Printf("%s", core.Colorize("- Desktop Shortcut (Recommended)\n", core.Bold))
	fmt.Println(" - Applications Menu -> Games -> Bellum")
	fmt.Println(" - Terminal Command: `Bellum`")
	fmt.Println()
	fmt.Printf("Launch Environment Variable File: %s/launch_vars.env\n", configureConfig.WINEPREFIX)
	fmt.Println()
}

// copyFSR41UpgradeDLL copies the FSR4.1 upgrade DLL to the protonfixes upscalers directory
func copyFSR41UpgradeDLL(workdir string, logger *core.Logger) error {
	fsPath := filepath.Join(workdir, "packages", "fsr4")

	// Source DLL path
	sourceDLL := filepath.Join(fsPath, "amdxcffx64.dll")

	// Target directory: ${HOME}/.cache/protonfixes/upscalers/
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	targetDir := filepath.Join(homeDir, ".cache", "protonfixes", "upscalers")

	// Create target directory if it doesn't exist
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	// Target file path with version suffix
	targetDLL := filepath.Join(targetDir, "amdxcffx64_v4.1.0_69A0952A304a000.dll")

	// Read source file
	source, err := os.Open(sourceDLL)
	if err != nil {
		return fmt.Errorf("failed to open source DLL: %w", err)
	}
	defer source.Close()

	// Create target file
	target, err := os.Create(targetDLL)
	if err != nil {
		return fmt.Errorf("failed to create target DLL: %w", err)
	}
	defer target.Close()

	// Copy file content
	if _, err := io.Copy(target, source); err != nil {
		return fmt.Errorf("failed to copy DLL: %w", err)
	}

	logger.Info(fmt.Sprintf("[OK] Copied FSR 4.1.0 DLL to %s", targetDLL))
	return nil
}

func printInstallerBanner() {
	banner := `=======================================================================================
|                     Linux Wine-Proton Installer for Bellum                          |
======================================================================================`
	fmt.Printf("%s%s%s\n", core.ColorBoldBlue, banner, core.ColorReset)
	fmt.Println()
}
