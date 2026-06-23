package envvars

import (
	"os"
	"strconv"
	"time"
)

func GetBool(varName string, defaultValue bool) bool {
	envValue := os.Getenv(varName)
	if envValue != "" {
		parsedValue, err := strconv.ParseBool(envValue)
		if err != nil {
			return defaultValue
		}

		return parsedValue
	}

	return defaultValue
}

func GetDuration(varName string, defaultValue time.Duration) time.Duration {
	envValue := os.Getenv(varName)
	if envValue != "" {
		parsed, err := time.ParseDuration(envValue)
		if err == nil {
			return parsed
		}
	}

	return defaultValue
}
