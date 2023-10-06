package test

import (
	"context"
	"github.com/Dynatrace/dynatrace-operator/src/api/scheme/fake"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/api/v1beta1/dynakube"
	"github.com/stretchr/testify/assert"
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
			Proxy: &dynakube.DynaKubeProxy{Value: testProxyData},
		},
	}
	proxy, err := dk.Proxy(context.TODO(), nil)
	assert.NoError(t, err)
	assert.Equal(t, testProxyData, proxy)

	emptyDk := dynakube.DynaKube{}
	proxy, err = emptyDk.Proxy(context.TODO(), nil)
	assert.NoError(t, err)
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
			Proxy: &dynakube.DynaKubeProxy{ValueFrom: testProxyName},
		},
	}
	proxy, err := dk.Proxy(context.TODO(), kubeReader)
	assert.NoError(t, err)
	assert.Equal(t, testProxyData, proxy)

	kubeReader = fake.NewClient()
	proxy, err = dk.Proxy(context.TODO(), kubeReader)
	assert.Error(t, err)
	assert.Equal(t, "", proxy)
}
