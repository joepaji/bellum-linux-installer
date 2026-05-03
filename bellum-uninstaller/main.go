// Package main provides the entry point for the Bellum uninstaller.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"bellum-installer/pkg/core"
	"bellum-installer/pkg/workflow"
)

func main() {
	// Print banner
	printUninstallerBanner()

	// Parse command line arguments
	wineprefix := flag.String("wineprefix", "", "Path to WINEPREFIX directory (optional if WINEPREFIX env var is set)")
	help := flag.Bool("help", false, "Show help message")

	flag.Parse()

	if *help {
		fmt.Println("Bellum Linux Uninstaller")
		fmt.Println()
		fmt.Println("Usage: bellum-uninstaller [options]")
		fmt.Println()
		fmt.Println("Options:")
		fmt.Println("  --wineprefix PATH  Path to WINEPREFIX directory (optional if WINEPREFIX env var is set)")
		fmt.Println("  --help             Show this help message")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  bellum-uninstaller --wineprefix /path/to/wineprefix")
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
	logFile := filepath.Join(logDir, "uninstaller.log")

	// Initialize logger
	logger, err := core.NewLogger(logFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Close()

	logger.Info("Bellum Linux Uninstaller")
	fmt.Println()

	// Resolve WINEPREFIX from env var or GUI picker if not provided via flag
	if *wineprefix == "" {
		if envPrefix := os.Getenv("WINEPREFIX"); envPrefix != "" {
			*wineprefix = envPrefix
		} else {
			// Use GUI-based WINEPREFIX selection
			selectedWINEPREFIX, err := workflow.ValidateWINEPREFIXWithGUIForUninstall(logger)
			if err != nil {
				logger.Error(fmt.Sprintf("WINEPREFIX selection failed: %v", err))
				os.Exit(1)
			}
			*wineprefix = selectedWINEPREFIX
		}
	}

	// Validate WINEPREFIX (should already be set from env var if not provided via flag)
	if *wineprefix == "" {
		logger.Error("WINEPREFIX is required. Use --wineprefix <path> or set WINEPREFIX environment variable.")
		os.Exit(1)
	}

	// Detect GPU type (needed for removing NVIDIA-specific files)
	gpuType, err := core.DetectGPU()
	if err != nil {
		logger.Warn(fmt.Sprintf("Failed to detect GPU type: %v, proceeding with generic cleanup", err))
		gpuType = "Unknown"
	}

	// Create UninstallConfig
	uninstallConfig := workflow.UninstallConfig{
		WINEPREFIX: *wineprefix,
		GPUType:    gpuType,
	}

	// Run uninstallation
	if err := workflow.RunUninstallation(uninstallConfig, logger); err != nil {
		logger.Error(fmt.Sprintf("Uninstallation failed: %v", err))
		os.Exit(1)
	}
}

func printUninstallerBanner() {
	banner := `=======================================================================================
|                     Linux Wine-Proton Uninstaller for Bellum                 	      |
=======================================================================================`
	fmt.Printf("%s%s%s\n", core.ColorBoldBlue, banner, core.ColorReset)
	fmt.Println()
}
