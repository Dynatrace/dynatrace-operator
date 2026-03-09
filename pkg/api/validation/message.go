package validation

import (
	"fmt"
	"strings"
)

func SumErrors(validationErrors []string, customResourceType string) string {
	var summedErrors strings.Builder
	fmt.Fprintf(&summedErrors, "\n%d error(s) found in the %s", len(validationErrors), customResourceType)

	for i, errMsg := range validationErrors {
		fmt.Fprintf(&summedErrors, "\n %d. %s", i+1, errMsg)
	}

	return summedErrors.String()
}
