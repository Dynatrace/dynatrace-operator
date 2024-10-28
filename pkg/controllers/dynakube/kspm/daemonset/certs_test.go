package daemonset

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/stretchr/testify/assert"
)

func getDynaKubeWithCerts(t *testing.T) dynakube.DynaKube {
	t.Helper()

	dk := dynakube.DynaKube{}
	dk.ActiveGate().Spec.TlsSecretName = "test"

	return dk
}

func TestNeedsCerts(t *testing.T) {
	t.Run("needs", func(t *testing.T) {
		assert.True(t, needsCerts(getDynaKubeWithCerts(t)))
	})
	t.Run("doesn't need", func(t *testing.T) {
		assert.False(t, needsCerts(dynakube.DynaKube{}))
	})
}
