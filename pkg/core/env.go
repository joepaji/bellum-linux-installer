package core

import (
	"fmt"
	"os"
)

// GetRequiredEnvVar retrieves a required environment variable.
// Returns the value if set, or an error if not set.
func GetRequiredEnvVar(name string) (string, error) {
	if value := os.Getenv(name); value != "" {
		return value, nil
	}
	return "", fmt.Errorf("required environment variable %s not set", name)
}

// GetEnvVarOrDefault retrieves an environment variable with a fallback default.
// Returns the value if set, otherwise returns the default.
func GetEnvVarOrDefault(name, defaultValue string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return defaultValue
}
