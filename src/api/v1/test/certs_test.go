package test

import (
	"context"
	"testing"

	dynatracev1 "github.com/Dynatrace/dynatrace-operator/src/api/v1"
	"github.com/Dynatrace/dynatrace-operator/src/scheme/fake"
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
			dynatracev1.TrustedCAKey: testConfigMapValue,
		},
	})
	dk := dynatracev1.DynaKube{
		Spec: dynatracev1.DynaKubeSpec{
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

	emptyDk := dynatracev1.DynaKube{}
	trustedCAs, err = emptyDk.TrustedCAs(context.TODO(), kubeReader)
	assert.NoError(t, err)
	assert.Empty(t, trustedCAs)
}

func activeGateTlsCertTester(t *testing.T) {
	kubeReader := fake.NewClient(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: testSecretName},
		Data: map[string][]byte{
			dynatracev1.TlsCertKey: []byte(testSecretValue),
		}})

	dk := dynatracev1.DynaKube{
		Spec: dynatracev1.DynaKubeSpec{
			ActiveGate: dynatracev1.ActiveGateSpec{
				Capabilities:  []dynatracev1.CapabilityDisplayName{dynatracev1.KubeMonCapability.DisplayName},
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

	emptyDk := dynatracev1.DynaKube{}
	tlsCert, err = emptyDk.ActiveGateTlsCert(context.TODO(), kubeReader)

	assert.NoError(t, err)
	assert.Empty(t, tlsCert)
}
