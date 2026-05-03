package packages

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestGetProtonUserSettingsPath tests the GetProtonUserSettingsPath function
func TestGetProtonUserSettingsPath(t *testing.T) {
	tests := []struct {
		name          string
		filesToCreate []string
		expected      string
	}{
		{
			name:          "user_settings.py exists",
			filesToCreate: []string{"user_settings.py"},
			expected:      "user_settings.py",
		},
		{
			name:          "user_settings.sample.py exists",
			filesToCreate: []string{"user_settings.sample.py"},
			expected:      "user_settings.sample.py",
		},
		{
			name:          "both files exist",
			filesToCreate: []string{"user_settings.py", "user_settings.sample.py"},
			expected:      "user_settings.py",
		},
		{
			name:          "neither file exists",
			filesToCreate: []string{},
			expected:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory
			tmpDir, err := os.MkdirTemp("", "test-proton-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			// Create test files
			for _, file := range tt.filesToCreate {
				path := filepath.Join(tmpDir, file)
				if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
					t.Fatalf("Failed to create test file %s: %v", file, err)
				}
			}

			// Call function
			result := GetProtonUserSettingsPath(tmpDir)

			// Verify result
			if tt.expected == "" {
				if result != "" {
					t.Errorf("Expected  string, got %s", result)
				}
			} else {
				expectedPath := filepath.Join(tmpDir, tt.expected)
				if result != expectedPath {
					t.Errorf("Expected %s, got %s", expectedPath, result)
				}
			}
		})
	}
}

// TestPatchProtonSettingsWithSampleFile tests patching when user_settings.sample.py exists
func TestPatchProtonSettingsWithSampleFile(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "test-proton-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create user_settings.sample.py
	sampleContent := `# Sample settings file
user_settings = {
    "PROTON_LOG": "1",
}
`
	samplePath := filepath.Join(tmpDir, "user_settings.sample.py")
	if err := os.WriteFile(samplePath, []byte(sampleContent), 0644); err != nil {
		t.Fatalf("Failed to create sample file: %v", err)
	}

	// Call PatchProtonSettings
	err = PatchProtonSettings(samplePath, true, true)
	if err != nil {
		t.Fatalf("PatchProtonSettings failed: %v", err)
	}

	// Verify user_settings.py was created
	userSettingsPath := filepath.Join(tmpDir, "user_settings.py")
	if _, err := os.Stat(userSettingsPath); os.IsNotExist(err) {
		t.Error("user_settings.py was not created after patching")
	}

	// Verify user_settings.sample.py was removed (renamed)
	if _, err := os.Stat(samplePath); err == nil {
		t.Error("user_settings.sample.py should have been renamed")
	}

	// Verify the patched content contains the expected settings
	content, err := os.ReadFile(userSettingsPath)
	if err != nil {
		t.Fatalf("Failed to read user_settings.py: %v", err)
	}

	expectedSettings := []string{
		"PROTON_ENABLE_NVAPI",
		"PROTON_DLSS_UPGRADE",
		"MALLOC_ARENA_MAX",
		"PROTON_VKD3D_HEAP",
		"VKD3D_CONFIG",
		"PROTON_DXVK_D3D8",
		"PROTON_NVIDIA_LIBS",
		"PROTON_FSR4_UPGRADE",
		"PROTON_FSR4_RDNA3_UPGRADE",
	}

	contentStr := string(content)
	for _, setting := range expectedSettings {
		if !strings.Contains(contentStr, setting) {
			t.Errorf("Patched file does not contain expected setting: %s", setting)
		}
	}
}

// TestPatchProtonSettingsWithExistingPyFile tests patching when user_settings.py already exists
func TestPatchProtonSettingsWithExistingPyFile(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "test-proton-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create user_settings.py (not .sample.py)
	existingContent := `# Existing settings file
user_settings = {
    "PROTON_LOG": "1",
}
`
	existingPath := filepath.Join(tmpDir, "user_settings.py")
	if err := os.WriteFile(existingPath, []byte(existingContent), 0644); err != nil {
		t.Fatalf("Failed to create existing file: %v", err)
	}

	// Call PatchProtonSettings
	err = PatchProtonSettings(existingPath, false, false)
	if err != nil {
		t.Fatalf("PatchProtonSettings failed: %v", err)
	}

	// Verify the file still exists and was patched
	content, err := os.ReadFile(existingPath)
	if err != nil {
		t.Fatalf("Failed to read user_settings.py: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "PROTON_ENABLE_NVAPI") {
		t.Error("Patched file does not contain expected setting: PROTON_ENABLE_NVAPI")
	}
}

// TestPatchProtonSettingsMissingFile tests error handling when file doesn't exist
func TestPatchProtonSettingsMissingFile(t *testing.T) {
	err := PatchProtonSettings("/nonexistent/path/user_settings.py", false, false)
	if err == nil {
		t.Error("Expected error for missing file, got nil")
	}
}

// TestPatchProtonSettingsEmptyPath tests error handling when path is empty
func TestPatchProtonSettingsEmptyPath(t *testing.T) {
	err := PatchProtonSettings("", false, false)
	if err == nil {
		t.Error("Expected error for empty path, got nil")
	}
}

// TestExtractTarXZWithStrip tests the extraction function with a known tar.xz structure
func TestExtractTarXZWithStrip(t *testing.T) {
	// This test requires a real tar.xz file, so we skip it if none is available
	// In a real scenario, you'd create a test archive with the expected structure
	t.Skip("Skipping test requiring tar.xz file creation")
}
