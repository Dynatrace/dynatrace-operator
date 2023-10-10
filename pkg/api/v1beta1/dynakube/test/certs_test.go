package test

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	"github.com/stretchr/testify/assert"
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
	t.Run(`get tls certificates`, activeGateTlsCertTester)
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
	assert.NoError(t, err)
	assert.Equal(t, []byte(testConfigMapValue), trustedCAs)

	kubeReader = fake.NewClient()
	trustedCAs, err = dk.TrustedCAs(context.TODO(), kubeReader)

	assert.Error(t, err)
	assert.Empty(t, trustedCAs)

	emptyDk := dynakube.DynaKube{}
	trustedCAs, err = emptyDk.TrustedCAs(context.TODO(), kubeReader)
	assert.NoError(t, err)
	assert.Empty(t, trustedCAs)
}

func activeGateTlsCertTester(t *testing.T) {
	kubeReader := fake.NewClient(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: testSecretName},
		Data: map[string][]byte{
			dynakube.TlsCertKey: []byte(testSecretValue),
		}})

	dk := dynakube.DynaKube{
		Spec: dynakube.DynaKubeSpec{
			ActiveGate: dynakube.ActiveGateSpec{
				Capabilities:  []dynakube.CapabilityDisplayName{dynakube.KubeMonCapability.DisplayName},
				TlsSecretName: testSecretName,
			},
		},
	}
	tlsCert, err := dk.ActiveGateTlsCert(context.TODO(), kubeReader)

	assert.NoError(t, err)
	assert.Equal(t, testSecretValue, tlsCert)

	kubeReader = fake.NewClient()
	tlsCert, err = dk.ActiveGateTlsCert(context.TODO(), kubeReader)

	assert.Error(t, err)
	assert.Empty(t, tlsCert)

	emptyDk := dynakube.DynaKube{}
	tlsCert, err = emptyDk.ActiveGateTlsCert(context.TODO(), kubeReader)

	assert.NoError(t, err)
	assert.Empty(t, tlsCert)
}
