package query

import (
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/scheme/fake"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	testProxyName = "test-proxy"
	testProxyData = "test-proxy-value"

	testConfigMapName  = "test-config-map"
	testConfigMapValue = "test-config-map-value"

	testSecretName  = "test-secret"
	testSecretValue = "test-secret-value"
)

func TestDynakubeQuery(t *testing.T) {
	t.Run(`get proxy value`, testProxyValue)
	t.Run(`get proxy value from secret`, testProxyValueFrom)
	t.Run(`get trusted certificate authorities`, testTrustedCAs)
	t.Run(`get tls certificates`, testTlsCert)
}

func testProxyValue(t *testing.T) {
	query := DynakubeQuery{
		clt:       nil,
		namespace: "",
	}
	proxy, err := query.Proxy(&dynatracev1beta1.DynaKube{
		Spec: dynatracev1beta1.DynaKubeSpec{
			Proxy: &dynatracev1beta1.DynaKubeProxy{Value: testProxyData},
		},
	})

	assert.NoError(t, err)
	assert.Equal(t, testProxyData, proxy)

	proxy, err = query.Proxy(&dynatracev1beta1.DynaKube{})

	assert.NoError(t, err)
	assert.Equal(t, "", proxy)
}

func testProxyValueFrom(t *testing.T) {
	query := DynakubeQuery{
		clt: fake.NewClient(&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: testProxyName},
			Data: map[string][]byte{
				proxyKey: []byte(testProxyData),
			}}),
		namespace: "",
	}
	proxy, err := query.Proxy(&dynatracev1beta1.DynaKube{
		Spec: dynatracev1beta1.DynaKubeSpec{
			Proxy: &dynatracev1beta1.DynaKubeProxy{ValueFrom: testProxyName},
		},
	})

	assert.NoError(t, err)
	assert.Equal(t, testProxyData, proxy)

	query.clt = fake.NewClient()
	proxy, err = query.Proxy(&dynatracev1beta1.DynaKube{
		Spec: dynatracev1beta1.DynaKubeSpec{
			Proxy: &dynatracev1beta1.DynaKubeProxy{ValueFrom: testProxyName},
		},
	})

	assert.Error(t, err)
	assert.Equal(t, "", proxy)
}

func testTrustedCAs(t *testing.T) {
	query := DynakubeQuery{
		clt: fake.NewClient(&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: testConfigMapName},
			Data: map[string]string{
				trustedCAKey: testConfigMapValue,
			},
		}),
	}
	trustedCAs, err := query.TrustedCAs(&dynatracev1beta1.DynaKube{
		Spec: dynatracev1beta1.DynaKubeSpec{
			TrustedCAs: testConfigMapName,
		},
	})

	assert.NoError(t, err)
	assert.Equal(t, []byte(testConfigMapValue), trustedCAs)

	query.clt = fake.NewClient()
	trustedCAs, err = query.TrustedCAs(&dynatracev1beta1.DynaKube{
		Spec: dynatracev1beta1.DynaKubeSpec{
			TrustedCAs: testConfigMapName,
		},
	})

	assert.Error(t, err)
	assert.Empty(t, trustedCAs)

	trustedCAs, err = query.TrustedCAs(&dynatracev1beta1.DynaKube{
		Spec: dynatracev1beta1.DynaKubeSpec{},
	})

	assert.NoError(t, err)
	assert.Empty(t, trustedCAs)
}

func testTlsCert(t *testing.T) {
	query := DynakubeQuery{
		clt: fake.NewClient(&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: testSecretName},
			Data: map[string][]byte{
				tlsCertKey: []byte(testSecretValue),
			}}),
		namespace: "",
	}
	tlsCert, err := query.TlsCert(&dynatracev1beta1.DynaKube{
		Spec: dynatracev1beta1.DynaKubeSpec{
			ActiveGate: dynatracev1beta1.ActiveGateSpec{
				Capabilities:  []dynatracev1beta1.CapabilityDisplayName{dynatracev1beta1.KubeMonCapability.DisplayName},
				TlsSecretName: testSecretName,
			},
		},
	})

	assert.NoError(t, err)
	assert.Equal(t, testSecretValue, tlsCert)

	query.clt = fake.NewClient()
	tlsCert, err = query.TlsCert(&dynatracev1beta1.DynaKube{
		Spec: dynatracev1beta1.DynaKubeSpec{
			ActiveGate: dynatracev1beta1.ActiveGateSpec{
				Capabilities:  []dynatracev1beta1.CapabilityDisplayName{dynatracev1beta1.KubeMonCapability.DisplayName},
				TlsSecretName: testSecretName,
			},
		},
	})

	assert.Error(t, err)
	assert.Empty(t, tlsCert)

	tlsCert, err = query.TlsCert(&dynatracev1beta1.DynaKube{})

	assert.NoError(t, err)
	assert.Empty(t, tlsCert)
}
