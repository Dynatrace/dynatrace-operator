// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package dynakube_test

import (
	"encoding/json"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
)

const (
	testConfigMapName  = "test-config-map"
	testConfigMapValue = "test-config-map-value"

	testSecretName     = "test-secret"
	testSecretValue    = "test-secret-value"
	testSecretValueNew = "test-secret-value-new"
)

func TestCerts(t *testing.T) {
	t.Run("get trusted certificate authorities", trustedCAsTester)
	t.Run("get no tls certificates", activeGateTLSNoCertificateTester)
	activeGateTLSCertificate(t)
	t.Run("get activegate tls data", activeGateTLSDataTester)
}

func activeGateTLSDataTester(t *testing.T) {
	data := map[string][]byte{
		consts.TLSCrtDataName: []byte(testSecretValue),
		"tls.key":             []byte(testSecretValueNew),
	}
	dk := dynakube.DynaKube{
		Spec: dynakube.DynaKubeSpec{
			ActiveGate: activegate.Spec{
				Capabilities:  []activegate.CapabilityDisplayName{activegate.KubeMonCapability.DisplayName},
				TLSSecretName: testSecretName,
			},
		},
	}

	t.Run("returns whole secret data", func(t *testing.T) {
		kubeReader := fake.NewClient(&corev1.Secret{Name: testSecretName, Data: data})

		tlsData, err := dk.ActiveGateTLSData(t.Context(), kubeReader)

		require.NoError(t, err)

		expected, err := json.Marshal(data)
		require.NoError(t, err)
		assert.Equal(t, expected, tlsData)
	})
	t.Run("missing secret returns not found error", func(t *testing.T) {
		tlsData, err := dk.ActiveGateTLSData(t.Context(), fake.NewClient())

		require.Error(t, err)
		assert.True(t, k8serrors.IsNotFound(err))
		assert.Empty(t, tlsData)
	})
	t.Run("no tls configured returns empty data", func(t *testing.T) {
		emptyDK := dynakube.DynaKube{}

		tlsData, err := emptyDK.ActiveGateTLSData(t.Context(), fake.NewClient())

		require.NoError(t, err)
		assert.Empty(t, tlsData)
	})
}

func trustedCAsTester(t *testing.T) {
	kubeReader := fake.NewClient(&corev1.ConfigMap{
		Name: testConfigMapName,
		Data: map[string]string{
			dynakube.TrustedCAKey: testConfigMapValue,
		},
	})
	dk := dynakube.DynaKube{
		Spec: dynakube.DynaKubeSpec{
			TrustedCAs: testConfigMapName,
		},
	}
	trustedCAs, err := dk.TrustedCAs(t.Context(), kubeReader)
	require.NoError(t, err)
	assert.Equal(t, []byte(testConfigMapValue), trustedCAs)

	kubeReader = fake.NewClient()
	trustedCAs, err = dk.TrustedCAs(t.Context(), kubeReader)

	require.Error(t, err)
	assert.Empty(t, trustedCAs)

	emptyDK := dynakube.DynaKube{}
	trustedCAs, err = emptyDK.TrustedCAs(t.Context(), kubeReader)
	require.NoError(t, err)
	assert.Empty(t, trustedCAs)
}

func activeGateTLSNoCertificateTester(t *testing.T) {
	dk := dynakube.DynaKube{
		Spec: dynakube.DynaKubeSpec{
			ActiveGate: activegate.Spec{
				Capabilities:  []activegate.CapabilityDisplayName{activegate.KubeMonCapability.DisplayName},
				TLSSecretName: testSecretName,
			},
		},
	}

	kubeReader := fake.NewClient()
	tlsCert, err := dk.ActiveGateTLSCert(t.Context(), kubeReader)

	require.Error(t, err)
	assert.Empty(t, tlsCert)

	emptyDK := dynakube.DynaKube{}
	tlsCert, err = emptyDK.ActiveGateTLSCert(t.Context(), kubeReader)

	require.NoError(t, err)
	assert.Empty(t, tlsCert)
}

func activeGateTLSCertificate(t *testing.T) {
	testFunc := func(t *testing.T, data map[string][]byte) {
		kubeReader := fake.NewClient(&corev1.Secret{
			Name: testSecretName,
			Data: data,
		})

		dk := dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				ActiveGate: activegate.Spec{
					Capabilities:  []activegate.CapabilityDisplayName{activegate.KubeMonCapability.DisplayName},
					TLSSecretName: testSecretName,
				},
			},
		}
		tlsCert, err := dk.ActiveGateTLSCert(t.Context(), kubeReader)

		require.NoError(t, err)
		assert.Equal(t, testSecretValue, string(tlsCert))
	}

	t.Run("get tls certificates from server.crt", func(t *testing.T) {
		testFunc(t, map[string][]byte{
			consts.TLSServerCrtDataName: []byte(testSecretValue),
		})
	})

	t.Run("get tls certificates from tls.crt", func(t *testing.T) {
		testFunc(t, map[string][]byte{
			// prioritize tls.crt over server.crt
			consts.TLSCrtDataName:       []byte(testSecretValue),
			consts.TLSServerCrtDataName: []byte(testSecretValueNew),
		})
	})
}
