package core

import (
	"os"
	"path/filepath"
	"strings"
)

// ScopedWine manages a scoped wine environment for the installer lifecycle.
// It prepends the packaged wine bin directory to PATH and updates LD_LIBRARY_PATH
// so all wine commands use the packaged version. At Restore(), the original PATH
// is restored so the system continues working as before.
type ScopedWine struct {
	originalPath   string
	scopedPath     string
	wineBinDir     string
	origLDLib      string
	origLDLibSet   bool
}

// NewScopedWine creates a new ScopedWine that will scope the given wine bin
// directory. The original PATH is captured immediately.
func NewScopedWine(wineBinDir string) *ScopedWine {
	return &ScopedWine{
		originalPath: os.Getenv("PATH"),
		wineBinDir:   wineBinDir,
		origLDLib:    os.Getenv("LD_LIBRARY_PATH"),
		origLDLibSet: os.Getenv("LD_LIBRARY_PATH") != "",
	}
}

// Apply prepends the wine bin directory to PATH and updates LD_LIBRARY_PATH
// so that all subsequent commands (via RunCommand, etc.) use the packaged wine.
func (s *ScopedWine) Apply() {
	// Prepend wine bin dir to PATH
	s.scopedPath = s.wineBinDir + string(filepath.ListSeparator) + s.originalPath
	os.Setenv("PATH", s.scopedPath)

	// Add wine lib dirs to LD_LIBRARY_PATH
	// Bellum wine uses lib/wine/x86_64-unix and lib/wine/x86_64-windows
	wineBaseDir := filepath.Dir(s.wineBinDir)
	libPaths := []string{
		filepath.Join(wineBaseDir, "lib", "wine", "x86_64-unix"),
		filepath.Join(wineBaseDir, "lib", "wine", "x86_64-windows"),
	}

	validPaths := []string{}
	for _, p := range libPaths {
		if _, err := os.Stat(p); err == nil {
			validPaths = append(validPaths, p)
		}
	}
	if len(validPaths) > 0 {
		newLDLib := strings.Join(validPaths, string(filepath.ListSeparator))
		if s.origLDLibSet {
			newLDLib = newLDLib + string(filepath.ListSeparator) + s.origLDLib
		}
		os.Setenv("LD_LIBRARY_PATH", newLDLib)
	}
}

// Restore restores the original PATH and LD_LIBRARY_PATH, undoing the scope.
// This should be called at the end of the installer lifecycle.
func (s *ScopedWine) Restore() {
	if s.scopedPath != "" {
		os.Setenv("PATH", s.originalPath)
	}
	if s.origLDLibSet {
		os.Setenv("LD_LIBRARY_PATH", s.origLDLib)
	} else {
		os.Unsetenv("LD_LIBRARY_PATH")
	}
}
