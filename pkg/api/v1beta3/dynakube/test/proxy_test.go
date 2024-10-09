package test

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/value"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	testProxyName = "test-proxy"
	testProxyData = "test-proxy-value"
)

func TestProxy(t *testing.T) {
	t.Run(`get proxy value`, proxyValueTester)
	t.Run(`get proxy value from secret`, proxyValueFromTester)
}

func proxyValueTester(t *testing.T) {
	dk := dynakube.DynaKube{
		Spec: dynakube.DynaKubeSpec{
			Proxy: &value.Source{Value: testProxyData},
		},
	}
	proxy, err := dk.Proxy(context.TODO(), nil)
	require.NoError(t, err)
	assert.Equal(t, testProxyData, proxy)

	emptyDk := dynakube.DynaKube{}
	proxy, err = emptyDk.Proxy(context.TODO(), nil)
	require.NoError(t, err)
	assert.Equal(t, "", proxy)
}

func proxyValueFromTester(t *testing.T) {
	kubeReader := fake.NewClient(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: testProxyName},
		Data: map[string][]byte{
			dynakube.ProxyKey: []byte(testProxyData),
		}})
	dk := dynakube.DynaKube{
		Spec: dynakube.DynaKubeSpec{
			Proxy: &value.Source{ValueFrom: testProxyName},
		},
	}
	proxy, err := dk.Proxy(context.TODO(), kubeReader)
	require.NoError(t, err)
	assert.Equal(t, testProxyData, proxy)

	kubeReader = fake.NewClient()
	proxy, err = dk.Proxy(context.TODO(), kubeReader)
	require.Error(t, err)
	assert.Equal(t, "", proxy)
}
