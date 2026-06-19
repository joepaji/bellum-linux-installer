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
	installDir := flag.String("install-dir", "", "Path to installation directory (optional if INSTALL_DIR env var is set). WINEPREFIX will be set to the same value as INSTALL_DIR")
	launcherInstaller := flag.String("launcher-installer", "", "Path to launcher installer executable")
	fsr41 := flag.Bool("fsr41", false, "Use FSR 4.1 upgrade path")
	useProtonForAMD := flag.Bool("proton", false, "Use Proton launch approach for AMD GPUs (same as NVIDIA)")
	help := flag.Bool("help", false, "Show help message")

	flag.Parse()

	if *help {
		fmt.Println("Bellum Linux Installer")
		fmt.Println()
		fmt.Println("Usage: bellum-installer [options]")
		fmt.Println()
		fmt.Println("Options:")
		fmt.Println("  --force-wine-version  Force Wine version check (not recommended)")
		fmt.Println("  --install-dir PATH    Path to installation directory (optional if INSTALL_DIR env var is set). WINEPREFIX will be set to the same value as INSTALL_DIR")
		fmt.Println("  --launcher-installer PATH  Path to launcher installer executable")
		fmt.Println("  --fsr41               Use FSR 4.1 upgrade path")
		fmt.Println("  --proton              Use Proton launch approach for AMD GPUs (same as NVIDIA)")
		fmt.Println("  --help                Show this help message")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  bellum-installer --install-dir /path/to/install")
		fmt.Println("  bellum-installer --install-dir /path/to/install --launcher-installer /path/to/launcher.exe")
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

	logger.Info("Bellum Linux Installer")

	// Determine INSTALL_DIR (user-facing path)
	var selectedInstallDir string
	if *installDir != "" {
		selectedInstallDir = *installDir
		logger.Info(fmt.Sprintf("INSTALL_DIR from flag: %s", core.Colorize(selectedInstallDir, core.ColorBoldYellow)))
	} else if envDir := os.Getenv("INSTALL_DIR"); envDir != "" {
		selectedInstallDir = envDir
		logger.Info(fmt.Sprintf("INSTALL_DIR from environment: %s", core.Colorize(selectedInstallDir, core.ColorBoldYellow)))
	} else if envPrefix := os.Getenv("WINEPREFIX"); envPrefix != "" {
		// Backward compatibility: if WINEPREFIX is set, derive INSTALL_DIR from it
		selectedInstallDir = envPrefix
		logger.Info(fmt.Sprintf("WINEPREFIX environment variable found (legacy), using as INSTALL_DIR: %s", core.Colorize(selectedInstallDir, core.ColorBoldYellow)))
	} else {
		// Use GUI-based directory selection
		selectedInstallDir, err = workflow.SelectInstallDir(logger)
		if err != nil {
			logger.Error(fmt.Sprintf("Installation directory selection failed: %v", err))
			os.Exit(1)
		}
	}

	// Resolve INSTALL_DIR to absolute path if needed
	if !filepath.IsAbs(selectedInstallDir) {
		absInstallDir, err := filepath.Abs(selectedInstallDir)
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to resolve INSTALL_DIR to absolute path: %v", err))
			os.Exit(1)
		}
		selectedInstallDir = absInstallDir
	}

	// WINEPREFIX is the same as INSTALL_DIR
	wineprefix := selectedInstallDir

	// Run prechecks with absolute paths
	result, err := workflow.RunPrechecks(wineprefix, *launcherInstaller, *forceWineVersion, *fsr41, logger)
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
	if !core.AskBool("Do you want to proceed with the installation? (Y/n): ") {
		fmt.Println("Installation cancelled.")
		os.Exit(0)
	}

	// Create InstallConfig
	installConfig := workflow.InstallConfig{
		InstallDir:         selectedInstallDir,
		WINEPREFIX:         result.WINEPREFIX,
		ProtonPath:         result.ProtonPath,
		GPUType:            result.GPUType,
		IsAMDGPU:           result.IsAMDGPU,
		LauncherInstaller:  result.LauncherInstaller,
		Workdir:            workdir,
		IsFSR41:            *fsr41,
		LauncherBinaryPath: config.LauncherBinaryPath,
		WinetricksPath:     result.WinetricksPath,
		UseProtonForAMD:    *useProtonForAMD,
		WineBinPath:        result.WineBinPath,
		EACRuntimePath:     result.EACRuntimePath,
	}

	// Run FSR4.1 upgrade DLL copy before installation if --fsr41 flag is passed
	if *fsr41 && result.IsAMDGPU {
		logger.Info("Preparing FSR 4.1 upgrade DLL...")
		if err := copyFSR41UpgradeDLL(workdir, logger); err != nil {
			logger.Error(fmt.Sprintf("Failed to copy FSR 4.1 upgrade DLL: %v", err))
			os.Exit(1)
		}
	}

	// Scope wine: prepend packaged wine bin to PATH and update LD_LIBRARY_PATH
	// so all wine commands during the installer lifecycle use the packaged version.
	scoped := core.NewScopedWine(result.WineBinPath)
	scoped.Apply()
	defer scoped.Restore()

	// Run installation
	logger.Info("Starting installation phase...")
	if err := workflow.RunInstaller(installConfig, logger); err != nil {
		logger.Error(fmt.Sprintf("Installation failed: %v", err))
		os.Exit(1)
	}

	// Run configuration
	configureConfig := workflow.ConfigureConfig{
		InstallDir:     selectedInstallDir,
		WINEPREFIX:     result.WINEPREFIX,
		ProtonPath:     result.ProtonPath,
		GPUType:        result.GPUType,
		IsAMDGPU:       result.IsAMDGPU,
		Workdir:        workdir,
		IsFSR41:        *fsr41,
		WineBinPath:    result.WineBinPath,
		EACRuntimePath: result.EACRuntimePath,
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
	fmt.Printf("%s", core.Colorize(" - Desktop Shortcut (Recommended)\n", core.Bold))
	fmt.Println(" - Applications Menu -> Games -> Bellum")
	fmt.Println(" - Terminal Command: `Bellum`")
	fmt.Println()
	fmt.Printf("Launch Environment Variable File: %s/launch_vars.env\n", configureConfig.WINEPREFIX)
	fmt.Printf("Installation directory: %s\n", configureConfig.InstallDir)
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
=======================================================================================`
	fmt.Printf("%s%s%s\n", core.ColorBoldBlue, banner, core.ColorReset)
	fmt.Println()
}
