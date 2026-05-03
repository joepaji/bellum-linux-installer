package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	cfg "bellum-installer/pkg/config"
	"bellum-installer/pkg/core"
	"bellum-installer/pkg/launchers"
	"bellum-installer/pkg/packages"
)

// InstallConfig holds configuration for installation
type InstallConfig struct {
	WINEPREFIX        string
	ProtonPath        string
	GPUType           string
	IsAMDGPU          bool
	LauncherInstaller string
	Workdir           string
	IsFSR41           bool
}

// InstallDXVK installs DXVK for AMD GPUs
func InstallDXVK(gpuType string, workdir string, logger *core.Logger) error {
	// Only install DXVK for AMD GPUs
	if !strings.Contains(strings.ToLower(gpuType), "amd") && !strings.Contains(strings.ToLower(gpuType), "radeon") {
		logger.Info(fmt.Sprintf("Skipping DXVK installation for non-AMD GPU: %s", gpuType))
		return nil
	}

	archive := filepath.Join(workdir, "packages", "dxvk-"+cfg.DefaultVersions.DXVKVer+".tar.gz")
	var tmpDir string

	logger.Info("Installing DXVK...")
	if _, err := os.Stat(archive); os.IsNotExist(err) {
		logger.Error(fmt.Sprintf("DXVK archive not found: %s", archive))
		return fmt.Errorf("DXVK archive not found: %s", archive)
	}

	tmpDir, err := packages.ExtractPackage(archive, "dxvk")
	if err != nil {
		logger.Error("Failed to extract DXVK archive")
		return err
	}

	// Find the dxvk_setup.sh script
	installDir := tmpDir
	if _, err := os.Stat(filepath.Join(tmpDir, "dxvk_setup.sh")); os.IsNotExist(err) {
		// Try to find subdirectory
		entries, err := os.ReadDir(tmpDir)
		if err != nil || len(entries) == 0 {
			logger.Error("DXVK setup script not found after extraction.")
			packages.CleanupTempDir(archive)
			return fmt.Errorf("DXVK setup script not found")
		}
		for _, entry := range entries {
			if entry.IsDir() {
				installDir = filepath.Join(tmpDir, entry.Name())
				break
			}
		}
	}

	if _, err := os.Stat(filepath.Join(installDir, "dxvk_setup.sh")); os.IsNotExist(err) {
		logger.Error("DXVK setup script not found after extraction.")
		packages.CleanupTempDir(archive)
		return fmt.Errorf("DXVK setup script not found")
	}

	// Run dxvk_setup.sh install
	logFile := filepath.Join(workdir, "logs", "installer.log")
	if err := core.RunCommand(core.RunModeSilent, []string{filepath.Join(installDir, "dxvk_setup.sh"), "install"}, logger, logFile); err != nil {
		logger.Error("DXVK installation failed.")
		packages.CleanupTempDir(archive)
		return err
	}

	// Copy dxvk.conf to WINEPREFIX
	dxvkConf := filepath.Join(installDir, "dxvk.conf")
	wineprefix := os.Getenv("WINEPREFIX")
	if wineprefix == "" {
		logger.Error("WINEPREFIX not set")
		packages.CleanupTempDir(archive)
		return fmt.Errorf("WINEPREFIX not set")
	}

	if err := copyFile(dxvkConf, filepath.Join(wineprefix, "dxvk.conf")); err != nil {
		logger.Error("Failed to copy dxvk.conf.")
		packages.CleanupTempDir(archive)
		return err
	}

	packages.CleanupTempDir(archive)
	logger.Info("[OK] DXVK installed")
	return nil
}

