package certificates

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube/activegate"
	"github.com/stretchr/testify/assert"
)

func TestIsAGCertificateNeeded(t *testing.T) {
	t.Run("remote AG and no trustedCAs", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{},
		}
		assert.False(t, IsAGCertificateNeeded(dk))
	})
	t.Run("remote AG and trustedCAs", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				TrustedCAs: "test",
			},
		}
		assert.False(t, IsAGCertificateNeeded(dk))
	})
	t.Run("in-cluster AG and no trustedCAs", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				ActiveGate: activegate.Spec{
					TlsSecretName: "test-ag-cert",
					Capabilities: []activegate.CapabilityDisplayName{
						activegate.DynatraceApiCapability.DisplayName,
					},
				},
			},
		}
		assert.True(t, IsAGCertificateNeeded(dk))
	})
	t.Run("in-cluster AG and trustedCAs", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				ActiveGate: activegate.Spec{
					TlsSecretName: "test-ag-cert",
					Capabilities: []activegate.CapabilityDisplayName{
						activegate.DynatraceApiCapability.DisplayName,
					},
				},
				TrustedCAs: "test",
			},
		}
		assert.True(t, IsAGCertificateNeeded(dk))
	})
}

func TestIsCACertificateNeeded(t *testing.T) {
	t.Run("remote AG and no trustedCAs", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{},
		}
		assert.False(t, IsCACertificateNeeded(dk))
	})
	t.Run("remote AG and trustedCAs", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				TrustedCAs: "test",
			},
		}
		assert.True(t, IsCACertificateNeeded(dk))
	})
	t.Run("in-cluster AG and no trustedCAs", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				ActiveGate: activegate.Spec{
					TlsSecretName: "test-ag-cert",
					Capabilities: []activegate.CapabilityDisplayName{
						activegate.DynatraceApiCapability.DisplayName,
					},
				},
			},
		}
		assert.False(t, IsCACertificateNeeded(dk))
	})
	t.Run("in-cluster AG and trustedCAs", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				ActiveGate: activegate.Spec{
					TlsSecretName: "test-ag-cert",
					Capabilities: []activegate.CapabilityDisplayName{
						activegate.DynatraceApiCapability.DisplayName,
					},
				},
				TrustedCAs: "test",
			},
		}
		assert.False(t, IsCACertificateNeeded(dk))
	})
}
