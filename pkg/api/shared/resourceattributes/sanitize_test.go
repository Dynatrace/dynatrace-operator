// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package resourceattributes

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_SanitizeKey(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{input: "k8s.pod.name", expected: "k8s.pod.name"},
		{input: "dt.application_container_hour", expected: "dt.application_container_hour"},
		{input: "dt.security_context", expected: "dt.security_context"},
		{input: "some/key", expected: "some_key"},
		{input: "key with spaces", expected: "key_with_spaces"},
		{input: "key@domain", expected: "key_domain"},
		{input: "/leading-slash", expected: "leading-slash"},
		{input: "...dots", expected: "dots"},
		{input: "trailing-slash/", expected: "trailing-slash"},
		{input: "dots...", expected: "dots"},
		{input: "///", expected: ""},
		{input: "", expected: ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, SanitizeKey(tt.input))
		})
	}
}