// RunInstaller runs the main installation workflow
func RunInstaller(config InstallConfig, logger *core.Logger) error {
	logger.Info("Starting Installation")
	fmt.Println()

	// Set environment variables
	os.Setenv("PROTONPATH", config.ProtonPath)
	os.Setenv("WINEPREFIX", config.WINEPREFIX)
	os.Setenv("WINEARCH", "win64")
	os.Setenv("STEAM_APP_PATH", config.WINEPREFIX)
	os.Setenv("STEAM_APPID", "1")
	os.Setenv("STEAM_COMPAT_DATA_PATH", config.WINEPREFIX)
	os.Setenv("STEAM_COMPAT_CLIENT_INSTALL_PATH", filepath.Join(os.Getenv("HOME"), ".steam", "steam"))
	os.Setenv("GAMEID", "1")

	// Get launcher installer path
	launcherInstaller := config.LauncherInstaller
	if launcherInstaller == "" {
		state, err := packages.DownloadLauncherInstaller(config.Workdir, logger)
		if err != nil {
			logger.Error("Failed to download launcher installer")
			return err
		}
		launcherInstaller = state.InstallerPath
		defer packages.CleanupLauncherInstaller(state, logger)
	}

	// Initialize WINEPREFIX with Proton base
	logger.Info("Initializing WINEPREFIX with Proton base")
	logFile := filepath.Join(config.Workdir, "logs", "installer.log")
	if err := core.RunCommand(core.RunModeSilent, []string{"umu-run", cfg.DefaultVersions.Binaries.Msidb}, logger, logFile); err != nil {
		logger.Warn("umu-run /usr/bin/msidb failed")
		return err
	}

	if err := core.RunCommand(core.RunModeSilent, []string{cfg.DefaultVersions.Binaries.Wineboot, "--init"}, logger, logFile); err != nil {
		logger.Error("wineboot --init failed")
		return err
	}

	// Install required winedlls
	logger.Info("Installing required winedlls")
	dlls := []string{
		"vcrun2026",
		"d3dcompiler_43",
		"d3dcompiler_47",
		"faudio",
		"msls31",
		"dotnet9",
		"dotnetdesktop9",
		"mfc140",
	}

	for _, dll := range dlls {
		if err := core.RunCommand(core.RunModeSilent, []string{"winetricks", "-q", dll}, logger, logFile); err != nil {
			logger.Error(fmt.Sprintf("Failed to install %s", dll))
			return err
		}
		logger.Info(fmt.Sprintf("[OK] %s", dll))
	}

	fmt.Println()
	logger.Info("Time to install the launcher! Follow the on screen prompts once the GUI pops up.")

	// Kill wine server before running installer
	core.RunCommand(core.RunModeSilent, []string{"wineserver", "-k"}, logger, logFile)

	// Run the launcher installer
	proton := filepath.Join(config.ProtonPath, "proton")
	if err := core.RunCommand(core.RunModeSilent, []string{proton, "run", launcherInstaller}, logger, logFile); err != nil {
		logger.Error("Launcher installation failed.")
		return err
	}

	logger.Info("Astarte Launcher install completed successfully! Few more steps to go...")
	logger.Warn("I'm not done! Don't launch game or close this script just yet")

	// Set Windows 11
	if err := core.RunCommand(core.RunModeSilent, []string{"winetricks", "win11"}, logger, logFile); err != nil {
		logger.Warn("winetricks win11 failed (may be expected)")
	}

	fmt.Println()

	// Install DXVK (AMD only)
	if err := InstallDXVK(config.GPUType, config.Workdir, logger); err != nil {
		return err
	}

	// Configure WINEPREFIX
	logger.Info("Configuring WINEPREFIX with things Bellum likes")
	if err := core.RunCommand(core.RunModeSilent, []string{"winetricks", "grabfullscreen=y", "windowmanagerdecorated=n", "mwo=disabled"}, logger, logFile); err != nil {
		logger.Error("Winetricks configuration failed.")
		return err
	}

	// Remove mono for AMD GPUs
	if config.IsAMDGPU {
		if err := core.RunCommand(core.RunModeSilent, []string{"winetricks", "remove_mono"}, logger, logFile); err != nil {
			logger.Error("Mono removal failed.")
			return err
		}
	}

	// Generate launcher
	if err := GenerateLauncher(config, logger); err != nil {
		return err
	}

	// Set DLL overrides
	core.RunCommand(core.RunModeSilent, []string{"wine", "reg", "add", `HKCU\Software\Wine\DirectInput`, "/v", "RawInput", "/t", "REG_DWORD", "/d", "1", "/f"}, logger, logFile)

	// End wine session
	core.RunCommand(core.RunModeSilent, []string{"wineboot", "--end-session"}, logger, logFile)

	return nil
}

// GenerateLauncher generates the launcher wrappers and desktop files
func GenerateLauncher(config InstallConfig, logger *core.Logger) error {
	// Copy icon to system location
	iconPath := filepath.Join(config.Workdir, "packages", "launcher_1_256x256x32.png")
	if err := launchers.CopyIcon(iconPath); err != nil {
		logger.Warn(fmt.Sprintf("Failed to copy icon: %v", err))
	}

	// Generate launcher config
	launcherConfig := launchers.LauncherConfig{
		Wineprefix: config.WINEPREFIX,
		Protonpath: config.ProtonPath,
		GPUType:    config.GPUType,
		IconPath:   iconPath,
	}

	if err := launchers.GenerateLauncher(launcherConfig); err != nil {
		logger.Error(fmt.Sprintf("Failed to generate launcher: %v", err))
		return err
	}

	logger.Info("[OK] Game launcher installed: /usr/local/bin/Bellum")

	// Generate launch vars file
	if config.GPUType == "NVIDIA" {
		if err := launchers.GenerateLaunchVarsFileNvidia(config.WINEPREFIX); err != nil {
			logger.Warn(fmt.Sprintf("Failed to generate NVIDIA launch vars: %v", err))
		}
	} else if config.GPUType == "AMD" {
		if err := launchers.GenerateLaunchVarsAMD(config.WINEPREFIX, config.IsFSR41); err != nil {
			logger.Warn(fmt.Sprintf("Failed to generate AMD launch vars: %v", err))
		}
	}

	return nil
}
