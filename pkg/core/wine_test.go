package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestScopedWineCapturesOriginalPath verifies that NewScopedWine captures
// the original PATH before any modifications.
func TestScopedWineCapturesOriginalPath(t *testing.T) {
	originalPath, pathExists := os.LookupEnv("PATH")
	if pathExists {
		defer os.Setenv("PATH", originalPath)
	} else {
		defer os.Unsetenv("PATH")
	}

	// Set a known PATH for testing
	testPath := "/custom/bin:/usr/bin"
	os.Setenv("PATH", testPath)

	scoped := NewScopedWine("/home/user/.local/share/bellum/bellum-wine-11.8/bin")

	// PATH should still be the original at this point (captured on construction)
	captured := scoped.originalPath
	if captured != testPath {
		t.Errorf("Expected captured PATH to be %q, got %q", testPath, captured)
	}
}

// TestScopedWineApplyPrependsToPath verifies that Apply() prepends the wine
// bin directory to PATH.
func TestScopedWineApplyPrependsToPath(t *testing.T) {
	originalPath, pathExists := os.LookupEnv("PATH")
	originalLDLib, ldLibExists := os.LookupEnv("LD_LIBRARY_PATH")

	defer func() {
		if pathExists {
			os.Setenv("PATH", originalPath)
		} else {
			os.Unsetenv("PATH")
		}
		if ldLibExists {
			os.Setenv("LD_LIBRARY_PATH", originalLDLib)
		} else {
			os.Unsetenv("LD_LIBRARY_PATH")
		}
	}()

	os.Setenv("PATH", "/original/bin:/usr/bin")
	os.Unsetenv("LD_LIBRARY_PATH")

	wineBinDir := "/home/user/.local/share/bellum/bellum-wine-11.8/bin"
	scoped := NewScopedWine(wineBinDir)
	scoped.Apply()

	currentPath := os.Getenv("PATH")

	// PATH should start with the wine bin dir
	if !strings.HasPrefix(currentPath, wineBinDir) {
		t.Errorf("Expected PATH to start with %q, got: %s", wineBinDir, currentPath)
	}

	// The original PATH should be in the scoped PATH
	expectedScopedPath := wineBinDir + string(filepath.ListSeparator) + "/original/bin:/usr/bin"
	if currentPath != expectedScopedPath {
		t.Errorf("Expected PATH to be %q, got %q", expectedScopedPath, currentPath)
	}
}

