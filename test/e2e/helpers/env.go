//go:build e2e

package helpers

import "os"

func EnvOrDefault(envVar, defaultValue string) string {
	value := os.Getenv(envVar)
	if value == "" {
		return defaultValue
	}

	return value
}
