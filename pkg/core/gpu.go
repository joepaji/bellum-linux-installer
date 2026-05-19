package core

import (
	"strings"
)

// DetectGPU detects the GPU type using glxinfo
// Returns the GPU type (e.g., "NVIDIA", "AMD", "Intel") or an error if detection fails
func DetectGPU() (string, error) {
	var output string
	err := RunCommand(RunModeCapture, []string{"glxinfo", "-B"}, nil, "", nil, &output)
	if err != nil {
		return "", err
	}

	renderer := parseRenderer(output)
	if renderer == "" {
		return "", nil
	}

	return classifyGPU(renderer), nil
}

func parseRenderer(output string) string {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(strings.ToLower(line), "opengl renderer") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return ""
}

func classifyGPU(renderer string) string {
	rendererLower := strings.ToLower(renderer)

	if strings.Contains(rendererLower, "nvidia") {
		return "NVIDIA"
	}
	if strings.Contains(rendererLower, "amd") || strings.Contains(rendererLower, "radeon") {
		return "AMD"
	}
	if strings.Contains(rendererLower, "intel") {
		return "Intel"
	}

	// Return the first word of the renderer as a fallback
	parts := strings.Fields(renderer)
	if len(parts) > 0 {
		return parts[0]
	}

	return ""
}
