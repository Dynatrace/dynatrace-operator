package dynakube

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta5/dynakube/activegate"
	"github.com/stretchr/testify/assert"
)

func TestIsAGCertificateNeeded(t *testing.T) {
	t.Run("remote AG and no trustedCAs", func(t *testing.T) {
		dk := &DynaKube{
			Spec: DynaKubeSpec{},
		}
		assert.False(t, dk.IsAGCertificateNeeded())
	})
	t.Run("remote AG and trustedCAs", func(t *testing.T) {
		dk := &DynaKube{
			Spec: DynaKubeSpec{
				TrustedCAs: "test",
			},
		}
		assert.False(t, dk.IsAGCertificateNeeded())
	})
	t.Run("in-cluster AG and no trustedCAs", func(t *testing.T) {
		dk := &DynaKube{
			Spec: DynaKubeSpec{
				ActiveGate: activegate.Spec{
					TLSSecretName: "test-ag-cert",
					Capabilities: []activegate.CapabilityDisplayName{
						activegate.DynatraceAPICapability.DisplayName,
					},
				},
			},
		}
		assert.True(t, dk.IsAGCertificateNeeded())
	})
	t.Run("in-cluster AG and trustedCAs", func(t *testing.T) {
		dk := &DynaKube{
			Spec: DynaKubeSpec{
				ActiveGate: activegate.Spec{
					TLSSecretName: "test-ag-cert",
					Capabilities: []activegate.CapabilityDisplayName{
						activegate.DynatraceAPICapability.DisplayName,
					},
				},
				TrustedCAs: "test",
			},
		}
		assert.True(t, dk.IsAGCertificateNeeded())
	})
}

func TestIsCACertificateNeeded(t *testing.T) {
	t.Run("remote AG and no trustedCAs", func(t *testing.T) {
		dk := &DynaKube{
			Spec: DynaKubeSpec{},
		}
		assert.False(t, dk.IsCACertificateNeeded())
	})
	t.Run("remote AG and trustedCAs", func(t *testing.T) {
		dk := &DynaKube{
			Spec: DynaKubeSpec{
				TrustedCAs: "test",
			},
		}
		assert.True(t, dk.IsCACertificateNeeded())
	})
	t.Run("in-cluster AG and no trustedCAs", func(t *testing.T) {
		dk := &DynaKube{
			Spec: DynaKubeSpec{
				ActiveGate: activegate.Spec{
					TLSSecretName: "test-ag-cert",
					Capabilities: []activegate.CapabilityDisplayName{
						activegate.DynatraceAPICapability.DisplayName,
					},
				},
			},
		}
		assert.False(t, dk.IsCACertificateNeeded())
	})
	t.Run("in-cluster AG and trustedCAs", func(t *testing.T) {
		dk := &DynaKube{
			Spec: DynaKubeSpec{
				ActiveGate: activegate.Spec{
					TLSSecretName: "test-ag-cert",
					Capabilities: []activegate.CapabilityDisplayName{
						activegate.DynatraceAPICapability.DisplayName,
					},
				},
				TrustedCAs: "test",
			},
		}
		assert.False(t, dk.IsCACertificateNeeded())
	})
}
