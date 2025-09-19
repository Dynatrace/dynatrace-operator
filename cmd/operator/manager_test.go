package operator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetControllerAddFuncs(t *testing.T) {
	t.Run("without OLM", func(t *testing.T) {
		funcs := getControllerAddFuncs(false)

		assert.Len(t, funcs, 4) // dk, ec, nodes, certs
	})

	t.Run("with OLM", func(t *testing.T) {
		funcs := getControllerAddFuncs(true)

		assert.Len(t, funcs, 3) // dk, ec, nodes
	})

	t.Run("without HostAvailabilityDetectionEnvVar", func(t *testing.T) {
		funcs := getControllerAddFuncs(true)

		assert.Len(t, funcs, 2) // dk, ec
	})
}
