package workflow

import (
	"fmt"
	"os"
	"path/filepath"

	cfg "bellum-installer/pkg/config"
	"bellum-installer/pkg/core"
	"bellum-installer/pkg/launchers"
)

// ConfigureConfig holds configuration for post-install configuration
type ConfigureConfig struct {
	WINEPREFIX string
	ProtonPath string
	GPUType    string
	IsAMDGPU   bool
	Workdir    string
	IsFSR41    bool
}

// RunConfiguration runs the post-install configuration phase
func RunConfiguration(config ConfigureConfig, logger *core.Logger) error {
	fmt.Println()
	logger.Info("Starting configuration phase...")

	// Update DLL overrides
	if err := UpdateDLLs(logger); err != nil {
		return err
	}

	// GPU-specific configuration
	if config.GPUType == "NVIDIA" {
		if err := launchers.CreateLaunchVarsFileNvidia(config.WINEPREFIX, config.ProtonPath, logger); err != nil {
			return err
		}
		// dxvk_nvapi is included in Proton for NVIDIA
	} else if config.GPUType == "AMD" {
		// Only run UpgradeFSR when --fsr41 flag is passed
		if config.IsFSR41 {
			if err := UpgradeFSR(config, logger); err != nil {
				return err
			}
		}
		if err := launchers.CreateLaunchVarsFileAMD(config.WINEPREFIX, config.ProtonPath, logger); err != nil {
			return err
		}
	} else {
		logger.Warn(fmt.Sprintf("Unknown or unsupported GPU type: %s", config.GPUType))
	}

	logger.Info("[OK] Configuration phase complete!")
	fmt.Println()

	return nil
}

// UpdateDLLs sets up DLL overrides for the WINEPREFIX
func UpdateDLLs(logger *core.Logger) error {
	logger.Info("Setting DLL overrides")

	wine := cfg.DefaultVersions.Binaries.Wine
	logFile := ""

	// System-wide overrides
	systemDLLs := []string{
		"d3d12",
		"d3d12core",
		"d3d10core",
		"d3d9",
		"d3d8",
	}

	for _, dll := range systemDLLs {
		if err := core.RunCommand(core.RunModeSilent, []string{wine, "reg", "add", `HKEY_CURRENT_USER\Software\Wine\DllOverrides`, "/v", dll, "/d", "native,builtin", "/f"}, logger, logFile); err != nil {
			return err
		}
	}

	// Application-specific overrides for d3d11 and dxgi
	appDLLs := []string{"d3d11", "dxgi"}

	for _, dll := range appDLLs {
		// Launcher overrides
		if err := core.RunCommand(core.RunModeSilent, []string{wine, "reg", "add", `HKCU\Software\Wine\AppDefaults\AstarteLauncher.exe\DllOverrides`, "/v", dll, "/d", "builtin", "/f"}, logger, logFile); err != nil {
			return err
		}

		// Game overrides
		if err := core.RunCommand(core.RunModeSilent, []string{wine, "reg", "add", `HKCU\Software\Wine\AppDefaults\Bellum-Win64-Shipping.exe\DllOverrides`, "/v", dll, "/d", "native", "/f"}, logger, logFile); err != nil {
			return err
		}
	}

	return nil
}

// UpgradeFSR upgrades to FSR 4.1.0 for AMD GPUs
func UpgradeFSR(config ConfigureConfig, logger *core.Logger) error {
	logger.Info("Upgrading to FSR 4.1.0")

	fsPath := filepath.Join(config.Workdir, "packages", "fsr4")
	if _, err := os.Stat(fsPath); os.IsNotExist(err) {
		logger.Warn(fmt.Sprintf("FSR 4.1.0 directory not found: %s, skipping upgrade", fsPath))
		return nil
	}

	fgDLL := "amd_fidelityfx_framegeneration_dx12.dll"
	d3DLL := "D3D12Core.dll"

	// Determine target directories
	progFiles := `Program Files`
	winePrefix := core.GetEnvVarOrDefault("WINEPREFIX", config.WINEPREFIX)

	fgTargetDir := filepath.Join(winePrefix, "drive_c", progFiles, "Astarte Industries", "Bellum", "Project_Bellum", "Plugins", "AMD", "FSR", "Source", "fidelityfx-sdk", "Kits", "FidelityFX", "signedbin")
	d3dTargetDir := filepath.Join(winePrefix, "drive_c", progFiles, "Astarte Industries", "Bellum", "Project_Bellum", "Binaries", "Win64", "D3D12", "x64")

	fgTarget := filepath.Join(fgTargetDir, fgDLL)
	d3dTarget := filepath.Join(d3dTargetDir, d3DLL)

	fgSource := filepath.Join(fsPath, fgDLL)
	d3dSource := filepath.Join(fsPath, d3DLL)

	// Get log file path
	logFile := filepath.Join(config.Workdir, "logs", "installer.log")

	// Create target directories
	core.RunCommand(core.RunModeSilent, []string{"mkdir", "-p", fgTargetDir, d3dTargetDir}, logger, logFile)

	// Copy FSR DLLs
	if _, err := os.Stat(fgSource); err == nil {
		if err := copyFile(fgSource, fgTarget); err != nil {
			logger.Warn(fmt.Sprintf("Failed to copy %s -> %s: %v", fgSource, fgTarget, err))
		} else {
			logger.Info(fmt.Sprintf("[OK] Copied %s", fgDLL))
		}
	}

	if _, err := os.Stat(d3dSource); err == nil {
		if err := copyFile(d3dSource, d3dTarget); err != nil {
			logger.Warn(fmt.Sprintf("Failed to copy %s -> %s: %v", d3dSource, d3dTarget, err))
		} else {
			logger.Info(fmt.Sprintf("[OK] Copied %s", d3DLL))
		}
	}

	// Register DLL override for amdxcffx64
	core.RunCommand(core.RunModeSilent, []string{cfg.DefaultVersions.Binaries.Wine, "reg", "add", `HKEY_CURRENT_USER\Software\Wine\DllOverrides`, "/v", "amdxcffx64", "/d", "native", "/f"}, logger, "")

	logger.Info("FSR 4.1.0 Upgrade Complete!")
	return nil
}

// copyFile copies a file from src to dst using the core utility.
func copyFile(src, dst string) error {
	return core.CopyFile(src, dst)
}
