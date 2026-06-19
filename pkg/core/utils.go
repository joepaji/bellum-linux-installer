package core

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// IsDir checks if a path is a directory.
// Returns false if the path does not exist or is not a directory.
func IsDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// IsMountpoint checks if a path is a mountpoint.
// Uses findmnt command to determine if the path is a mountpoint.
func IsMountpoint(path string) bool {
	return isMountpoint(path)
}

// isMountpoint checks if a path is a mountpoint using findmnt
func isMountpoint(path string) bool {
	cmd := exec.Command("findmnt", "-n", "-o", "TARGET", "-T", path)
	output, err := cmd.Output()
	if err != nil {
		return isMountpointFallback(path)
	}

	mountTarget := strings.TrimSpace(string(output))
	return mountTarget == path
}

// isMountpointFallback is a fallback method to check if a path is a mountpoint
func isMountpointFallback(path string) bool {
	// Get device info for the path
	cmd := exec.Command("stat", "-c", "%d:%u", path)
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	pathDevIno := strings.TrimSpace(string(output))

	// Get device info for the parent directory
	parent := filepath.Dir(path)
	cmd = exec.Command("stat", "-c", "%d:%u", parent)
	output, err = cmd.Output()
	if err != nil {
		return false
	}

	parentDevIno := strings.TrimSpace(string(output))

	// If device numbers differ, path is on a different filesystem (likely a mountpoint)
	pathDev := strings.Split(pathDevIno, ":")[0]
	parentDev := strings.Split(parentDevIno, ":")[0]

	return pathDev != parentDev
}

// IsEmptyDir checks if a directory is empty.
// Returns true if the directory is empty, false otherwise.
func IsEmptyDir(path string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()

	_, err = file.Readdirnames(1)
	return err == io.EOF
}

// IsWritable checks if a path is writable by attempting to create a test file.
// Returns true if a test file can be created successfully, false otherwise.
func IsWritable(path string) bool {
	testFile := filepath.Join(path, ".write_test_"+fmt.Sprintf("%d", os.Getpid()))
	file, err := os.Create(testFile)
	if err != nil {
		return false
	}
	if err := file.Close(); err != nil {
		return false
	}
	os.Remove(testFile)
	return true
}

// CopyFile copies a file from src to dst with proper error handling and cleanup.
// If the copy fails, it removes any partial destination file.
func CopyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer source.Close()

	destination, err := os.OpenFile(dst, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destination.Close()

	if _, err := io.Copy(destination, source); err != nil {
		os.Remove(dst)
		return fmt.Errorf("failed to copy file: %w", err)
	}

	if err := destination.Close(); err != nil {
		os.Remove(dst)
		return fmt.Errorf("failed to close destination file: %w", err)
	}

	return nil
}
