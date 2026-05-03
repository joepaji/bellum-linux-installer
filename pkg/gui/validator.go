package gui

import (
	"fmt"
	"os"
	"os/exec"
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
	if !isWritable(parentDir) {
		return false, fmt.Sprintf("You don't have permission to write to the parent directory '%s'. Please select a directory where you have write access.", parentDir)
	}

	// The WINEPREFIX will be at path/Bellum
	bellumPath := filepath.Join(path, "Bellum")

	// Check if Bellum directory already exists
	if isDir(bellumPath) {
		return false, fmt.Sprintf("A Bellum installation already exists at '%s'. Please uninstall it first before installing again.", bellumPath)
	}

	// Check if the selected path is a mountpoint - this is now allowed
	// since we'll create Bellum inside it
	if isMountpoint(path) {
		logger.Info(fmt.Sprintf("Selected path '%s' is a mountpoint, which is allowed. The WINEPREFIX will be created at '%s'.", path, bellumPath))
	}

	// All checks passed
	return true, ""
}

// isMountpoint checks if a path is a mountpoint
// Uses the findmnt command to determine if the path is a mountpoint
func isMountpoint(path string) bool {
	// Use findmnt to check if path is a mountpoint
	cmd := exec.Command("findmnt", "-n", "-o", "TARGET", "-T", path)
	output, err := cmd.Output()
	if err != nil {
		// findmnt may not be available, fallback to other methods
		return isMountpointFallback(path)
	}

	// If findmnt returns the path itself, it's a mountpoint
	mountTarget := strings.TrimSpace(string(output))
	return mountTarget == path
}

// isMountpointFallback is a fallback method to check if a path is a mountpoint
func isMountpointFallback(path string) bool {
	// Try stat to get device info
	cmd := exec.Command("stat", "-c", "%d:%u", path)
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	devIno := strings.TrimSpace(string(output))

	// Check parent directories to see if we're at a mount boundary
	current := path
	for current != "/" && current != "." {
		parent := filepath.Dir(current)

		// Get parent device info
		cmd := exec.Command("stat", "-c", "%d:%u", parent)
		parentOutput, err := cmd.Output()
		if err != nil {
			break
		}

		parentDevIno := strings.TrimSpace(string(parentOutput))

		// If device numbers differ, we're at a mountpoint
		// Extract just the device number (first part before :)
		devNum := strings.Split(devIno, ":")[0]
		parentDevNum := strings.Split(parentDevIno, ":")[0]

		if devNum != parentDevNum {
			return true
		}

		current = parent
	}

	return false
}

// isDir checks if a path is a directory
func isDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// isWritable checks if a path is writable
func isWritable(path string) bool {
	testFile := filepath.Join(path, ".write_test_"+fmt.Sprintf("%d", os.Getpid()))
	file, err := os.Create(testFile)
	if err != nil {
		return false
	}
	file.Close()
	os.Remove(testFile)
	return true
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
