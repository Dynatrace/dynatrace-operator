// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

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

func DNS1035LabelViolation(subject string) string {
	return subject + " violates DNS-1035, " +
		// This error message is copied from the apimachinery validation code.
		`a DNS-1035 label must consist of lower case alphanumeric characters or '-', start with an alphabetic character, and end with an alphanumeric character (e.g. 'my-name',  or 'abc-123', regex used for validation is '[a-z]([-a-z0-9]*[a-z0-9])?')`
}
