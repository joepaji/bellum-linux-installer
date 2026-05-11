package packages

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// PatchProtonSettings patches the Proton user_settings.py file with GPU-specific settings
// settingsFile: path to the user_settings.py file (can be user_settings.sample.py)
// isAMD: true if the GPU is AMD, false otherwise
// isFSR41: true if using FSR 4.1 upgrade path, false for regular mode
func PatchProtonSettings(settingsFile string, isAMD bool, isFSR41 bool) error {
	if settingsFile == "" {
		return fmt.Errorf("settings file path is empty")
	}

	if _, err := os.Stat(settingsFile); os.IsNotExist(err) {
		return fmt.Errorf("settings file not found: %s", settingsFile)
	}

	// Determine the actual settings file path and target name
	fileDir := filepath.Dir(settingsFile)
	fileBase := filepath.Base(settingsFile)
	targetFile := filepath.Join(fileDir, "user_settings.py")

	// If the file is user_settings.sample.py, rename it to user_settings.py first
	if strings.HasSuffix(fileBase, ".sample.py") {
		if _, err := os.Stat(targetFile); err == nil {
			// user_settings.py already exists, use it
			settingsFile = targetFile
		} else {
			// Rename user_settings.sample.py to user_settings.py
			if err := os.Rename(settingsFile, targetFile); err != nil {
				return fmt.Errorf("failed to rename settings file: %w", err)
			}
			settingsFile = targetFile
		}
	}

	// Create a temporary file for the patched content
	tmpFile, err := os.CreateTemp(filepath.Dir(settingsFile), "user_settings.patched.*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpFilePath := tmpFile.Name()
	defer func() {
		tmpFile.Close()
		if err != nil {
			os.Remove(tmpFilePath)
		}
	}()

	// Build the desired settings map
	desired := make(map[string]string)

	// Unified settings for all GPUs using proton-cachyos
	desired["MALLOC_ARENA_MAX"] = "1"
	desired["PROTON_VKD3D_HEAP"] = "1"
	desired["VKD3D_CONFIG"] = "descriptor_heap"

	// AMD-specific settings (FSR4 upgrade)
	if isAMD {
		// Default mode uses "1", FSR4.1 mode uses "4.1.0"
		if isFSR41 {
			desired["PROTON_FSR4_UPGRADE"] = "4.1.0"
			desired["PROTON_FSR4_RDNA3_UPGRADE"] = "4.1.0"
		} else {
			desired["PROTON_FSR4_UPGRADE"] = "1"
			desired["PROTON_FSR4_RDNA3_UPGRADE"] = "1"
			desired["PROTON_MLFG_UPGRADE"] = "1"
		}
	} else {
		desired["PROTON_ENABLE_NVAPI"] = "1"
		desired["PROTON_DLSS_UPGRADE"] = "1"
		desired["PROTON_DXVK_D3D8"] = "1"
		desired["PROTON_NVIDIA_LIBS"] = "1"
	}

	// Read and process the settings file
	file, err := os.Open(settingsFile)
	if err != nil {
		return fmt.Errorf("failed to open settings file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	foundKeys := make(map[string]bool)
	inSettingsBlock := false

	for scanner.Scan() {
		line := scanner.Text()

		// Check if we're entering the settings block
		if strings.Contains(line, "user_settings = {") {
			inSettingsBlock = true
			fmt.Fprintln(tmpFile, line)
			continue
		}

		// Check if we're leaving the settings block (closing brace)
		if inSettingsBlock && strings.TrimSpace(line) == "}" {
			// Add any missing desired settings before the closing brace
			for key, value := range desired {
				if !foundKeys[key] {
					fmt.Fprintln(tmpFile, "    \""+key+"\": \""+value+"\",")
				}
			}
			inSettingsBlock = false
			fmt.Fprintln(tmpFile, line)
			continue
		}

		// Process lines within the settings block
		if inSettingsBlock {
			matched := false
			for key := range desired {
				// Check if this line matches the key we're looking for
				pattern := "\"" + key + "\":"
				if strings.Contains(line, pattern) {
					// Extract indentation
					indent := ""
					for _, ch := range line {
						if ch == ' ' || ch == '\t' {
							indent += string(ch)
						} else {
							break
						}
					}

					// Write the patched line
					fmt.Fprintf(tmpFile, "%s\"%s\": \"%s\",\n", indent, key, desired[key])
					foundKeys[key] = true
					matched = true
					break
				}
			}
			if matched {
				continue
			}
		}

		// Write the original line
		fmt.Fprintln(tmpFile, line)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading settings file: %w", err)
	}

	tmpFile.Close()

	// Verify the temp file has content
	if stat, err := os.Stat(tmpFilePath); err != nil || stat.Size() == 0 {
		os.Remove(tmpFilePath)
		return fmt.Errorf("failed to write patched settings")
	}

	// Get the original file's permissions before replacing
	origStat, err := os.Stat(settingsFile)
	if err != nil {
		return fmt.Errorf("failed to stat original settings file: %w", err)
	}
	origMode := origStat.Mode()

	// Move the temp file to replace the original
	if err := os.Rename(tmpFilePath, settingsFile); err != nil {
		os.Remove(tmpFilePath)
		return fmt.Errorf("failed to replace settings file: %w", err)
	}

	// Restore the original file permissions
	if err := os.Chmod(settingsFile, origMode); err != nil {
		return fmt.Errorf("failed to restore file permissions: %w", err)
	}

	return nil
}

// GetProtonUserSettingsPath returns the path to the Proton user_settings.py file
// if it exists (or user_settings.sample.py), or an empty string if not found
func GetProtonUserSettingsPath(protonDir string) string {
	settingsFile := filepath.Join(protonDir, "user_settings.py")
	if _, err := os.Stat(settingsFile); err == nil {
		return settingsFile
	}

	// Also check for user_settings.sample.py
	sampleFile := filepath.Join(protonDir, "user_settings.sample.py")
	if _, err := os.Stat(sampleFile); err == nil {
		return sampleFile
	}

	return ""
}
