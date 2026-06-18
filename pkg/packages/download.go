package packages

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"bellum-installer/pkg/config"
	"bellum-installer/pkg/core"

	"github.com/ulikunitz/xz"
)

const launcherInstallerURL = "https://auto-updater.astarte.industries/astartelauncher/windows-amd64/AstarteLauncher-amd64-installer.exe"

// LauncherInstallerState tracks the state of the launcher installer download
type LauncherInstallerState struct {
	InstallerPath string
	Downloaded    bool
	DownloadDir   string
}

// GetProtonURL returns the download URL for AMD/CachyOS Proton
func GetProtonURL(protonVer, protonBaseURL string) string {
	// Extract the directory prefix by stripping "proton-" prefix and "-x86_64" suffix
	prefix := protonVer
	if len(prefix) > 7 && prefix[:7] == "proton-" {
		prefix = prefix[7:]
	}
	if len(prefix) > 7 && prefix[len(prefix)-7:] == "-x86_64" {
		prefix = prefix[:len(prefix)-7]
	}
	// Use the full protonVer for the filename, prefix for the directory
	return fmt.Sprintf("%s/%s/%s.tar.xz", protonBaseURL, prefix, protonVer)
}

// DownloadLauncherInstaller downloads the launcher installer to a cache directory
func DownloadLauncherInstaller(workdir string, logger *core.Logger) (*LauncherInstallerState, error) {
	downloadDir := filepath.Join(workdir, "installer-cache")
	filename := "AstarteLauncher-amd64-installer.exe"
	dest := filepath.Join(downloadDir, filename)

	if err := os.MkdirAll(downloadDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create download directory: %w", err)
	}

	logger.Info(fmt.Sprintf("Downloading AstarteLauncher installer: %s", filename))

	logFile := filepath.Join(workdir, "logs", "installer.log")
	// Create the logs directory
	if err := os.MkdirAll(filepath.Dir(logFile), 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}
	if err := core.RunCommand(core.RunModeSilent, []string{"wget", "-O", dest, launcherInstallerURL}, logger, logFile, nil, nil); err != nil {
		return nil, core.LogAndReturn(fmt.Errorf("failed to download launcher installer: %w", err), core.ErrorLevelCritical, logger)
	}

	if _, err := os.Stat(dest); os.IsNotExist(err) {
		return nil, core.LogAndReturn(fmt.Errorf("download verification failed: launcher installer not found"), core.ErrorLevelCritical, logger)
	}

	return &LauncherInstallerState{
		InstallerPath: dest,
		Downloaded:    true,
		DownloadDir:   downloadDir,
	}, nil
}

// CleanupLauncherInstaller removes the downloaded launcher installer
func CleanupLauncherInstaller(state *LauncherInstallerState, logger *core.Logger) error {
	if !state.Downloaded || state.InstallerPath == "" {
		return nil
	}

	logger.Info("Cleaning up downloaded launcher installer...")

	if err := os.Remove(state.InstallerPath); err != nil && !os.IsNotExist(err) {
		logger.Warn(fmt.Sprintf("Failed to remove launcher installer: %v", err))
	}

	if state.DownloadDir != "" {
		if err := os.RemoveAll(state.DownloadDir); err != nil && !os.IsNotExist(err) {
			logger.Warn(fmt.Sprintf("Failed to remove launcher installer directory: %v (directory not empty or does not exist)", err))
		}
	}

	return nil
}

// // GetProtonURL returns the download URL for Proton (same for AMD and NVIDIA)
// func GetProtonURL(protonVer, protonBaseURL string) string {
// 	return Get(protonVer, protonBaseURL)
// }

// GetLocalProtonPath returns the path to a local Proton package if it exists
func GetLocalProtonPath(workdir, protonVer string) string {
	localProtonPath := filepath.Join(workdir, "packages", protonVer+".tar.gz")
	if _, err := os.Stat(localProtonPath); err == nil {
		return localProtonPath
	}
	return ""
}

// GetProtonInstallPath returns the path to the proton install directory
func GetProtonInstallPath(protonVer string) (string, error) {
	bellumDir, err := config.GetBellumInstallPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(bellumDir, "proton", fmt.Sprintf("bellum-%s", protonVer)), nil
}

// GetWineInstallPath returns the path to the Wine install directory
func GetWineInstallPath(wineVer string) (string, error) {
	bellumDir, err := config.GetBellumInstallPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(bellumDir, wineVer), nil
}

// GetEACRuntimeInstallPath returns the path to the EAC Runtime install directory
func GetEACRuntimeInstallPath() (string, error) {
	bellumDir, err := config.GetBellumInstallPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(bellumDir, "EAC_Runtime"), nil
}

