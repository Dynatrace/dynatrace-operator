package validation

import "fmt"

func SumErrors(validationErrors []string, customResourceType string) string {
	summedErrors := fmt.Sprintf("\n%d error(s) found in the %s", len(validationErrors), customResourceType)
	for i, errMsg := range validationErrors {
		summedErrors += fmt.Sprintf("\n %d. %s", i+1, errMsg)
	}

	return summedErrors
}
