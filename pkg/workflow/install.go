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

// validateEnvironmentVariables validates all environment variables before setting them
func validateEnvironmentVariables(wineprefix, protonpath string) error {
	if err := validateEnvVar("WINEPREFIX"); err != nil {
		return err
	}

	if err := validateEnvVar("PROTONPATH"); err != nil {
		return err
	}

	return nil
}

// validateEnvVar checks that a required environment variable is set.
func validateEnvVar(name string) error {
	if _, err := core.GetRequiredEnvVar(name); err != nil {
		return err
	}
	return nil
}

// InstallConfig holds configuration for installation
type InstallConfig struct {
	InstallDir         string
	WINEPREFIX         string
	ProtonPath         string
	GPUType            string
	IsAMDGPU           bool
	LauncherInstaller  string
	Workdir            string
	IsFSR41            bool
	LauncherBinaryPath string
	WinetricksPath     string
	WinetricksTmpDir   string
	UseProtonForAMD    bool
	WineBinPath        string
	EACRuntimePath     string
}

// InstallDXVK installs DXVK using the packaged setup script
func InstallDXVK(gpuType string, workdir string, logger *core.Logger) (string, error) {
	// Only install DXVK for AMD GPUs
	if !strings.Contains(strings.ToLower(gpuType), "amd") && !strings.Contains(strings.ToLower(gpuType), "radeon") {
		logger.Info(fmt.Sprintf("Skipping DXVK installation for non-AMD GPU: %s", gpuType))
		return "", nil
	}

	archive := filepath.Join(workdir, "packages", "dxvk-"+cfg.DefaultVersions.DXVKVer+".tar.gz")

	logger.Info("Installing DXVK...")
	if _, err := os.Stat(archive); os.IsNotExist(err) {
		return "", fmt.Errorf("DXVK archive not found: %s", archive)
	}

	tmpDir, err := packages.ExtractPackage(archive, "dxvk")
	if err != nil {
		return "", err
	}

	// Find the DXVK root directory (archive extracts with version prefix)
	installDir := tmpDir
	if _, err := os.Stat(filepath.Join(tmpDir, "dxvk_setup.sh")); os.IsNotExist(err) {
		entries, err := os.ReadDir(tmpDir)
		if err != nil || len(entries) == 0 {
			return "", fmt.Errorf("DXVK directory not found in archive")
		}
		for _, entry := range entries {
			if entry.IsDir() {
				if _, err := os.Stat(filepath.Join(tmpDir, entry.Name(), "dxvk_setup.sh")); err == nil {
					installDir = filepath.Join(tmpDir, entry.Name())
					break
				}
			}
		}
	}

	setupScript := filepath.Join(installDir, "dxvk_setup.sh")
	if _, err := os.Stat(setupScript); os.IsNotExist(err) {
		return "", fmt.Errorf("DXVK setup script not found at %s", setupScript)
	}

	wineprefix, err := core.GetRequiredEnvVar("WINEPREFIX")
	if err != nil {
		return "", err
	}

	// Validate WINEPREFIX
	if _, err := os.Stat(filepath.Join(wineprefix, "system.reg")); os.IsNotExist(err) {
		return "", fmt.Errorf("%s: Not a valid wine prefix", wineprefix)
	}

	// Make sure the setup script is executable
	if err := os.Chmod(setupScript, 0755); err != nil {
		return "", fmt.Errorf("failed to make dxvk_setup.sh executable: %w", err)
	}

	// Run dxvk_setup.sh with DXVK_HOME and WINEPREFIX set.
	// Start from os.Environ() so the scoped PATH/LD_LIBRARY_PATH from
	// ScopedWine.Apply() in main.go is inherited by the script.
	logFile := filepath.Join(workdir, "logs", "installer.log")
	dxvkEnv := append(os.Environ(),
		fmt.Sprintf("DXVK_HOME=%s", installDir),
		fmt.Sprintf("WINEPREFIX=%s", wineprefix),
	)

	if err := core.RunCommand(core.RunModeSilent, []string{setupScript, "install"}, logger, logFile, dxvkEnv, nil); err != nil {
		return "", fmt.Errorf("dxvk_setup.sh failed: %w", err)
	}

	logger.Info("[OK] DXVK installed")
	return tmpDir, nil
}