// EnsureEACRuntime extracts and installs the EAC Runtime package
func EnsureEACRuntime(workdir string, logger *core.Logger) error {
	eacDir, err := GetEACRuntimeInstallPath()
	if err != nil {
		return err
	}

	// Check if EAC Runtime directory already exists
	if _, err := os.Stat(eacDir); err == nil {
		logger.Info("EAC Runtime already installed")
		return nil
	}

	// Check for EAC Runtime archive
	archivePath := filepath.Join(workdir, "packages", "EAC_Runtime.tar.gz")
	if _, err := os.Stat(archivePath); os.IsNotExist(err) {
		logger.Warn(fmt.Sprintf("EAC Runtime archive not found: %s, skipping", archivePath))
		return nil
	}

	logger.Info("Installing EAC Runtime...")
	if err := os.MkdirAll(eacDir, 0755); err != nil {
		return fmt.Errorf("failed to create EAC Runtime directory: %w", err)
	}

	// Extract tar.gz (strip one level - the archive root directory)
	if err := ExtractPackageTo(archivePath, eacDir, 1); err != nil {
		os.RemoveAll(eacDir)
		return fmt.Errorf("failed to extract EAC Runtime: %w", err)
	}

	logger.Info("[OK] EAC Runtime installed")
	return nil
}

// GetWineBinPath returns the path to the Wine bin directory
func GetWineBinPath(wineDir string) string {
	return filepath.Join(wineDir, "bin")
}

// GetWineURL returns the download URL for Wine based on the version
func GetWineURL(wineVer string) string {
	// Use the full version string directly (e.g., "bellum-wine-11.8.tar.gz")
	return fmt.Sprintf("https://github.com/joepaji/bellum-linux-installer/releases/download/winepkg/%s.tar.gz", wineVer)
}

// EnsureWine downloads and sets up the Wine directory
func EnsureWine(wineVer string, logger *core.Logger) (string, error) {
	wineURL := GetWineURL(wineVer)

	// Get the actual wine install path
	wineDir, err := GetWineInstallPath(wineVer)
	if err != nil {
		return "", err
	}

	// Check if wine directory exists
	dirExists := true
	if _, err := os.Stat(wineDir); os.IsNotExist(err) {
		dirExists = false
	}

	if !dirExists {
		archivePath := ""
		tmpDir := ""

		// Check for WINE_PKG_OVERRIDE env var (dev/debug only, not exposed to end users)
		overridePath := os.Getenv("WINE_PKG_OVERRIDE")
		if overridePath != "" {
			logger.Info(fmt.Sprintf("Using Wine package from WINE_PKG_OVERRIDE: %s", overridePath))
			if _, err := os.Stat(overridePath); os.IsNotExist(err) {
				return "", fmt.Errorf("Wine package override file not found: %s", overridePath)
			}
			archivePath = overridePath
		} else {
			logger.Info("Wine directory not found, downloading...")

			tmpDir, err = os.MkdirTemp("", "wine.")
			if err != nil {
				return "", fmt.Errorf("failed to create temp directory for Wine download: %w", err)
			}
			defer os.RemoveAll(tmpDir)

			archivePath = filepath.Join(tmpDir, fmt.Sprintf("%s.tar.gz", wineVer))
			if err := downloadFile(archivePath, wineURL, logger); err != nil {
				return "", err
			}
		}

		if archivePath != "" {
			if _, err := os.Stat(archivePath); os.IsNotExist(err) {
				return "", fmt.Errorf("archive file not found: %s", archivePath)
			}

			logger.Info(fmt.Sprintf("Extracting Wine to %s...", wineDir))
			if err := os.MkdirAll(wineDir, 0755); err != nil {
				return "", fmt.Errorf("failed to create Wine directory: %w", err)
			}

			// Extract tar.gz (strip one level - the archive root directory)
			if err := ExtractPackageTo(archivePath, wineDir, 1); err != nil {
				os.RemoveAll(wineDir)
				return "", fmt.Errorf("failed to extract Wine: %w", err)
			}

			// Only remove the archive if it was downloaded (not from override)
			if overridePath == "" {
				if err := os.Remove(archivePath); err != nil {
					logger.Warn(fmt.Sprintf("Failed to remove Wine archive: %v", err))
				}
			}
		}
	}

	return wineDir, nil
}

