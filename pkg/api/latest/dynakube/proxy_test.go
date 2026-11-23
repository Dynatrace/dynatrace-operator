// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package dynakube_test

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

const (
	testProxyName = "test-proxy"
	testProxyData = "test-proxy-value"
)

func TestProxy(t *testing.T) {
	t.Run("get proxy value", proxyValueTester)
	t.Run("get proxy value from secret", proxyValueFromTester)
}

func proxyValueTester(t *testing.T) {
	dk := dynakube.DynaKube{
		Spec: dynakube.DynaKubeSpec{
			Proxy: &types.ValueSource{Value: testProxyData},
		},
	}
	proxy, err := dk.Proxy(t.Context(), nil)
	require.NoError(t, err)
	assert.Equal(t, testProxyData, proxy)

	emptyDK := dynakube.DynaKube{}
	proxy, err = emptyDK.Proxy(t.Context(), nil)
	require.NoError(t, err)
	assert.Empty(t, proxy)
}

func proxyValueFromTester(t *testing.T) {
	kubeReader := fake.NewClient(&corev1.Secret{
		Name: testProxyName,
		Data: map[string][]byte{
			dynakube.ProxyKey: []byte(testProxyData),
		}})
	dk := dynakube.DynaKube{
		Spec: dynakube.DynaKubeSpec{
			Proxy: &types.ValueSource{ValueFrom: testProxyName},
		},
	}
	proxy, err := dk.Proxy(t.Context(), kubeReader)
	require.NoError(t, err)
	assert.Equal(t, testProxyData, proxy)

	kubeReader = fake.NewClient()
	proxy, err = dk.Proxy(t.Context(), kubeReader)
	require.Error(t, err)
	assert.Empty(t, proxy)
}

func TestNoProxyFF(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		dk := dynakube.DynaKube{}
		noProxyHosts := dk.GetNoProxyHosts()
		assert.Empty(t, noProxyHosts)
	})
	t.Run("value", func(t *testing.T) {
		dk := dynakube.DynaKube{
			Annotations: map[string]string{
				exp.NoProxyKey: "test1 , test2",
			},
		}
		noProxyHosts := dk.GetNoProxyHosts()
		assert.Equal(t, []string{"test1", "test2"}, noProxyHosts)
	})
	t.Run("append empty no-proxy ff", func(t *testing.T) {
		dk := dynakube.DynaKube{}
		values := []string{"test"}
		// no IsEmpty guard is needed
		noProxyHosts := dk.GetNoProxyHosts()
		values = append(values, noProxyHosts...)
		assert.Equal(t, []string{"test"}, values)
	})
}
