// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package k8ssecuritycontext

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetValidatedRunAsID(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected *int64
	}{
		{name: "empty", value: "", expected: nil},
		{name: "zero", value: "0", expected: new(int64(0))},
		{name: "valid id", value: "1234", expected: new(int64(1234))},
		{name: "max int32", value: "2147483647", expected: new(int64(2147483647))},
		{name: "above int32 range", value: "2147483648", expected: nil},
		{name: "negative", value: "-1", expected: nil},
		{name: "not a number", value: "abc", expected: nil},
		{name: "with whitespace", value: " 12 ", expected: nil},
		{name: "decimal", value: "1.5", expected: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, GetValidatedRunAsID(t.Context(), tt.value))
		})
	}
}
