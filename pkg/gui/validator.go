package gui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"bellum-installer/pkg/core"
)

// ValidateDirectory validates a selected directory for WINEPREFIX usage
// The WINEPREFIX will be created as <path>/Bellum
// Returns error and error message if validation fails
func ValidateDirectory(path string, logger *core.Logger) (bool, string) {
	// Normalize path
	path = strings.TrimSuffix(path, "/")

	// Check if path is absolute
	if !strings.HasPrefix(path, "/") {
		return false, "Please select an absolute path (starting with /). Example: /games"
	}

	// Check if parent directory exists
	parentDir := filepath.Dir(path)
	if _, err := os.Stat(parentDir); os.IsNotExist(err) {
		return false, fmt.Sprintf("The parent directory '%s' does not exist. Please select a directory within an existing location.", parentDir)
	}

	// Check if parent directory is writable
	if !core.IsWritable(parentDir) {
		return false, fmt.Sprintf("You don't have permission to write to the parent directory '%s'. Please select a directory where you have write access.", parentDir)
	}

	// The WINEPREFIX will be at path/Bellum
	bellumPath := filepath.Join(path, "Bellum")

	// Check if Bellum directory already exists
	if core.IsDir(bellumPath) {
		return false, fmt.Sprintf("A Bellum installation already exists at '%s'. Please uninstall it first before installing again.", bellumPath)
	}

	// Check if the selected path is a mountpoint - this is now allowed
	// since we'll create Bellum inside it
	if core.IsMountpoint(path) {
		logger.Info(fmt.Sprintf("Selected path '%s' is a mountpoint, which is allowed. The WINEPREFIX will be created at '%s'.", path, bellumPath))
	}

	// All checks passed
	return true, ""
}





// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