// EnsureProton downloads and sets up the Proton directory
func EnsureProton(protonDir, protonVer string, isAMD bool, isFSR41 bool, logger *core.Logger) error {
	// Use proton-cachyos for all GPUs (AMD and NVIDIA)
	protonURL := GetProtonURL(protonVer, config.DefaultVersions.ProtonBaseURL)

	// Use the dedicated proton install directory
	actualProtonDir, err := GetProtonInstallPath(protonVer)
	if err != nil {
		return err
	}

	// Check if proton directory exists
	dirExists := true
	if _, err := os.Stat(actualProtonDir); os.IsNotExist(err) {
		dirExists = false
	}

	// Check for user_settings.py
	settingsFile := GetProtonUserSettingsPath(actualProtonDir)

	if !dirExists || settingsFile == "" {
		// Download and extract
		logger.Info("Proton directory not found, checking for cached version...")

		// Check for cached version - only if proton directory exists but is empty
		if dirExists {
			// Directory exists but may be empty - check if it has any content
			entries, err := os.ReadDir(actualProtonDir)
			if err == nil && len(entries) == 0 {
				// Directory is empty, remove it and re-download
				logger.Info("Proton directory is empty, removing...")
				os.RemoveAll(actualProtonDir)
				dirExists = false
			}
		}

		// Now check for cached version
		if !dirExists {
			// Also check in packages/ subdirectory since Proton is downloaded there
			cachedPattern := filepath.Join("./", "packages", "proton-*")
			matches, err := filepath.Glob(cachedPattern)
			if err == nil && len(matches) > 0 {
				// Verify the cached version has the expected files
				for _, match := range matches {
					if strings.HasSuffix(match, protonVer) {
						if _, err := os.Stat(filepath.Join(match, "user_settings.py")); err == nil {
							logger.Info(fmt.Sprintf("Found cached Proton in %s", match))
							// Copy from cache to new location
							logger.Info(fmt.Sprintf("Copying cached Proton from %s to %s...", match, actualProtonDir))
							if err := copyDirectory(match, actualProtonDir); err != nil {
								logger.Warn(fmt.Sprintf("Failed to copy cached Proton: %v", err))
							}
							// Re-check settings file after copy
							settingsFile = GetProtonUserSettingsPath(actualProtonDir)
							if settingsFile != "" {
								return nil
							}
						}
					}
				}
			}
		}

		logger.Info(fmt.Sprintf("Downloading Proton %s...", protonVer))

		tmpDir, err := os.MkdirTemp("", "proton.")
		if err != nil {
			return fmt.Errorf("failed to create temp directory for Proton download: %w", err)
		}
		defer os.RemoveAll(tmpDir)

		archivePath := filepath.Join(tmpDir, protonVer+".tar.xz")
		if err := downloadFile(archivePath, protonURL, logger); err != nil {
			return err
		}

		// Verify archive exists
		if _, err := os.Stat(archivePath); os.IsNotExist(err) {
			return fmt.Errorf("archive file not found after download: %s", archivePath)
		}

		logger.Info(fmt.Sprintf("Extracting Proton to %s...", actualProtonDir))
		if err := os.MkdirAll(actualProtonDir, 0755); err != nil {
			return fmt.Errorf("failed to create Proton directory: %w", err)
		}

		// Extract tar.xz (strip one level - the archive root directory)
		if err := ExtractPackageTo(archivePath, actualProtonDir, 1); err != nil {
			os.RemoveAll(actualProtonDir)
			return fmt.Errorf("failed to extract Proton: %w", err)
		}
	}

	// Check for user_settings.py
	settingsFile = GetProtonUserSettingsPath(actualProtonDir)
	if settingsFile == "" {
		return core.LogAndReturn(fmt.Errorf("Proton user settings file missing: %s", settingsFile), core.ErrorLevelCritical, logger)
	}

	// Patch settings
	if err := PatchProtonSettings(settingsFile, isAMD, isFSR41); err != nil {
		os.RemoveAll(actualProtonDir)
		return core.LogAndReturn(fmt.Errorf("failed to patch Proton user settings: %w", err), core.ErrorLevelCritical, logger)
	}

	return nil
}

// copyDirectory copies a directory recursively
func copyDirectory(srcDir, dstDir string) error {
	return filepath.Walk(srcDir, func(srcPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(srcDir, srcPath)
		if err != nil {
			return err
		}
		dstPath := filepath.Join(dstDir, relPath)
		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}
		data, err := os.ReadFile(srcPath)
		if err != nil {
			return err
		}
		return os.WriteFile(dstPath, data, info.Mode())
	})
}

// downloadFile downloads a file using wget
func downloadFile(dest, url string, logger *core.Logger) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Clean the path to resolve .. components
	logFile := filepath.Clean(filepath.Join(filepath.Dir(dest), "..", "logs", "installer.log"))
	// Create the logs directory
	if err := os.MkdirAll(filepath.Dir(logFile), 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}
	if err := core.RunCommand(core.RunModeSilent, []string{"wget", "-O", dest, url}, logger, logFile, nil, nil); err != nil {
		return core.LogAndReturn(fmt.Errorf("failed to download %s: %w", url, err), core.ErrorLevelCritical, logger)
	}

	if _, err := os.Stat(dest); os.IsNotExist(err) {
		return fmt.Errorf("download verification failed: %s not found", dest)
	}

	return nil
}

// extractTarXZWithStrip extracts a tar.xz archive with a given number of path components stripped
func extractTarXZWithStrip(archivePath, destDir string, stripComponents int) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open archive: %w", err)
	}
	defer f.Close()

	xzReader, err := xz.NewReader(f)
	if err != nil {
		return fmt.Errorf("failed to create xz reader: %w", err)
	}

	tr := tar.NewReader(xzReader)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar entry: %w", err)
		}

		// Apply strip components
		parts := strings.Split(header.Name, string(filepath.Separator))
		if len(parts) > stripComponents {
			header.Name = strings.Join(parts[stripComponents:], string(filepath.Separator))
		}

		target := filepath.Join(destDir, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return fmt.Errorf("failed to create parent directory: %w", err)
			}
			outFile, err := os.Create(target)
			if err != nil {
				return fmt.Errorf("failed to create file: %w", err)
			}
			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return fmt.Errorf("failed to write file: %w", err)
			}
			outFile.Close()
		}
	}

	return nil
}
