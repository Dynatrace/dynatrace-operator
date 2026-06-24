package envvars

import (
	"os"
	"strconv"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
)

func GetDuration(varName string, defaultValue time.Duration) time.Duration {
	envValue := os.Getenv(varName)
	if envValue == "" {
		return defaultValue
	}

	parsed, err := time.ParseDuration(envValue)
	if err != nil {
		logd.Get().WithName("envvars").Info("invalid duration value, using default", "env", varName, "value", envValue, "default", defaultValue)

		return defaultValue
	}

	return parsed
}

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
