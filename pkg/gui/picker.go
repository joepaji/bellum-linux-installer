package gui

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// PickerResult holds the result from the GUI directory picker
type PickerResult struct {
	Path    string
	Success bool
	Error   error
}

// PickDirectory opens a GUI dialog to select a directory
// Returns the selected path or an error if the user cancels or no tool is available
func PickDirectory(initialPath string) (*PickerResult, error) {
	// Try different GUI tools in order of preference
	// yad is prioritized over zenity as it's more stable and doesn't have the flickering issue
	tools := []string{"selectdir", "kdialog", "zenity"}

	for _, tool := range tools {
		result := tryPickDirectory(tool, initialPath)
		if result != nil {
			return result, nil
		}
	}

	// If no GUI tool is available, fall back to terminal input
	return PickDirectoryFallback(initialPath)
}

// PickDirectoryExisting opens a GUI dialog to select an existing directory
// Returns the selected path or an error if the user cancels or no tool is available
func PickDirectoryExisting(initialPath string) (*PickerResult, error) {
	// Try different GUI tools in order of preference
	tools := []string{"selectdir", "kdialog", "zenity"}

	for _, tool := range tools {
		result := tryPickDirectoryExisting(tool, initialPath)
		if result != nil {
			return result, nil
		}
	}

	// If no GUI tool is available, fall back to terminal input
	return PickDirectoryFallback(initialPath)
}

// tryPickDirectory attempts to use a specific GUI tool
func tryPickDirectory(tool, initialPath string) *PickerResult {
	// Check if the tool exists
	path, err := exec.LookPath(tool)
	if err != nil {
		return nil
	}

	// Build the command based on the tool
	var cmd *exec.Cmd
	switch tool {
	case "zenity":
		// Use --file-selection with --directory; avoid --filename to prevent flickering
		cmd = exec.Command(path, "--file-selection", "--directory", "--title=Select Install Directory")
		if initialPath != "" {
			// Set initial directory via environment variable instead
			cmd.Env = append(os.Environ(), "Zenity_FileSelector_Dir="+initialPath)
		}
	case "yad":
		// yad uses --directory flag directly for folder selection
		cmd = exec.Command(path, "--directory", "--title=Select Install Directory")
		if initialPath != "" {
			cmd.Args = append(cmd.Args, initialPath)
		}
	case "kdialog":
		cmd = exec.Command(path, "--getexistingdirectory", "--title=Select Install Directory")
		if initialPath != "" {
			cmd.Args = append(cmd.Args, initialPath)
		}
	}

	if cmd == nil {
		return nil
	}

	// Execute the command
	output, err := cmd.Output()
	if err != nil {
		// Check if user cancelled (zenity returns exit code 1)
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 1 {
				return &PickerResult{
					Success: false,
					Error:   fmt.Errorf("directory selection cancelled by user"),
				}
			}
		}
		return nil
	}

	pathResult := strings.TrimSpace(string(output))
	if pathResult == "" {
		return &PickerResult{
			Success: false,
			Error:   fmt.Errorf("no directory selected"),
		}
	}

	return &PickerResult{
		Path:    pathResult,
		Success: true,
	}
}

// tryPickDirectoryExisting attempts to use a specific GUI tool to select an existing directory
func tryPickDirectoryExisting(tool, initialPath string) *PickerResult {
	// Check if the tool exists
	path, err := exec.LookPath(tool)
	if err != nil {
		return nil
	}

	// Build the command based on the tool
	var cmd *exec.Cmd
	switch tool {
	case "zenity":
		// Use --file-selection with --directory for existing directory selection
		cmd = exec.Command(path, "--file-selection", "--directory", "--title=Select Bellum WINEPREFIX Directory to Uninstall")
		if initialPath != "" {
			cmd.Env = append(os.Environ(), "Zenity_FileSelector_Dir="+initialPath)
		}
	case "yad":
		cmd = exec.Command(path, "--directory", "--title=Select WINEPREFIX Directory")
		if initialPath != "" {
			cmd.Args = append(cmd.Args, initialPath)
		}
	case "kdialog":
		cmd = exec.Command(path, "--getexistingdirectory", "--title=Select WINEPREFIX Directory")
		if initialPath != "" {
			cmd.Args = append(cmd.Args, initialPath)
		}
	}

	if cmd == nil {
		return nil
	}

	// Execute the command
	output, err := cmd.Output()
	if err != nil {
		// Check if user cancelled (zenity returns exit code 1)
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 1 {
				return &PickerResult{
					Success: false,
					Error:   fmt.Errorf("directory selection cancelled by user"),
				}
			}
		}
		return nil
	}

	pathResult := strings.TrimSpace(string(output))
	if pathResult == "" {
		return &PickerResult{
			Success: false,
			Error:   fmt.Errorf("no directory selected"),
		}
	}

	return &PickerResult{
		Path:    pathResult,
		Success: true,
	}
}

// PickDirectoryFallback is a terminal-based fallback for directory selection
func PickDirectoryFallback(initialPath string) (*PickerResult, error) {
	fmt.Println()
	fmt.Println("No GUI directory picker found. Using terminal input.")
	fmt.Println()

	if initialPath != "" {
		fmt.Printf("Initial path: %s\n", initialPath)
	}

	fmt.Print("Enter the directory path for WINEPREFIX (or press Enter to use current directory): ")

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return &PickerResult{
			Success: false,
			Error:   fmt.Errorf("failed to read input: %w", err),
		}, nil
	}

	path := strings.TrimSpace(input)
	if path == "" {
		// Use current working directory
		path, err = os.Getwd()
		if err != nil {
			return &PickerResult{
				Success: false,
				Error:   fmt.Errorf("failed to get current directory: %w", err),
			}, nil
		}
	}

	// Normalize path
	path = strings.TrimSuffix(path, "/")

	return &PickerResult{
		Path:    path,
		Success: true,
	}, nil
}
