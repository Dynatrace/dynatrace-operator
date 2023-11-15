package dtotel

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/net/context"
)

func TestSetupTraces(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		resource, err := newResource("testOtel")
		require.NoError(t, err)

		tracerProvider, shutdownFn, err := setupTracesWithOtlp(context.Background(), resource, "abc12345.dynatrace.com", "dtabc.abc.abc")
		require.NoError(t, err)

		assert.NotNil(t, shutdownFn)
		assert.NotNil(t, tracerProvider)
	})
	t.Run("do not use OpenTelementry", func(t *testing.T) {
		resource, err := newResource("testOtel")
		require.NoError(t, err)

		tracerProvider, shutdownFn, err := setupTracesWithOtlp(context.Background(), resource, "", "")
		require.Error(t, err)

		assert.Nil(t, shutdownFn)
		assert.Nil(t, tracerProvider)
	})
}

func TestStartSpan(t *testing.T) {
	t.Run("nil tracer", func(t *testing.T) {
		ctx, span := StartSpan(context.Background(), trace.Tracer(nil))

		assert.IsType(t, noopSpan{}, span)
		assert.Equal(t, context.Background(), ctx)
	})
}

func TestGetCaller(t *testing.T) {
	// Call getCaller with stack depth of 1
	caller := getCaller(1)

	t.Logf("calloer=%s", caller)

	// Assert caller contains function name
	assert.Contains(t, caller, "TestGetCaller")

	// Assert caller contains file name
	assert.Contains(t, caller, "traces_test.go")

	// Assert caller contains line number
	assert.Regexp(t, `\(\w+_test.go:\d+\)`, caller)
}
