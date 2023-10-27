package otel

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetupMetrics(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		resource, err := newResource("testOtel")
		require.NoError(t, err)

		meterProvider, shutdownFn, err := setupMetricsWithOtlp(context.Background(), resource, "abc12345.dynatrace.com", "dtabc.abc.abc")
		require.NoError(t, err)

		assert.NotNil(t, shutdownFn)
		assert.NotNil(t, meterProvider)
	})
	t.Run("do not use OpenTelementry", func(t *testing.T) {
		resource, err := newResource("testOtel")
		require.NoError(t, err)

		meterProvider, shutdownFn, err := setupMetricsWithOtlp(context.Background(), resource, "", "")
		require.Error(t, err)

		assert.Nil(t, shutdownFn)
		assert.Nil(t, meterProvider)
	})
}