// TestScopedWineApplyAddsLibPaths verifies that Apply() adds wine lib/ and
// lib64/ directories to LD_LIBRARY_PATH when they exist.
func TestScopedWineApplyAddsLibPaths(t *testing.T) {
	originalPath, pathExists := os.LookupEnv("PATH")
	originalLDLib, ldLibExists := os.LookupEnv("LD_LIBRARY_PATH")

	// Restore environment at the end of the test
	defer func() {
		if pathExists {
			os.Setenv("PATH", originalPath)
		} else {
			os.Unsetenv("PATH")
		}
		if ldLibExists {
			os.Setenv("LD_LIBRARY_PATH", originalLDLib)
		} else {
			os.Unsetenv("LD_LIBRARY_PATH")
		}
	}()

	// Create a temp directory structure mimicking the wine install
	tmpDir, err := os.MkdirTemp("", "test-scoped-wine-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create bin, lib/wine/x86_64-unix, lib/wine/x86_64-windows directories
	binDir := filepath.Join(tmpDir, "bin")
	libUnixDir := filepath.Join(tmpDir, "lib", "wine", "x86_64-unix")
	libWindowsDir := filepath.Join(tmpDir, "lib", "wine", "x86_64-windows")

	os.MkdirAll(binDir, 0755)
	os.MkdirAll(libUnixDir, 0755)
	os.MkdirAll(libWindowsDir, 0755)

	os.Unsetenv("LD_LIBRARY_PATH")
	os.Setenv("PATH", "/usr/bin")

	scoped := NewScopedWine(binDir)
	scoped.Apply()

	currentLDLib := os.Getenv("LD_LIBRARY_PATH")

	if currentLDLib == "" {
		t.Fatal("Expected LD_LIBRARY_PATH to be set after Apply()")
	}

	expectedLibUnix := filepath.Join(tmpDir, "lib", "wine", "x86_64-unix")
	expectedLibWindows := filepath.Join(tmpDir, "lib", "wine", "x86_64-windows")

	if !strings.Contains(currentLDLib, expectedLibUnix) {
		t.Errorf("Expected LD_LIBRARY_PATH to contain %q, got: %s", expectedLibUnix, currentLDLib)
	}

	if !strings.Contains(currentLDLib, expectedLibWindows) {
		t.Errorf("Expected LD_LIBRARY_PATH to contain %q, got: %s", expectedLibWindows, currentLDLib)
	}
}

// TestScopedWineApplyLibPathsOnlyIfExists verifies that Apply() only adds
// lib/ and lib64/ to LD_LIBRARY_PATH when those directories actually exist.
func TestScopedWineApplyLibPathsOnlyIfExists(t *testing.T) {
	originalPath, pathExists := os.LookupEnv("PATH")
	originalLDLib, ldLibExists := os.LookupEnv("LD_LIBRARY_PATH")

	defer func() {
		if pathExists {
			os.Setenv("PATH", originalPath)
		} else {
			os.Unsetenv("PATH")
		}
		if ldLibExists {
			os.Setenv("LD_LIBRARY_PATH", originalLDLib)
		} else {
			os.Unsetenv("LD_LIBRARY_PATH")
		}
	}()

	// Create a temp directory with only bin/ (no lib/ or lib64/)
	tmpDir, err := os.MkdirTemp("", "test-scoped-wine-nolib-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	binDir := filepath.Join(tmpDir, "bin")
	os.MkdirAll(binDir, 0755)

	os.Unsetenv("LD_LIBRARY_PATH")

	scoped := NewScopedWine(binDir)
	scoped.Apply()

	currentLDLib := os.Getenv("LD_LIBRARY_PATH")

	// LD_LIBRARY_PATH should remain empty/unset when no lib dirs exist
	if currentLDLib != "" {
		t.Errorf("Expected LD_LIBRARY_PATH to be empty when no lib dirs exist, got: %q", currentLDLib)
	}
}

// TestScopedWineApplyPreservesExistingLDLibraryPath verifies that Apply()
// appends the existing LD_LIBRARY_PATH when it is set.
func TestScopedWineApplyPreservesExistingLDLibraryPath(t *testing.T) {
	originalPath, pathExists := os.LookupEnv("PATH")
	originalLDLib, ldLibExists := os.LookupEnv("LD_LIBRARY_PATH")

	// Restore environment at the end of the test
	defer func() {
		if pathExists {
			os.Setenv("PATH", originalPath)
		} else {
			os.Unsetenv("PATH")
		}
		if ldLibExists {
			os.Setenv("LD_LIBRARY_PATH", originalLDLib)
		} else {
			os.Unsetenv("LD_LIBRARY_PATH")
		}
	}()

	tmpDir, err := os.MkdirTemp("", "test-scoped-wine-existing-ld-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	binDir := filepath.Join(tmpDir, "bin")
	libUnixDir := filepath.Join(tmpDir, "lib", "wine", "x86_64-unix")
	os.MkdirAll(binDir, 0755)
	os.MkdirAll(libUnixDir, 0755)

	existingLDLib := "/existing/lib:/another/lib"
	os.Setenv("LD_LIBRARY_PATH", existingLDLib)
	os.Setenv("PATH", "/usr/bin")

	scoped := NewScopedWine(binDir)
	scoped.Apply()

	currentLDLib := os.Getenv("LD_LIBRARY_PATH")

	// Should contain both the new lib path and the existing one
	expectedLibUnix := filepath.Join(tmpDir, "lib", "wine", "x86_64-unix")
	if !strings.Contains(currentLDLib, expectedLibUnix) {
		t.Errorf("Expected LD_LIBRARY_PATH to contain %q, got: %s", expectedLibUnix, currentLDLib)
	}

	if !strings.Contains(currentLDLib, existingLDLib) {
		t.Errorf("Expected LD_LIBRARY_PATH to contain existing value %q, got: %s", existingLDLib, currentLDLib)
	}
}

// TestScopedWineRestoreRevertsPath verifies that Restore() restores the
// original PATH.
func TestScopedWineRestoreRevertsPath(t *testing.T) {
	originalPath, pathExists := os.LookupEnv("PATH")
	originalLDLib, ldLibExists := os.LookupEnv("LD_LIBRARY_PATH")

	// Restore environment at the end of the test
	defer func() {
		if pathExists {
			os.Setenv("PATH", originalPath)
		} else {
			os.Unsetenv("PATH")
		}
		if ldLibExists {
			os.Setenv("LD_LIBRARY_PATH", originalLDLib)
		} else {
			os.Unsetenv("LD_LIBRARY_PATH")
		}
	}()

	os.Setenv("PATH", "/original/bin:/usr/bin")
	os.Unsetenv("LD_LIBRARY_PATH")

	wineBinDir := "/home/user/.local/share/bellum/bellum-wine-11.8/bin"
	scoped := NewScopedWine(wineBinDir)
	scoped.Apply()

	// Verify PATH was modified
	currentPath := os.Getenv("PATH")
	if currentPath == "/original/bin:/usr/bin" {
		t.Fatal("Expected PATH to be modified after Apply()")
	}

	scoped.Restore()

	// PATH should be restored to original
	restoredPath := os.Getenv("PATH")
	if restoredPath != "/original/bin:/usr/bin" {
		t.Errorf("Expected PATH to be restored to %q, got %q", "/original/bin:/usr/bin", restoredPath)
	}
}

// TestScopedWineRestoreRestoresLDLibraryPath verifies that Restore() reverts
// LD_LIBRARY_PATH to its original value.
func TestScopedWineRestoreRestoresLDLibraryPath(t *testing.T) {
	originalPath, pathExists := os.LookupEnv("PATH")
	originalLDLib, ldLibExists := os.LookupEnv("LD_LIBRARY_PATH")

	// Restore environment at the end of the test
	defer func() {
		if pathExists {
			os.Setenv("PATH", originalPath)
		} else {
			os.Unsetenv("PATH")
		}
		if ldLibExists {
			os.Setenv("LD_LIBRARY_PATH", originalLDLib)
		} else {
			os.Unsetenv("LD_LIBRARY_PATH")
		}
	}()

	tmpDir, err := os.MkdirTemp("", "test-scoped-wine-restore-ld-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	binDir := filepath.Join(tmpDir, "bin")
	libUnixDir := filepath.Join(tmpDir, "lib", "wine", "x86_64-unix")
	os.MkdirAll(binDir, 0755)
	os.MkdirAll(libUnixDir, 0755)

	os.Setenv("LD_LIBRARY_PATH", "/original/lib:/original/lib64")
	os.Setenv("PATH", "/original/path")

	scoped := NewScopedWine(binDir)
	scoped.Apply()

	// Verify LD_LIBRARY_PATH was modified
	afterApply := os.Getenv("LD_LIBRARY_PATH")
	if afterApply == "/original/lib:/original/lib64" {
		t.Fatal("Expected LD_LIBRARY_PATH to be modified after Apply()")
	}

	scoped.Restore()

	// LD_LIBRARY_PATH should be restored to original
	restoredLDLib := os.Getenv("LD_LIBRARY_PATH")
	if restoredLDLib != "/original/lib:/original/lib64" {
		t.Errorf("Expected LD_LIBRARY_PATH to be restored to %q, got %q", "/original/lib:/original/lib64", restoredLDLib)
	}
}

// TestScopedWineRestoreUnsetsLDLibraryPath verifies that Restore() unsets
// LD_LIBRARY_PATH when it was not set before Apply().
func TestScopedWineRestoreUnsetsLDLibraryPath(t *testing.T) {
	originalPath, pathExists := os.LookupEnv("PATH")
	originalLDLib, ldLibExists := os.LookupEnv("LD_LIBRARY_PATH")

	// Restore environment at the end of the test
	defer func() {
		if pathExists {
			os.Setenv("PATH", originalPath)
		} else {
			os.Unsetenv("PATH")
		}
		if ldLibExists {
			os.Setenv("LD_LIBRARY_PATH", originalLDLib)
		} else {
			os.Unsetenv("LD_LIBRARY_PATH")
		}
	}()

	tmpDir, err := os.MkdirTemp("", "test-scoped-wine-unset-ld-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	binDir := filepath.Join(tmpDir, "bin")
	libUnixDir := filepath.Join(tmpDir, "lib", "wine", "x86_64-unix")
	os.MkdirAll(binDir, 0755)
	os.MkdirAll(libUnixDir, 0755)

	os.Unsetenv("LD_LIBRARY_PATH")
	os.Setenv("PATH", "/original/path")

	scoped := NewScopedWine(binDir)
	scoped.Apply()

	// Verify LD_LIBRARY_PATH was set
	if os.Getenv("LD_LIBRARY_PATH") == "" {
		t.Fatal("Expected LD_LIBRARY_PATH to be set after Apply()")
	}

	scoped.Restore()

	// LD_LIBRARY_PATH should be unset
	_, exists := os.LookupEnv("LD_LIBRARY_PATH")
	if exists {
		t.Errorf("Expected LD_LIBRARY_PATH to be unset after Restore(), but it is: %q", os.Getenv("LD_LIBRARY_PATH"))
	}
}

// TestScopedWineRestoreRevertsPathEvenWhenLDLibraryPathWasUnset verifies
// that PATH is restored even when LD_LIBRARY_PATH was not originally set.
func TestScopedWineRestoreRevertsPathEvenWhenLDLibraryPathWasUnset(t *testing.T) {
	originalPath, pathExists := os.LookupEnv("PATH")
	originalLDLib, ldLibExists := os.LookupEnv("LD_LIBRARY_PATH")

	// Restore environment at the end of the test
	defer func() {
		if pathExists {
			os.Setenv("PATH", originalPath)
		} else {
			os.Unsetenv("PATH")
		}
		if ldLibExists {
			os.Setenv("LD_LIBRARY_PATH", originalLDLib)
		} else {
			os.Unsetenv("LD_LIBRARY_PATH")
		}
	}()

	os.Setenv("PATH", "/original/bin")
	os.Unsetenv("LD_LIBRARY_PATH")

	wineBinDir := "/home/user/.local/share/bellum/bellum-wine-11.8/bin"
	scoped := NewScopedWine(wineBinDir)
	scoped.Apply()

	scoped.Restore()

	// PATH should still be restored
	restoredPath := os.Getenv("PATH")
	if restoredPath != "/original/bin" {
		t.Errorf("Expected PATH to be restored to %q, got %q", "/original/bin", restoredPath)
	}
}

// TestScopedWinePathDoesNotContainSystemWineBin verifies that after Apply(),
// the scoped PATH does not contain system wine paths (e.g. /usr/bin) before
// the wine bin dir. The packaged wine bin dir should be first.
func TestScopedWinePathDoesNotContainSystemWineBin(t *testing.T) {
	originalPath, pathExists := os.LookupEnv("PATH")
	originalLDLib, ldLibExists := os.LookupEnv("LD_LIBRARY_PATH")

	defer func() {
		if pathExists {
			os.Setenv("PATH", originalPath)
		} else {
			os.Unsetenv("PATH")
		}
		if ldLibExists {
			os.Setenv("LD_LIBRARY_PATH", originalLDLib)
		} else {
			os.Unsetenv("LD_LIBRARY_PATH")
		}
	}()

	os.Setenv("PATH", "/usr/bin:/usr/local/bin")
	os.Unsetenv("LD_LIBRARY_PATH")

	wineBinDir := "/home/user/.local/share/bellum/bellum-wine-11.8/bin"
	scoped := NewScopedWine(wineBinDir)
	scoped.Apply()

	currentPath := os.Getenv("PATH")

	// Wine bin dir should appear before /usr/bin
	wineBinIndex := strings.Index(currentPath, wineBinDir)
	systemBinIndex := strings.Index(currentPath, "/usr/bin")

	if wineBinIndex < 0 {
		t.Errorf("Expected wine bin dir %q to be in PATH, got: %s", wineBinDir, currentPath)
	}

	if systemBinIndex < 0 {
		t.Errorf("Expected /usr/bin to be in PATH, got: %s", currentPath)
	}

	if wineBinIndex > systemBinIndex {
		t.Errorf("Expected wine bin dir %q to appear before /usr/bin in PATH, got: %s", wineBinDir, currentPath)
	}
}

// TestScopedWineInstallPathStructure verifies that the wine install path
// follows the expected structure: ~/.local/share/bellum/bellum-wine-<version>/bin
func TestScopedWineInstallPathStructure(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Failed to get home dir: %v", err)
	}

	expectedBase := filepath.Join(homeDir, ".local", "share", "bellum", "bellum-wine-11.8")
	expectedBin := filepath.Join(expectedBase, "bin")

	scoped := NewScopedWine(expectedBin)

	if scoped.wineBinDir != expectedBin {
		t.Errorf("Expected wine bin dir to be %q, got %q", expectedBin, scoped.wineBinDir)
	}

	// Verify the ScopedWine struct captured the path correctly
	if scoped.originalPath == "" {
		t.Error("Expected originalPath to be captured")
	}
}

// TestScopedWineRestoreIsIdempotent verifies that calling Restore() multiple
// times does not cause errors.
func TestScopedWineRestoreIsIdempotent(t *testing.T) {
	originalPath, pathExists := os.LookupEnv("PATH")
	originalLDLib, ldLibExists := os.LookupEnv("LD_LIBRARY_PATH")

	defer func() {
		if pathExists {
			os.Setenv("PATH", originalPath)
		} else {
			os.Unsetenv("PATH")
		}
		if ldLibExists {
			os.Setenv("LD_LIBRARY_PATH", originalLDLib)
		} else {
			os.Unsetenv("LD_LIBRARY_PATH")
		}
	}()

	os.Setenv("PATH", "/original/path")
	os.Setenv("LD_LIBRARY_PATH", "/original/lib")

	scoped := NewScopedWine("/some/wine/bin")
	scoped.Apply()

	// Call Restore multiple times - should not panic or error
	scoped.Restore()
	scoped.Restore()

	// PATH should still be the original
	if os.Getenv("PATH") != "/original/path" {
		t.Errorf("Expected PATH to be %q after multiple Restores, got %q", "/original/path", os.Getenv("PATH"))
	}
}

// TestScopedWineBinaryPathInInstallDir verifies that the wine binary path
// constructed from the wine bin directory points to the packaged wine
// installation under ~/.local/share/bellum/ instead of system wine.
func TestScopedWineBinaryPathInInstallDir(t *testing.T) {
	originalPath, pathExists := os.LookupEnv("PATH")
	originalLDLib, ldLibExists := os.LookupEnv("LD_LIBRARY_PATH")

	defer func() {
		if pathExists {
			os.Setenv("PATH", originalPath)
		} else {
			os.Unsetenv("PATH")
		}
		if ldLibExists {
			os.Setenv("LD_LIBRARY_PATH", originalLDLib)
		} else {
			os.Unsetenv("LD_LIBRARY_PATH")
		}
	}()

	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Failed to get home dir: %v", err)
	}

	wineBinPath := filepath.Join(homeDir, ".local", "share", "bellum", "bellum-wine-11.8", "bin")
	wineBin := filepath.Join(wineBinPath, "wine")

	scoped := NewScopedWine(wineBinPath)
	scoped.Apply()

	// Verify PATH starts with the packaged wine bin dir
	currentPath := os.Getenv("PATH")
	if !strings.HasPrefix(currentPath, wineBinPath) {
		t.Errorf("Expected PATH to start with packaged wine bin %q, got: %s", wineBinPath, currentPath)
	}

	// Verify the wine binary path is within the bellum install directory
	expectedWineBin := filepath.Join(homeDir, ".local", "share", "bellum", "bellum-wine-11.8", "bin", "wine")
	if wineBin != expectedWineBin {
		t.Errorf("Expected wine binary path to be %q, got %q", expectedWineBin, wineBin)
	}

	// Verify the wine binary path is NOT a system path
	if strings.HasPrefix(wineBin, "/usr/") || strings.HasPrefix(wineBin, "/bin/") || strings.HasPrefix(wineBin, "/sbin/") {
		t.Errorf("Wine binary path %q should not point to a system directory", wineBin)
	}

	// Verify the wine binary path contains the bellum install path
	if !strings.Contains(wineBin, ".local/share/bellum") {
		t.Errorf("Wine binary path %q should contain .local/share/bellum", wineBin)
	}
}

// TestScopedWineLDLibraryPathContainsPackagedLibs verifies that after Apply(),
// LD_LIBRARY_PATH contains the packaged wine lib directories, not system libs.
func TestScopedWineLDLibraryPathContainsPackagedLibs(t *testing.T) {
	originalPath, pathExists := os.LookupEnv("PATH")
	originalLDLib, ldLibExists := os.LookupEnv("LD_LIBRARY_PATH")

	// Restore environment at the end of the test
	defer func() {
		if pathExists {
			os.Setenv("PATH", originalPath)
		} else {
			os.Unsetenv("PATH")
		}
		if ldLibExists {
			os.Setenv("LD_LIBRARY_PATH", originalLDLib)
		} else {
			os.Unsetenv("LD_LIBRARY_PATH")
		}
	}()

	tmpDir, err := os.MkdirTemp("", "test-scoped-wine-packaged-libs-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	binDir := filepath.Join(tmpDir, "bin")
	libUnixDir := filepath.Join(tmpDir, "lib", "wine", "x86_64-unix")
	libWindowsDir := filepath.Join(tmpDir, "lib", "wine", "x86_64-windows")
	os.MkdirAll(binDir, 0755)
	os.MkdirAll(libUnixDir, 0755)
	os.MkdirAll(libWindowsDir, 0755)

	os.Unsetenv("LD_LIBRARY_PATH")
	os.Setenv("PATH", "/usr/bin")

	scoped := NewScopedWine(binDir)
	scoped.Apply()

	currentLDLib := os.Getenv("LD_LIBRARY_PATH")

	// Verify the packaged lib dirs are in LD_LIBRARY_PATH
	if !strings.Contains(currentLDLib, libUnixDir) {
		t.Errorf("Expected LD_LIBRARY_PATH to contain packaged lib dir %q, got: %s", libUnixDir, currentLDLib)
	}

	if !strings.Contains(currentLDLib, libWindowsDir) {
		t.Errorf("Expected LD_LIBRARY_PATH to contain packaged lib dir %q, got: %s", libWindowsDir, currentLDLib)
	}

	// Verify system lib dirs are NOT in LD_LIBRARY_PATH
	systemLibPaths := []string{"/usr/lib", "/usr/lib64", "/lib", "/lib64"}
	for _, sysPath := range systemLibPaths {
		// Split by separator and check each path component
		parts := strings.Split(currentLDLib, string(filepath.ListSeparator))
		for _, part := range parts {
			cleanPart := filepath.Clean(part)
			if cleanPart == sysPath || strings.HasPrefix(cleanPart, sysPath+string(filepath.Separator)) {
				t.Errorf("LD_LIBRARY_PATH should not contain system lib path %q, got: %s", sysPath, currentLDLib)
			}
		}
	}
}

// TestScopedWineRestoreAfterApplyRestoresSystemPath verifies that after
// Restore(), PATH reverts to the system PATH (e.g. /usr/bin), not the
// packaged wine path.
func TestScopedWineRestoreAfterApplyRestoresSystemPath(t *testing.T) {
	originalPath, pathExists := os.LookupEnv("PATH")
	originalLDLib, ldLibExists := os.LookupEnv("LD_LIBRARY_PATH")

	// Restore environment at the end of the test
	defer func() {
		if pathExists {
			os.Setenv("PATH", originalPath)
		} else {
			os.Unsetenv("PATH")
		}
		if ldLibExists {
			os.Setenv("LD_LIBRARY_PATH", originalLDLib)
		} else {
			os.Unsetenv("LD_LIBRARY_PATH")
		}
	}()

	os.Setenv("PATH", "/usr/bin:/usr/local/bin:/bin")
	os.Unsetenv("LD_LIBRARY_PATH")

	wineBinDir := "/home/user/.local/share/bellum/bellum-wine-11.8/bin"
	scoped := NewScopedWine(wineBinDir)
	scoped.Apply()

	// Verify Apply() prepended the wine path
	afterApply := os.Getenv("PATH")
	if afterApply == "/usr/bin:/usr/local/bin:/bin" {
		t.Fatal("Expected PATH to change after Apply()")
	}

	scoped.Restore()

	// After Restore, PATH should be the original system PATH
	restoredPath := os.Getenv("PATH")
	if restoredPath != "/usr/bin:/usr/local/bin:/bin" {
		t.Errorf("Expected PATH to revert to system PATH %q, got %q", "/usr/bin:/usr/local/bin:/bin", restoredPath)
	}

	// Verify the restored PATH does NOT contain the wine install path
	if strings.Contains(restoredPath, "bellum-wine") {
		t.Errorf("Restored PATH should not contain wine install path, got: %s", restoredPath)
	}
}

// TestScopedWineWineBinPathPointsToPackagedWine verifies that the wine
// binary path used during installation points to the packaged wine under
// ~/.local/share/bellum/ rather than system wine.
func TestScopedWineWineBinPathPointsToPackagedWine(t *testing.T) {
	originalPath, pathExists := os.LookupEnv("PATH")
	originalLDLib, ldLibExists := os.LookupEnv("LD_LIBRARY_PATH")

	defer func() {
		if pathExists {
			os.Setenv("PATH", originalPath)
		} else {
			os.Unsetenv("PATH")
		}
		if ldLibExists {
			os.Setenv("LD_LIBRARY_PATH", originalLDLib)
		} else {
			os.Unsetenv("LD_LIBRARY_PATH")
		}
	}()

	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Failed to get home dir: %v", err)
	}

	// Simulate the wine bin path that would be returned by packages.GetWineBinPath()
	wineInstallDir := filepath.Join(homeDir, ".local", "share", "bellum", "bellum-wine-11.8")
	wineBinPath := filepath.Join(wineInstallDir, "bin")

	scoped := NewScopedWine(wineBinPath)
	scoped.Apply()

	// Construct the wine binary path the way configure.go does it
	wineBin := filepath.Join(wineBinPath, "wine")

	// Verify wine binary is in the bellum install directory
	if !strings.Contains(wineBin, filepath.Join(".local", "share", "bellum")) {
		t.Errorf("Wine binary %q should be in ~/.local/share/bellum/", wineBin)
	}

	// Verify it is NOT the system wine
	systemPaths := []string{"/usr/bin/wine", "/usr/local/bin/wine", "/bin/wine"}
	for _, sysWine := range systemPaths {
		if wineBin == sysWine {
			t.Errorf("Wine binary should not point to system wine at %s", sysWine)
		}
	}

	// Verify PATH is scoped to the packaged wine
	currentPath := os.Getenv("PATH")
	if !strings.HasPrefix(currentPath, wineBinPath) {
		t.Errorf("Expected PATH to start with packaged wine bin %q, got: %s", wineBinPath, currentPath)
	}
}

// TestScopedWineNoSystemWineDependency verifies that the installer's wine
// scoping does not depend on system wine being installed. The PATH should
// only contain the packaged wine bin dir at the front.
func TestScopedWineNoSystemWineDependency(t *testing.T) {
	originalPath, pathExists := os.LookupEnv("PATH")
	originalLDLib, ldLibExists := os.LookupEnv("LD_LIBRARY_PATH")

	defer func() {
		if pathExists {
			os.Setenv("PATH", originalPath)
		} else {
			os.Unsetenv("PATH")
		}
		if ldLibExists {
			os.Setenv("LD_LIBRARY_PATH", originalLDLib)
		} else {
			os.Unsetenv("LD_LIBRARY_PATH")
		}
	}()

	// Simulate a system with no wine installed
	os.Setenv("PATH", "/usr/local/sbin:/usr/local/bin")
	os.Unsetenv("LD_LIBRARY_PATH")

	wineBinDir := "/home/user/.local/share/bellum/bellum-wine-11.8/bin"
	scoped := NewScopedWine(wineBinDir)
	scoped.Apply()

	currentPath := os.Getenv("PATH")

	// Packaged wine should be first
	if !strings.HasPrefix(currentPath, wineBinDir) {
		t.Errorf("Expected PATH to start with packaged wine %q, got: %s", wineBinDir, currentPath)
	}

	// Original PATH should still be in there (for system tools)
	if !strings.Contains(currentPath, "/usr/local/bin") {
		t.Errorf("Expected original PATH to still be present, got: %s", currentPath)
	}
}
