package envvars

import (
	"context"
	"os"
	"strconv"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
)

func GetDuration(ctx context.Context, varName string, defaultValue time.Duration) time.Duration {
	envValue := os.Getenv(varName)
	if envValue == "" {
		return defaultValue
	}

	parsed, err := time.ParseDuration(envValue)
	if err != nil {
		_, log := logd.NewFromContext(ctx, "envvars")
		log.Info("invalid duration value, using default", "env", varName, "value", envValue, "default", defaultValue)

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
