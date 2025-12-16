package validation

import (
	"fmt"
	"strings"
)

func SumErrors(validationErrors []string, customResourceType string) string {
	var summedErrors strings.Builder
	summedErrors.WriteString(fmt.Sprintf("\n%d error(s) found in the %s", len(validationErrors), customResourceType))

	for i, errMsg := range validationErrors {
		summedErrors.WriteString(fmt.Sprintf("\n %d. %s", i+1, errMsg))
	}

	return summedErrors.String()
}
