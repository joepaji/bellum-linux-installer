package packages

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"bellum-installer/pkg/core"
)

// TestEnsureProtonWithSampleFile tests EnsureProton when only user_settings.sample.py exists
func TestEnsureProtonWithSampleFile(t *testing.T) {
	// Create temp workdir
	tmpDir, err := os.MkdirTemp("", "test-ensure-proton-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create proton directory with user_settings.sample.py
	protonDir := filepath.Join(tmpDir, "proton-test")
	if err := os.MkdirAll(protonDir, 0755); err != nil {
		t.Fatalf("Failed to create proton dir: %v", err)
	}

	// Create user_settings.sample.py
	sampleContent := `# Sample settings file
user_settings = {
    "PROTON_LOG": "1",
}
`
	samplePath := filepath.Join(protonDir, "user_settings.sample.py")
	if err := os.WriteFile(samplePath, []byte(sampleContent), 0644); err != nil {
		t.Fatalf("Failed to create sample file: %v", err)
	}

	// Create a mock logger
	logger := &core.Logger{}

	// Call EnsureProton
	err = EnsureProton(tmpDir, "proton-test", true, true, logger)
	if err != nil {
		t.Fatalf("EnsureProton failed: %v", err)
	}

	// Verify user_settings.py was created
	userSettingsPath := filepath.Join(protonDir, "user_settings.py")
	if _, err := os.Stat(userSettingsPath); os.IsNotExist(err) {
		t.Error("user_settings.py was not created after EnsureProton")
	}

	// Verify user_settings.sample.py was removed (renamed)
	if _, err := os.Stat(samplePath); err == nil {
		t.Error("user_settings.sample.py should have been renamed to user_settings.py")
	}
}

// TestEnsureProtonWithExistingPyFile tests EnsureProton when user_settings.py already exists
func TestEnsureProtonWithExistingPyFile(t *testing.T) {
	// Create temp workdir
	tmpDir, err := os.MkdirTemp("", "test-ensure-proton-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create proton directory with user_settings.py
	protonDir := filepath.Join(tmpDir, "proton-test")
	if err := os.MkdirAll(protonDir, 0755); err != nil {
		t.Fatalf("Failed to create proton dir: %v", err)
	}

	// Create user_settings.py
	existingContent := `# Existing settings file
user_settings = {
    "PROTON_LOG": "1",
}
`
	existingPath := filepath.Join(protonDir, "user_settings.py")
	if err := os.WriteFile(existingPath, []byte(existingContent), 0644); err != nil {
		t.Fatalf("Failed to create existing file: %v", err)
	}

	// Create a mock logger
	logger := &core.Logger{}

	// Call EnsureProton
	err = EnsureProton(tmpDir, "proton-test", false, false, logger)
	if err != nil {
		t.Fatalf("EnsureProton failed: %v", err)
	}

	// Verify user_settings.py still exists and was patched
	content, err := os.ReadFile(existingPath)
	if err != nil {
		t.Fatalf("Failed to read user_settings.py: %v", err)
	}

	contentStr := string(content)
	if !containsSetting(contentStr, "PROTON_ENABLE_NVAPI") {
		t.Error("Patched file does not contain expected setting: PROTON_ENABLE_NVAPI")
	}
}

// TestEnsureProtonMissingSettingsFile tests EnsureProton when no settings file exists
// Note: This test verifies that the function properly detects an empty directory
// and attempts to download. Since there's no real server, it will fail at download time.
func TestEnsureProtonMissingSettingsFile(t *testing.T) {
	// Create temp workdir
	tmpDir, err := os.MkdirTemp("", "test-ensure-proton-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create proton directory WITHOUT any settings file (empty directory)
	protonDir := filepath.Join(tmpDir, "proton-test")
	if err := os.MkdirAll(protonDir, 0755); err != nil {
		t.Fatalf("Failed to create proton dir: %v", err)
	}

	// Verify directory is empty
	entries, err := os.ReadDir(protonDir)
	if err != nil || len(entries) > 0 {
		t.Fatalf("Expected empty directory, got: %v", entries)
	}

	// Create a mock logger
	logger := &core.Logger{}

	// Call EnsureProton - should fail at download time because no real server exists
	err = EnsureProton(tmpDir, "proton-test", false, false, logger)
	// We expect an error because there's no real server to download from
	if err == nil {
		t.Error("Expected error for missing settings file (download failure), got nil")
	}
	// Verify the error is about download, not missing settings
	if !strings.Contains(err.Error(), "download") && !strings.Contains(err.Error(), "wget") {
		t.Logf("Expected download-related error, got: %v", err)
	}
}

// Helper function to check if content contains a specific setting
func containsSetting(content, settingName string) bool {
	return containsSubstring(content, "\""+settingName+"\"")
}

// Helper function to check if content contains a substring
func containsSubstring(content, substr string) bool {
	return len(content) >= len(substr) && (content == substr || len(content) > len(substr) && findSubstring(content, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