// RunInstaller runs the main installation workflow
func RunInstaller(config InstallConfig, logger *core.Logger) error {
	logger.Info("Starting Installation")
	fmt.Println()

	// Clean up winetricks temp directory at the end
	if config.WinetricksTmpDir != "" {
		defer func() {
			if err := os.RemoveAll(config.WinetricksTmpDir); err != nil {
				logger.Warn(fmt.Sprintf("Failed to clean up winetricks temp directory: %v", err))
			}
		}()
	}

	// Set environment variables first
	os.Setenv("PROTONPATH", config.ProtonPath)
	os.Setenv("INSTALL_DIR", config.InstallDir)
	os.Setenv("WINEPREFIX", config.WINEPREFIX)
	os.Setenv("WINEARCH", "win64")
	os.Setenv("STEAM_APP_PATH", config.WINEPREFIX)
	os.Setenv("STEAM_APPID", "1")
	os.Setenv("STEAM_COMPAT_DATA_PATH", config.WINEPREFIX)
	os.Setenv("STEAM_COMPAT_CLIENT_INSTALL_PATH", filepath.Join(os.Getenv("HOME"), ".steam", "steam"))
	os.Setenv("GAMEID", "1")
	if config.EACRuntimePath != "" {
		os.Setenv("PROTON_EAC_RUNTIME", config.EACRuntimePath)
	}

	// Validate environment variables are set
	if err := validateEnvironmentVariables(config.WINEPREFIX, config.ProtonPath); err != nil {
		return core.LogAndReturn(fmt.Errorf("invalid environment variable: %w", err), core.ErrorLevelCritical, logger)
	}

	// Get launcher installer path
	launcherInstaller := config.LauncherInstaller
	if launcherInstaller == "" {
		state, err := packages.DownloadLauncherInstaller(config.Workdir, logger)
		if err != nil {
			return core.LogAndReturn(fmt.Errorf("failed to download launcher installer: %w", err), core.ErrorLevelCritical, logger)
		}
		launcherInstaller = state.InstallerPath
		defer packages.CleanupLauncherInstaller(state, logger)
	}

	// Initialize WINEPREFIX with Proton base
	logger.Info("Initializing WINEPREFIX with Proton base")
	logFile := filepath.Join(config.Workdir, "logs", "installer.log")

	wineBin := filepath.Join(config.WineBinPath, "wine")
	winebootBin := filepath.Join(config.WineBinPath, "wineboot")

	if err := core.RunCommand(core.RunModeSilent, []string{"umu-run", winebootBin, "--init"}, logger, logFile, nil, nil); err != nil {
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
		winetricksArgs := []string{config.WinetricksPath, "-q", dll}
		if err := core.RunCommand(core.RunModeSilent, winetricksArgs, logger, logFile, nil, nil); err != nil {
			return err
		}
		logger.Info(fmt.Sprintf("[OK] %s", dll))
	}

	fmt.Println()
	logger.Info("Time to install the launcher! Follow the on screen prompts once the GUI pops up.")

	// Kill wine server before running installer
	core.RunCommand(core.RunModeSilent, []string{filepath.Join(config.WineBinPath, "wineserver"), "-k"}, logger, logFile, nil, nil)

	// Run the launcher installer
	proton := filepath.Join(config.ProtonPath, "proton")
	if err := core.RunCommand(core.RunModeSilent, []string{proton, "run", launcherInstaller}, logger, logFile, nil, nil); err != nil {
		return err
	}

	logger.Info("Astarte Launcher install completed successfully! Few more steps to go...")
	logger.Warn("I'm not done! Don't launch game or close this script just yet")

	// Set Windows 11
	if err := core.RunCommand(core.RunModeSilent, []string{config.WinetricksPath, "win11"}, logger, logFile, nil, nil); err != nil {
		logger.Warn("winetricks win11 failed (may be expected)")
	}

	fmt.Println()

	// Install DXVK (AMD only)
	var dxvkTmpDir string
	defer func() {
		if dxvkTmpDir != "" {
			packages.CleanupSpecificTempDir(dxvkTmpDir)
		}
	}()
	dxvkTmpDir, err := InstallDXVK(config.GPUType, config.Workdir, logger)
	if err != nil {
		return err
	}

	// Configure WINEPREFIX
	logger.Info("Configuring WINEPREFIX with things Bellum likes")
	if err := core.RunCommand(core.RunModeSilent, []string{config.WinetricksPath, "grabfullscreen=y", "windowmanagerdecorated=n", "mwo=disabled"}, logger, logFile, nil, nil); err != nil {
		return err
	}

	// Remove mono for AMD GPUs
	if config.IsAMDGPU {
		if err := core.RunCommand(core.RunModeSilent, []string{config.WinetricksPath, "remove_mono"}, logger, logFile, nil, nil); err != nil {
			return err
		}
	}

	// Generate launcher
	if err := GenerateLauncher(config, logger); err != nil {
		return err
	}

	// Set DLL overrides
	core.RunCommand(core.RunModeSilent, []string{wineBin, "reg", "add", `HKCU\Software\Wine\DirectInput`, "/v", "RawInput", "/t", "REG_DWORD", "/d", "1", "/f"}, logger, logFile, nil, nil)

	// End wine session
	core.RunCommand(core.RunModeSilent, []string{winebootBin, "--end-session"}, logger, logFile, nil, nil)

	// Clean up the packages/.tmp directory
	if config.Workdir != "" {
		packages.CleanupTempDir(config.Workdir)
	}

	return nil
}

// GenerateLauncher generates the launcher wrappers and desktop files
func GenerateLauncher(config InstallConfig, logger *core.Logger) error {
	// Copy icon to system location
	iconPath := filepath.Join(config.Workdir, "packages", "launcher_1_256x256x32.png")
	if err := launchers.CopyIcon(iconPath); err != nil {
		logger.Warn(err.Error())
	}

	// Generate launcher config
	launcherConfig := launchers.LauncherConfig{
		Wineprefix:         config.WINEPREFIX,
		Protonpath:         config.ProtonPath,
		GPUType:            config.GPUType,
		IconPath:           iconPath,
		LauncherBinaryPath: config.LauncherBinaryPath,
		UseProtonForAMD:    config.UseProtonForAMD,
		WineBinPath:        config.WineBinPath,
		EACRuntimePath:     config.EACRuntimePath,
	}

	if err := launchers.GenerateLauncher(launcherConfig, logger); err != nil {
		return err
	}

	logger.Info(fmt.Sprintf("[OK] Game launcher installed: %s", config.LauncherBinaryPath))

	return nil
}
