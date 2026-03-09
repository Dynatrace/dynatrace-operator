package envvars

import (
	"math"
	"os"
	"strconv"
	"time"
)

const maxDurationMinutes = math.MaxInt64 / int64(time.Minute)

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

func GetDurationMinutes(varName string, defaultValue time.Duration) time.Duration {
	envValue := os.Getenv(varName)
	if envValue == "" {
		return defaultValue
	}

	parsedValue, err := strconv.ParseInt(envValue, 10, 64)
	if err != nil {
		return defaultValue
	}

	if parsedValue <= 0 {
		return defaultValue
	}

	if parsedValue > maxDurationMinutes {
		parsedValue = maxDurationMinutes
	}

	return time.Duration(parsedValue) * time.Minute
}
