package core

import (
	"fmt"
	"os"
	"path/filepath"
)

// PrintInstallerSummary prints the installer summary after prechecks complete.
// This mirrors the bash version's print_installer_summary function.
func PrintInstallerSummary(protonVer, wineVer, winetricksVer, vkd3dVer, dxvkVer,
	wineprefix, launcherInstallerPath, gpuType string, workdir string) {

	// Determine launcher summary
	launcherSummary := "(will be downloaded)"
	if launcherInstallerPath != "" {
		launcherSummary = launcherInstallerPath
	}

	// Print the summary header
	fmt.Printf("%s======================================================\n", ColorBoldCyan)
	fmt.Printf("#              Bellum Installer Summary              #\n")
	fmt.Printf("======================================================%s\n\n", ColorReset)

	// Print version information
	fmt.Printf("%s   Proton Version%s:   %s%s%s%s\n", ColorBoldCyan, ColorReset, ColorBold, protonVer, ColorReset, ColorReset)
	fmt.Printf("%s     Wine Version%s:   %s%s%s%s (Stable)\n", ColorBoldCyan, ColorReset, ColorBold, wineVer, ColorReset, ColorReset)
	fmt.Printf("%s   Winetricks Ver%s:   %s%s%s%s\n", ColorBoldCyan, ColorReset, ColorBold, winetricksVer, ColorReset, ColorReset)
	fmt.Printf("%s        VKD3D Ver%s:   %s%s%s%s\n", ColorBoldCyan, ColorReset, ColorBold, vkd3dVer, ColorReset, ColorReset)
	fmt.Printf("%s         DXVK Ver%s:   %s%s%s%s\n\n", ColorBoldCyan, ColorReset, ColorBold, dxvkVer, ColorReset, ColorReset)

	// Print paths and configuration
	fmt.Printf("%s       WINEPREFIX%s:   %s%s%s%s\n", ColorBoldCyan, ColorReset, ColorBold, Colorize(wineprefix, ColorBoldYellow), ColorReset, ColorReset)
	fmt.Printf("%s Bellum Installer%s:   %s%s%s%s\n", ColorBoldCyan, ColorReset, ColorBold, launcherSummary, ColorReset, ColorReset)
	fmt.Printf("\n%s         GPU TYPE%s:   %s%s%s%s\n\n", ColorBoldCyan, ColorReset, ColorBold, gpuType, ColorReset, ColorReset)

	// Print note
	fmt.Printf("%sNOTE:%s The game will be installed into the specified WINEPREFIX path.\n", ColorBoldYellow, ColorReset)
}

// ConfirmProceed prompts the user to confirm they want to continue with the installation.
// Returns true if user confirms, false otherwise.
func ConfirmProceed() bool {
	fmt.Println()
	reader := NewReader()

	fmt.Print("Do you want to proceed with the installation? (Y/n): ")
	input, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("Error reading input:", err)
		return false
	}

	input = trimSpace(input)

	// Empty input defaults to yes
	if input == "" {
		return true
	}

	inputLower := stringsToLower(input)
	if inputLower == "y" || inputLower == "yes" {
		return true
	}
	if inputLower == "n" || inputLower == "no" {
		fmt.Printf("%sInstallation cancelled by user%s\n", ColorBoldRed, ColorReset)
		return false
	}

	fmt.Printf("%sInvalid input: '%s'%s\n", ColorBoldRed, input, ColorReset)
	fmt.Println("Please enter 'y' or 'Y' to proceed, 'n' or 'N' to cancel, or press Enter to proceed (default: yes)")
	return ConfirmProceed()
}

// Helper functions (mimicking bash behavior)

func trimSpace(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}

func stringsToLower(s string) string {
	result := ""
	for _, c := range s {
		if c >= 'A' && c <= 'Z' {
			c = c + 'a' - 'A'
		}
		result += string(c)
	}
	return result
}

// GetLauncherSummary returns the launcher installer path or "(will be downloaded)"
func GetLauncherSummary(launcherInstallerPath string) string {
	if launcherInstallerPath != "" {
		return launcherInstallerPath
	}
	return "(will be downloaded)"
}

// GetLauncherPath returns the absolute path to the launcher installer if provided
func GetLauncherPath(launcherInstallerPath string) (string, error) {
	if launcherInstallerPath == "" {
		return "", nil
	}

	// Resolve to absolute path if relative
	if !filepath.IsAbs(launcherInstallerPath) {
		absPath, err := filepath.Abs(launcherInstallerPath)
		if err != nil {
			return "", fmt.Errorf("failed to resolve launcher installer path: %w", err)
		}
		launcherInstallerPath = absPath
	}

	// Check if file exists
	if _, err := os.Stat(launcherInstallerPath); os.IsNotExist(err) {
		return "", fmt.Errorf("launcher installer not found: %s", launcherInstallerPath)
	}

	return launcherInstallerPath, nil
}
