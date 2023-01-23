//go:build e2e

package environment

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

func TestEnvironmentCleanup(t *testing.T) {
	tests := []struct {
		title           string
		envVarValue     string
		testFailed      bool
		cleanupExpected bool
	}{
		{"always cleanup after success", "always", false, true},
		{"always cleanup after fail", "always", true, true},
		{"never cleanup after success", "never", false, false},
		{"never cleanup after fail", "never", true, false},
		{"cleanup on success", "onSuccess", false, true},
		{"do not cleanup on fail", "onSuccess", true, false},
	}

	for _, test := range tests {
		t.Run(test.title, func(t *testing.T) {
			t.Setenv(cleanupEnvVar, test.envVarValue)

			env := Get()
			env.t = StubbedT{
				failed: test.testFailed,
				t:      t,
			}

			cleanupFunctionCalled := false
			cleanupWrapper := env.wrapCleanupFunction(func(context.Context, *envconf.Config, *testing.T) (context.Context, error) {
				cleanupFunctionCalled = true
				return nil, nil
			})
			_, err := cleanupWrapper(nil, nil, nil)

			assert.Nil(t, err)
			assert.Equal(t, test.cleanupExpected, cleanupFunctionCalled)
		})
	}
}

type StubbedT struct {
	failed bool
	t      *testing.T
}

func (m StubbedT) Failed() bool {
	return m.failed
}

func (m StubbedT) Errorf(format string, args ...interface{}) {
	m.t.Errorf(format, args...)
}
