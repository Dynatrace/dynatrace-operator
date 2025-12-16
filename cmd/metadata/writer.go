package metadata

import (
	"strings"

	"github.com/pkg/errors"
)

func parseAttributes(attributesFlag string) (string, error) {
	if attributesFlag == "" {
		return "", errors.New("node-attributes flag cannot be empty")
	}

	pairs := strings.Split(attributesFlag, ",")
	validPairs := make([]string, 0, len(pairs))

	for _, pair := range pairs {
		trimmed := strings.TrimSpace(pair)
		if trimmed == "" {
			continue
		}

		if !strings.Contains(trimmed, "=") {
			return "", errors.Errorf("invalid attribute format: %s (expected key=value)", trimmed)
		}

		validPairs = append(validPairs, trimmed)
	}

	if len(validPairs) == 0 {
		return "", errors.New("no valid attributes found in node-attributes flag")
	}

	return strings.Join(validPairs, "\n") + "\n", nil
}
