// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package k8ssecuritycontext

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			got, err := GetValidatedRunAsID(tt.value)
			assert.Equal(t, tt.expected, got)

			if tt.value == "" {
				require.NoError(t, err)
			} else {
				assert.Equal(t, tt.expected == nil, err != nil)
			}
		})
	}
}

func TestIsNonRoot(t *testing.T) {
	const nonRootID int64 = 1000

	tests := []struct {
		name     string
		user     *int64
		group    *int64
		expected *bool
	}{
		{name: "nil user and nil group", expected: nil},
		{name: "root user and root group", user: new(RootUser), group: new(RootGroup), expected: new(false)},
		{name: "non-root user and non-root group", user: new(nonRootID), group: new(nonRootID), expected: new(true)},
		{name: "root user and nil group (OCP case)", user: new(RootUser), expected: new(false)},
		{name: "root user and non-root group", user: new(RootUser), group: new(nonRootID), expected: new(false)},
		{name: "non-root user and nil group (OCP case)", user: new(nonRootID), expected: new(true)},
		{name: "non-root user and root group, user takes precedence", user: new(nonRootID), group: new(RootGroup), expected: new(true)},
		{name: "nil user and root group (edge case)", group: new(RootGroup), expected: new(false)},
		{name: "nil user and non-root group (edge case)", group: new(nonRootID), expected: new(true)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, IsNonRoot(tt.user, tt.group))
		})
	}
}
