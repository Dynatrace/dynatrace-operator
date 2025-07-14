package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContainerExclusionAnnotations(t *testing.T) {
	annoations := map[string]string{
		"container.inject.dynatrace.com/falsebar": "false",
		"container.inject.dynatrace.com/truebar":  "true",
	}

	tests := []struct {
		name     string
		expected bool
	}{
		{
			name:     "falsebar",
			expected: true,
		},
		{
			name:     "truebar",
			expected: false,
		},
		{
			name:     "nobar",
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, checkInjectionAnnotation(annoations, test.name))
		})
	}
}
