package troubleshoot

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestCheckProxySettings(t *testing.T) {
	t.Run("No proxy settings", func(t *testing.T) {
		t.Setenv("HTTP_PROXY", "")
		t.Setenv("HTTPS_PROXY", "")

		logOutput := runWithTestLogger(func(logger logd.Logger) {
			checkProxySettings(context.Background(), logger, nil, &dynakube.DynaKube{})
		})

		require.NotContains(t, logOutput, "Unexpected error")
		assert.NotContains(t, logOutput, "HTTP_PROXY")
		assert.NotContains(t, logOutput, "HTTPS_PROXY")
		assert.NotContains(t, logOutput, "Dynakube")
		assert.Contains(t, logOutput, "No proxy settings found.")
	})
	t.Run("HTTP_PROXY", func(t *testing.T) {
		t.Setenv("HTTP_PROXY", "foobar:1234")
		t.Setenv("HTTPS_PROXY", "")

		logOutput := runWithTestLogger(func(logger logd.Logger) {
			checkProxySettings(context.Background(), logger, nil, &dynakube.DynaKube{})
		})

		require.NotContains(t, logOutput, "Unexpected error")
		assert.Contains(t, logOutput, "HTTP_PROXY")
		assert.NotContains(t, logOutput, "HTTPS_PROXY")
		assert.NotContains(t, logOutput, "Dynakube")
		assert.NotContains(t, logOutput, "No proxy settings found.")
	})
	t.Run("HTTPS_PROXY", func(t *testing.T) {
		t.Setenv("HTTP_PROXY", "")
		t.Setenv("HTTPS_PROXY", "foobar:1234")

		logOutput := runWithTestLogger(func(logger logd.Logger) {
			checkProxySettings(context.Background(), logger, nil, &dynakube.DynaKube{})
		})

		require.NotContains(t, logOutput, "Unexpected error")
		assert.NotContains(t, logOutput, "HTTP_PROXY")
		assert.Contains(t, logOutput, "HTTPS_PROXY")
		assert.NotContains(t, logOutput, "Dynakube")
		assert.NotContains(t, logOutput, "No proxy settings found.")
	})
	t.Run("Dynakube proxy", func(t *testing.T) {
		t.Setenv("HTTP_PROXY", "")
		t.Setenv("HTTPS_PROXY", "")

		dk := *testNewDynakubeBuilder(testNamespace, testDynakube).
			withProxy("http://foobar:1234").
			build()

		logOutput := runWithTestLogger(func(logger logd.Logger) {
			checkProxySettings(context.Background(), logger, nil, &dk)
		})

		require.NotContains(t, logOutput, "Unexpected error")
		assert.NotContains(t, logOutput, "HTTP_PROXY")
		assert.NotContains(t, logOutput, "HTTPS_PROXY")
		assert.Contains(t, logOutput, "Dynakube")
		assert.NotContains(t, logOutput, "No proxy settings found.")
	})
	t.Run("Dynakube proxy from secret", func(t *testing.T) {
		t.Setenv("HTTP_PROXY", "")
		t.Setenv("HTTPS_PROXY", "")

		proxySecret := testNewSecretBuilder(testNamespace, testSecretName)
		proxySecret.dataAppend(dynakube.ProxyKey, "foobar:1234")

		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(
				testNewDynakubeBuilder(testNamespace, testDynakube).withProxySecret(testSecretName).build(),
				testBuildNamespace(testNamespace),
				proxySecret.build(),
			).
			Build()

		dk := *testNewDynakubeBuilder(testNamespace, testDynakube).
			withProxySecret(testSecretName).
			build()

		logOutput := runWithTestLogger(func(logger logd.Logger) {
			checkProxySettings(context.Background(), logger, clt, &dk)
		})

		require.NotContains(t, logOutput, "Unexpected error")
		assert.NotContains(t, logOutput, "HTTP_PROXY")
		assert.NotContains(t, logOutput, "HTTPS_PROXY")
		assert.Contains(t, logOutput, "Dynakube")
		assert.NotContains(t, logOutput, "No proxy settings found.")
	})
	t.Run("HTTP_PROXY,HTTPS_PROXY,Dynakube proxy", func(t *testing.T) {
		t.Setenv("HTTP_PROXY", "foobar:1234")
		t.Setenv("HTTPS_PROXY", "foobar:1234")

		dk := *testNewDynakubeBuilder(testNamespace, testDynakube).
			withProxy("http://foobar:1234").
			build()

		logOutput := runWithTestLogger(func(logger logd.Logger) {
			checkProxySettings(context.Background(), logger, nil, &dk)
		})

		require.NotContains(t, logOutput, "Unexpected error")
		assert.Contains(t, logOutput, "HTTP_PROXY")
		assert.Contains(t, logOutput, "HTTPS_PROXY")
		assert.Contains(t, logOutput, "Dynakube")
		assert.NotContains(t, logOutput, "No proxy settings found.")
	})
}
