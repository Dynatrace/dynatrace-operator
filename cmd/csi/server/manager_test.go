package server

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme"
	"github.com/stretchr/testify/assert"
)

func TestCsiDriverManagerProvider(t *testing.T) {
	t.Run("is instantiable", func(t *testing.T) {
		csiManagerProvider := newCsiDriverManagerProvider()
		assert.NotNil(t, csiManagerProvider)
	})
	t.Run("creates options", func(t *testing.T) {
		csiManagerProvider := csiDriverManagerProvider{}

		options := csiManagerProvider.createOptions("namespace")

		assert.NotNil(t, options)
		assert.Contains(t, options.Cache.DefaultNamespaces, "namespace")
		assert.Equal(t, scheme.Scheme, options.Scheme)
		assert.Equal(t, metricsBindAddress, options.Metrics.BindAddress)

		assert.Equal(t, "", options.HealthProbeBindAddress)
	})
}
