package envvars

import (
	"os"
	"strconv"
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
