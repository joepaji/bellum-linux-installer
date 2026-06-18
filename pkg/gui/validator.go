package gui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"bellum-installer/pkg/core"
)

// ValidateDirectory validates a selected directory for WINEPREFIX usage
// The WINEPREFIX will be created as <path>/Bellum (or <path> if already named "Bellum")
// Returns error and error message if validation fails
func ValidateDirectory(path string, logger *core.Logger) (bool, string) {
	// Normalize path
	path = strings.TrimSuffix(path, "/")

	// Check if path is absolute
	if !strings.HasPrefix(path, "/") {
		return false, "Please select an absolute path (starting with /). Example: /games"
	}

	// Determine the resolved install path (same logic as ResolveInstallPath)
	baseName := filepath.Base(path)
	if strings.EqualFold(baseName, "Bellum") && IsDirEmpty(path) {
		// User selected an empty "Bellum" directory - validate it directly
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return false, fmt.Sprintf("The directory '%s' does not exist. Please select a directory that exists.", path)
		}
		if !core.IsWritable(path) {
			return false, fmt.Sprintf("You don't have permission to write to the directory '%s'. Please select a directory where you have write access.", path)
		}
	} else {
		// WINEPREFIX will be at path/Bellum
		bellumPath := filepath.Join(path, "Bellum")

		// Check if the selected path exists and is writable (parent of Bellum subdirectory)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return false, fmt.Sprintf("The directory '%s' does not exist. Please select a directory that exists.", path)
		}

		if !core.IsWritable(path) {
			return false, fmt.Sprintf("You don't have permission to write to the directory '%s'. Please select a directory where you have write access.", path)
		}

		// Check if Bellum directory already exists
		if core.IsDir(bellumPath) {
			return false, fmt.Sprintf("A Bellum installation already exists at '%s'. Please uninstall it first before installing again.", bellumPath)
		}

		// Check if the selected path is a mountpoint - this is now allowed
		// since we'll create Bellum inside it
		if core.IsMountpoint(path) {
			logger.Info(fmt.Sprintf("Selected path '%s' is a mountpoint, which is allowed. The WINEPREFIX will be created at '%s'.", path, bellumPath))
		}
	}

	// All checks passed
	return true, ""
}





// IsDirEmpty checks if a directory is empty (has no files or subdirectories)
func IsDirEmpty(path string) bool {
	entries, err := os.ReadDir(path)
	if err != nil {
		return false
	}
	return len(entries) == 0
}

// ResolveInstallPath determines the final install path from a user-selected directory.
// If the selected path is already named "Bellum" and is empty, it uses the path directly.
// Otherwise, it appends "Bellum" to the selected path.
func ResolveInstallPath(selectedPath string) string {
	selectedPath = strings.TrimSuffix(selectedPath, "/")
	baseName := filepath.Base(selectedPath)

	// If user selected a directory already named "Bellum" and it's empty, use it directly
	if strings.EqualFold(baseName, "Bellum") && IsDirEmpty(selectedPath) {
		return selectedPath
	}

	// Otherwise, create/use a Bellum subdirectory
	return filepath.Join(selectedPath, "Bellum")
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
