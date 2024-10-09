package test

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube/activegate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	testConfigMapName  = "test-config-map"
	testConfigMapValue = "test-config-map-value"

	testSecretName  = "test-secret"
	testSecretValue = "test-secret-value"
)

func TestCerts(t *testing.T) {
	t.Run(`get trusted certificate authorities`, trustedCAsTester)
	t.Run(`get no tls certificates`, activeGateTlsNoCertificateTester)
	activeGateTLSCertificate(t)
}

func trustedCAsTester(t *testing.T) {
	kubeReader := fake.NewClient(&corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: testConfigMapName},
		Data: map[string]string{
			dynakube.TrustedCAKey: testConfigMapValue,
		},
	})
	dk := dynakube.DynaKube{
		Spec: dynakube.DynaKubeSpec{
			TrustedCAs: testConfigMapName,
		},
	}
	trustedCAs, err := dk.TrustedCAs(context.TODO(), kubeReader)
	require.NoError(t, err)
	assert.Equal(t, []byte(testConfigMapValue), trustedCAs)

	kubeReader = fake.NewClient()
	trustedCAs, err = dk.TrustedCAs(context.TODO(), kubeReader)

	require.Error(t, err)
	assert.Empty(t, trustedCAs)

	emptyDk := dynakube.DynaKube{}
	trustedCAs, err = emptyDk.TrustedCAs(context.TODO(), kubeReader)
	require.NoError(t, err)
	assert.Empty(t, trustedCAs)
}

func activeGateTlsNoCertificateTester(t *testing.T) {
	dk := dynakube.DynaKube{
		Spec: dynakube.DynaKubeSpec{
			ActiveGate: activegate.Spec{
				Capabilities:  []activegate.CapabilityDisplayName{activegate.KubeMonCapability.DisplayName},
				TlsSecretName: testSecretName,
			},
		},
	}

	kubeReader := fake.NewClient()
	tlsCert, err := dk.ActiveGateTLSCert(context.TODO(), kubeReader)

	require.Error(t, err)
	assert.Empty(t, tlsCert)

	emptyDk := dynakube.DynaKube{}
	tlsCert, err = emptyDk.ActiveGateTLSCert(context.TODO(), kubeReader)

	require.NoError(t, err)
	assert.Empty(t, tlsCert)
}

func activeGateTLSCertificate(t *testing.T) {
	testFunc := func(t *testing.T, data map[string][]byte) {
		kubeReader := fake.NewClient(&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: testSecretName},
			Data:       data,
		})

		dk := dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				ActiveGate: activegate.Spec{
					Capabilities:  []activegate.CapabilityDisplayName{activegate.KubeMonCapability.DisplayName},
					TlsSecretName: testSecretName,
				},
			},
		}
		tlsCert, err := dk.ActiveGateTLSCert(context.TODO(), kubeReader)

		require.NoError(t, err)
		assert.Equal(t, testSecretValue, string(tlsCert))
	}

	t.Run("get tls certificates from server.crt", func(t *testing.T) {
		testFunc(t, map[string][]byte{
			dynakube.TLSCertKey: []byte(testSecretValue),
		})
	})
}
