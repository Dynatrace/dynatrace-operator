package troubleshoot

import (
	"bytes"
	"context"
	"os"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/scheme"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestCheckProxySettings(t *testing.T) {
	t.Run("No proxy settings", func(t *testing.T) {
		os.Setenv("HTTP_PROXY", "")
		os.Setenv("HTTPS_PROXY", "")

		troubleshootCtx := troubleshootContext{
			context:       context.TODO(),
			namespaceName: testNamespace,
			dynakubeName:  testDynakube,
		}

		logOutput := runProxyTestWithTestLogger(t.Name(), func(logger logr.Logger) {
			checkProxySettingsWithLog(&troubleshootCtx, logger)
		})

		require.NotContains(t, logOutput, "Unexpected error")
		assert.NotContains(t, logOutput, "HTTP_PROXY")
		assert.NotContains(t, logOutput, "HTTPS_PROXY")
		assert.NotContains(t, logOutput, "Dynakube")
		assert.Contains(t, logOutput, "No proxy settings found.")
	})
	t.Run("HTTP_PROXY", func(t *testing.T) {
		os.Setenv("HTTP_PROXY", "foobar:1234")
		os.Setenv("HTTPS_PROXY", "")

		troubleshootCtx := troubleshootContext{
			context:       context.TODO(),
			namespaceName: testNamespace,
			dynakubeName:  testDynakube,
		}

		logOutput := runProxyTestWithTestLogger(t.Name(), func(logger logr.Logger) {
			checkProxySettingsWithLog(&troubleshootCtx, logger)
		})

		require.NotContains(t, logOutput, "Unexpected error")
		assert.Contains(t, logOutput, "HTTP_PROXY")
		assert.NotContains(t, logOutput, "HTTPS_PROXY")
		assert.NotContains(t, logOutput, "Dynakube")
		assert.NotContains(t, logOutput, "No proxy settings found.")
	})
	t.Run("HTTPS_PROXY", func(t *testing.T) {
		os.Setenv("HTTP_PROXY", "")
		os.Setenv("HTTPS_PROXY", "foobar:1234")

		troubleshootCtx := troubleshootContext{
			context:       context.TODO(),
			namespaceName: testNamespace,
			dynakubeName:  testDynakube,
		}

		logOutput := runProxyTestWithTestLogger(t.Name(), func(logger logr.Logger) {
			checkProxySettingsWithLog(&troubleshootCtx, logger)
		})

		require.NotContains(t, logOutput, "Unexpected error")
		assert.NotContains(t, logOutput, "HTTP_PROXY")
		assert.Contains(t, logOutput, "HTTPS_PROXY")
		assert.NotContains(t, logOutput, "Dynakube")
		assert.NotContains(t, logOutput, "No proxy settings found.")

	})
	t.Run("Dynakube proxy", func(t *testing.T) {
		os.Setenv("HTTP_PROXY", "")
		os.Setenv("HTTPS_PROXY", "")

		troubleshootCtx := troubleshootContext{
			context:       context.TODO(),
			namespaceName: testNamespace,
			dynakubeName:  testDynakube,
		}

		troubleshootCtx.dynakube = *testNewDynakubeBuilder(testNamespace, testDynakube).
			withProxy("http://foobar:1234").
			build()

		logOutput := runProxyTestWithTestLogger(t.Name(), func(logger logr.Logger) {
			checkProxySettingsWithLog(&troubleshootCtx, logger)
		})

		require.NotContains(t, logOutput, "Unexpected error")
		assert.NotContains(t, logOutput, "HTTP_PROXY")
		assert.NotContains(t, logOutput, "HTTPS_PROXY")
		assert.Contains(t, logOutput, "Dynakube")
		assert.NotContains(t, logOutput, "No proxy settings found.")

	})
	t.Run("Dynakube proxy from secret", func(t *testing.T) {
		os.Setenv("HTTP_PROXY", "")
		os.Setenv("HTTPS_PROXY", "")

		proxySecret := testNewSecretBuilder(testNamespace, testSecretName)
		proxySecret.dataAppend(dtclient.CustomProxySecretKey, "foobar:1234")

		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(
				testNewDynakubeBuilder(testNamespace, testDynakube).withProxySecret(testSecretName).build(),
				testBuildNamespace(testNamespace),
				proxySecret.build(),
			).
			Build()

		troubleshootCtx := troubleshootContext{
			context:       context.TODO(),
			apiReader:     clt,
			namespaceName: testNamespace,
			dynakubeName:  testDynakube,
		}

		troubleshootCtx.dynakube = *testNewDynakubeBuilder(testNamespace, testDynakube).
			withProxySecret(testSecretName).
			build()

		logOutput := runProxyTestWithTestLogger(t.Name(), func(logger logr.Logger) {
			checkProxySettingsWithLog(&troubleshootCtx, logger)
		})

		require.NotContains(t, logOutput, "Unexpected error")
		assert.NotContains(t, logOutput, "HTTP_PROXY")
		assert.NotContains(t, logOutput, "HTTPS_PROXY")
		assert.Contains(t, logOutput, "Dynakube")
		assert.NotContains(t, logOutput, "No proxy settings found.")
	})
	t.Run("HTTP_PROXY,HTTPS_PROXY,Dynakube proxy", func(t *testing.T) {
		os.Setenv("HTTP_PROXY", "foobar:1234")
		os.Setenv("HTTPS_PROXY", "foobar:1234")

		troubleshootCtx := troubleshootContext{
			context:       context.TODO(),
			namespaceName: testNamespace,
			dynakubeName:  testDynakube,
		}

		troubleshootCtx.dynakube = *testNewDynakubeBuilder(testNamespace, testDynakube).
			withProxy("http://foobar:1234").
			build()

		logOutput := runProxyTestWithTestLogger(t.Name(), func(logger logr.Logger) {
			checkProxySettingsWithLog(&troubleshootCtx, logger)
		})

		require.NotContains(t, logOutput, "Unexpected error")
		assert.Contains(t, logOutput, "HTTP_PROXY")
		assert.Contains(t, logOutput, "HTTPS_PROXY")
		assert.Contains(t, logOutput, "Dynakube")
		assert.NotContains(t, logOutput, "No proxy settings found.")
	})
}

func runProxyTestWithTestLogger(testName string, function func(logger logr.Logger)) string {
	logBuffer := bytes.Buffer{}
	logger := newTroubleshootLoggerToWriter(testName, &logBuffer)
	function(logger)
	return logBuffer.String()
}
