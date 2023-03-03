package troubleshoot

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/dtpullsecret"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

const (
	testActiveGateImage          = "linux/activegate"
	testOneAgentImage            = "linux/oneagent"
	testOneAgentCodeModulesImage = "custom_dir/customcodemodules"
	testActiveGateCustomImage    = "customag"
	testCustomOneAgentImage      = "customoa"
	testVersion                  = "1.248"
)

func defaultAuths(server string) Auths {
	return Auths{
		Auths: Endpoints{
			server: Credentials{
				Username: "ac",
				Password: "dt",
				Auth:     "ZW",
			},
		},
	}
}

func setupDockerMocker(handleUrls []string) (*httptest.Server, *corev1.Secret, string, error) { //nolint:revive // maximum number of return results per function exceeded; max 3 but got 4
	dockerServer := httptest.NewTLSServer(testDockerServerHandler(http.MethodGet, handleUrls))

	url, err := url.Parse(dockerServer.URL)
	if err != nil {
		dockerServer.Close()
		return nil, nil, "", err
	}

	secret, err := createSecret(defaultAuths(url.Host))
	if err != nil {
		dockerServer.Close()
		return nil, nil, "", err
	}

	return dockerServer, secret, url.Host, nil
}

func createSecret(auths Auths) (*corev1.Secret, error) {
	authsBytes, err := json.Marshal(auths)
	if err != nil {
		return nil, err
	}
	return testNewSecretBuilder(testNamespace, testDynakube+pullSecretSuffix).
		dataAppend(dtpullsecret.DockerConfigJson, string(authsBytes)).
		build(), nil
}

func TestOneAgentImagePullable(t *testing.T) {
	dockerServer, secret, _, err := setupDockerMocker(
		[]string{
			"/v2/",
			"/v2/" + testOneAgentImage + "/manifests/" + "latest",
			"/v2/" + testOneAgentImage + "/manifests/" + testVersion,
		})
	require.NoError(t, err)
	defer dockerServer.Close()

	troubleshootCtx := troubleshootContext{
		context:       context.TODO(),
		namespaceName: testNamespace,
		pullSecret:    *secret,
		httpClient:    dockerServer.Client(),
	}

	t.Run("OneAgent image", func(t *testing.T) {
		troubleshootCtx.dynakube = *testNewDynakubeBuilder(testNamespace, testDynakube).
			withApiUrl(dockerServer.URL + "/api").
			withCloudNativeFullStack().
			build()

		logOutput := runWithTestLogger(func(log logr.Logger) {
			verifyImageIsAvailable(log, &troubleshootCtx, componentOneAgent, false)
		})
		assert.NotContains(t, logOutput, "failed")
		assert.Contains(t, logOutput, "can be successfully pulled")
	})
	t.Run("OneAgent versioned image for CloudNativeFullStack", func(t *testing.T) {
		troubleshootCtx.dynakube = *testNewDynakubeBuilder(testNamespace, testDynakube).
			withApiUrl(dockerServer.URL + "/api").
			withCloudNativeFullStackImageVersion(testVersion).
			build()

		logOutput := runWithTestLogger(func(log logr.Logger) {
			verifyImageIsAvailable(log, &troubleshootCtx, componentOneAgent, false)
		})
		require.NotContains(t, logOutput, "failed")
		assert.Contains(t, logOutput, "can be successfully pulled")
	})
	t.Run("OneAgent latest image for ClassicFullStack", func(t *testing.T) {
		troubleshootCtx.dynakube = *testNewDynakubeBuilder(testNamespace, testDynakube).
			withApiUrl(dockerServer.URL + "/api").
			withClassicFullStack().
			build()

		logOutput := runWithTestLogger(func(log logr.Logger) {
			verifyImageIsAvailable(log, &troubleshootCtx, componentOneAgent, false)
		})
		require.NotContains(t, logOutput, "failed")
		assert.Contains(t, logOutput, "can be successfully pulled")
	})
	t.Run("OneAgent versioned image for ClassicFullStack", func(t *testing.T) {
		troubleshootCtx.dynakube = *testNewDynakubeBuilder(testNamespace, testDynakube).
			withApiUrl(dockerServer.URL + "/api").
			withClassicFullStackImageVersion(testVersion).
			build()

		logOutput := runWithTestLogger(func(log logr.Logger) {
			verifyImageIsAvailable(log, &troubleshootCtx, componentOneAgent, false)
		})
		require.NotContains(t, logOutput, "failed")
		assert.Contains(t, logOutput, "can be successfully pulled")
	})
	t.Run("OneAgent latest image for HostMonitoring", func(t *testing.T) {
		troubleshootCtx.dynakube = *testNewDynakubeBuilder(testNamespace, testDynakube).
			withApiUrl(dockerServer.URL + "/api").
			withHostMonitoring().
			build()

		logOutput := runWithTestLogger(func(log logr.Logger) {
			verifyImageIsAvailable(log, &troubleshootCtx, componentOneAgent, false)
		})
		require.NotContains(t, logOutput, "failed")
		assert.Contains(t, logOutput, "can be successfully pulled")
	})
	t.Run("OneAgent versioned image for HostMonitoring", func(t *testing.T) {
		troubleshootCtx.dynakube = *testNewDynakubeBuilder(testNamespace, testDynakube).
			withApiUrl(dockerServer.URL + "/api").
			withHostMonitoringImageVersion(testVersion).
			build()

		logOutput := runWithTestLogger(func(log logr.Logger) {
			verifyImageIsAvailable(log, &troubleshootCtx, componentOneAgent, false)
		})
		require.NotContains(t, logOutput, "failed")
		assert.Contains(t, logOutput, "can be successfully pulled")
	})
}

func TestOneAgentCustomImagePullable(t *testing.T) {
	dockerServer, secret, server, err := setupDockerMocker(
		[]string{
			"/v2/",
			"/v2/" + testCustomOneAgentImage + "/manifests/" + "latest",
			"/v2/" + testCustomOneAgentImage + "/manifests/" + testVersion,
		})
	require.NoError(t, err)
	defer dockerServer.Close()

	troubleshootCtx := troubleshootContext{
		httpClient:    dockerServer.Client(),
		namespaceName: testNamespace,
		context:       context.TODO(),
		pullSecret:    *secret,
	}

	t.Run("OneAgent CloudNativeFullStack unversioned custom image", func(t *testing.T) {
		troubleshootCtx.dynakube = *testNewDynakubeBuilder(testNamespace, testDynakube).
			withApiUrl(dockerServer.URL + "/api").
			withCloudNativeFullStackCustomImage(server + "/" + testCustomOneAgentImage).
			build()

		logOutput := runWithTestLogger(func(log logr.Logger) {
			verifyImageIsAvailable(log, &troubleshootCtx, componentOneAgent, false)
		})
		require.NotContains(t, logOutput, "failed")
		assert.Contains(t, logOutput, "can be successfully pulled")
	})
	t.Run("OneAgent CloudNativeFullStack latest custom image", func(t *testing.T) {
		troubleshootCtx.dynakube = *testNewDynakubeBuilder(testNamespace, testDynakube).
			withApiUrl(dockerServer.URL + "/api").
			withCloudNativeFullStackCustomImage(server + "/" + testCustomOneAgentImage + ":latest").
			build()

		logOutput := runWithTestLogger(func(log logr.Logger) {
			verifyImageIsAvailable(log, &troubleshootCtx, componentOneAgent, false)
		})
		require.NotContains(t, logOutput, "failed")
		assert.Contains(t, logOutput, "can be successfully pulled")
	})
	t.Run("OneAgent CloudNativeFullStack versioned custom image", func(t *testing.T) {
		troubleshootCtx.dynakube = *testNewDynakubeBuilder(testNamespace, testDynakube).
			withApiUrl(dockerServer.URL + "/api").
			withCloudNativeFullStackCustomImage(server + "/" + testCustomOneAgentImage + ":" + testVersion).
			build()

		logOutput := runWithTestLogger(func(log logr.Logger) {
			verifyImageIsAvailable(log, &troubleshootCtx, componentOneAgent, false)
		})
		require.NotContains(t, logOutput, "failed")
		assert.Contains(t, logOutput, "can be successfully pulled")
	})
	t.Run("OneAgent ClassicFullStack custom image", func(t *testing.T) {
		troubleshootCtx.dynakube = *testNewDynakubeBuilder(testNamespace, testDynakube).
			withApiUrl(dockerServer.URL + "/api").
			withClassicFullStackCustomImage(server + "/" + testCustomOneAgentImage).
			build()

		logOutput := runWithTestLogger(func(log logr.Logger) {
			verifyImageIsAvailable(log, &troubleshootCtx, componentOneAgent, false)
		})
		require.NotContains(t, logOutput, "failed")
		assert.Contains(t, logOutput, "can be successfully pulled")
	})
	t.Run("OneAgent ClassicFullStack latest custom image", func(t *testing.T) {
		troubleshootCtx.dynakube = *testNewDynakubeBuilder(testNamespace, testDynakube).
			withApiUrl(dockerServer.URL + "/api").
			withClassicFullStackCustomImage(server + "/" + testCustomOneAgentImage + ":latest").
			build()

		logOutput := runWithTestLogger(func(log logr.Logger) {
			verifyImageIsAvailable(log, &troubleshootCtx, componentOneAgent, false)
		})
		require.NotContains(t, logOutput, "failed")
		assert.Contains(t, logOutput, "can be successfully pulled")
	})
	t.Run("OneAgent ClassicFullStack versioned custom image", func(t *testing.T) {
		troubleshootCtx.dynakube = *testNewDynakubeBuilder(testNamespace, testDynakube).
			withApiUrl(dockerServer.URL + "/api").
			withClassicFullStackCustomImage(server + "/" + testCustomOneAgentImage + ":" + testVersion).
			build()

		logOutput := runWithTestLogger(func(log logr.Logger) {
			verifyImageIsAvailable(log, &troubleshootCtx, componentOneAgent, false)
		})
		require.NotContains(t, logOutput, "failed")
		assert.Contains(t, logOutput, "can be successfully pulled")
	})
	t.Run("OneAgent HostMonitoring custom image", func(t *testing.T) {
		troubleshootCtx.dynakube = *testNewDynakubeBuilder(testNamespace, testDynakube).
			withApiUrl(dockerServer.URL + "/api").
			withHostMonitoringCustomImage(server + "/" + testCustomOneAgentImage).
			build()

		logOutput := runWithTestLogger(func(log logr.Logger) {
			verifyImageIsAvailable(log, &troubleshootCtx, componentOneAgent, false)
		})
		require.NotContains(t, logOutput, "failed")
		assert.Contains(t, logOutput, "can be successfully pulled")
	})
	t.Run("OneAgent HostMonitoring latest custom image", func(t *testing.T) {
		troubleshootCtx.dynakube = *testNewDynakubeBuilder(testNamespace, testDynakube).
			withApiUrl(dockerServer.URL + "/api").
			withHostMonitoringCustomImage(server + "/" + testCustomOneAgentImage + ":latest").
			build()

		logOutput := runWithTestLogger(func(log logr.Logger) {
			verifyImageIsAvailable(log, &troubleshootCtx, componentOneAgent, false)
		})
		require.NotContains(t, logOutput, "failed")
		assert.Contains(t, logOutput, "can be successfully pulled")
	})
	t.Run("OneAgent HostMonitoring versioned custom image", func(t *testing.T) {
		troubleshootCtx.dynakube = *testNewDynakubeBuilder(testNamespace, testDynakube).
			withApiUrl(dockerServer.URL + "/api").
			withHostMonitoringCustomImage(server + "/" + testCustomOneAgentImage + ":" + testVersion).
			build()

		logOutput := runWithTestLogger(func(log logr.Logger) {
			verifyImageIsAvailable(log, &troubleshootCtx, componentOneAgent, false)
		})
		require.NotContains(t, logOutput, "failed")
		assert.Contains(t, logOutput, "can be successfully pulled")
	})
}

func TestOneAgentImageNotPullable(t *testing.T) {
	dockerServer, secret, server, err := setupDockerMocker(
		[]string{
			"/v2/",
		})
	require.NoError(t, err)
	defer dockerServer.Close()

	troubleshootCtx := troubleshootContext{
		context:       context.TODO(),
		namespaceName: testNamespace,
		pullSecret:    *secret,
		httpClient:    dockerServer.Client(),
	}

	t.Run("OneAgent latest image for CloudNativeFullStack not available", func(t *testing.T) {
		troubleshootCtx.dynakube = *testNewDynakubeBuilder(testNamespace, testDynakube).
			withApiUrl(dockerServer.URL + "/api").
			withCloudNativeFullStack().
			build()

		logOutput := runWithTestLogger(func(log logr.Logger) {
			verifyImageIsAvailable(log, &troubleshootCtx, componentOneAgent, false)
		})
		require.Contains(t, logOutput, "failed")
		assert.Contains(t, logOutput, "reading manifest")
		assert.NotContains(t, logOutput, "can be successfully pulled")
	})
	t.Run("OneAgent CloudNativeFullStack non-existing custom image", func(t *testing.T) {
		troubleshootCtx.dynakube = *testNewDynakubeBuilder(testNamespace, testDynakube).
			withApiUrl(dockerServer.URL + "/api").
			withCloudNativeFullStackCustomImage(server + "/foobar").
			build()

		logOutput := runWithTestLogger(func(log logr.Logger) {
			verifyImageIsAvailable(log, &troubleshootCtx, componentOneAgent, false)
		})
		require.Contains(t, logOutput, "failed")
		assert.Contains(t, logOutput, "reading manifest")
		assert.NotContains(t, logOutput, "can be successfully pulled")
	})
	t.Run("OneAgent latest image for ClassicFullStack not available", func(t *testing.T) {
		troubleshootCtx.dynakube = *testNewDynakubeBuilder(testNamespace, testDynakube).
			withApiUrl(dockerServer.URL + "/api").
			withClassicFullStack().
			build()

		logOutput := runWithTestLogger(func(log logr.Logger) {
			verifyImageIsAvailable(log, &troubleshootCtx, componentOneAgent, false)
		})
		require.Contains(t, logOutput, "failed")
		assert.Contains(t, logOutput, "reading manifest")
		assert.NotContains(t, logOutput, "can be successfully pulled")
	})
	t.Run("OneAgent ClassicFullStack non-existing custom image", func(t *testing.T) {
		troubleshootCtx.dynakube = *testNewDynakubeBuilder(testNamespace, testDynakube).
			withApiUrl(dockerServer.URL + "/api").
			withClassicFullStackCustomImage(server + "/foobar").
			build()

		logOutput := runWithTestLogger(func(log logr.Logger) {
			verifyImageIsAvailable(log, &troubleshootCtx, componentOneAgent, false)
		})
		require.Contains(t, logOutput, "failed")
		assert.Contains(t, logOutput, "reading manifest")
		assert.NotContains(t, logOutput, "can be successfully pulled")
	})
	t.Run("OneAgent latest image for HostMonitoring not available", func(t *testing.T) {
		troubleshootCtx.dynakube = *testNewDynakubeBuilder(testNamespace, testDynakube).
			withApiUrl(dockerServer.URL + "/api").
			withHostMonitoring().
			build()

		logOutput := runWithTestLogger(func(log logr.Logger) {
			verifyImageIsAvailable(log, &troubleshootCtx, componentOneAgent, false)
		})
		require.Contains(t, logOutput, "failed")
		assert.Contains(t, logOutput, "reading manifest")
		assert.NotContains(t, logOutput, "can be successfully pulled")
	})
	t.Run("OneAgent HostMonitoring non-existing custom image", func(t *testing.T) {
		troubleshootCtx.dynakube = *testNewDynakubeBuilder(testNamespace, testDynakube).
			withApiUrl(dockerServer.URL + "/api").
			withHostMonitoringCustomImage(server + "/foobar").
			build()

		logOutput := runWithTestLogger(func(log logr.Logger) {
			verifyImageIsAvailable(log, &troubleshootCtx, componentOneAgent, false)
		})
		require.Contains(t, logOutput, "failed")
		assert.Contains(t, logOutput, "reading manifest")
		assert.NotContains(t, logOutput, "can be successfully pulled")
	})
	t.Run("OneAgent HostMonitoring non-existing server", func(t *testing.T) {
		troubleshootCtx.dynakube = *testNewDynakubeBuilder(testNamespace, testDynakube).
			withApiUrl(dockerServer.URL + "/api").
			withHostMonitoringCustomImage("myunknownserver.com/foobar/image").
			build()

		logOutput := runWithTestLogger(func(log logr.Logger) {
			verifyImageIsAvailable(log, &troubleshootCtx, componentOneAgent, false)
		})
		require.Contains(t, logOutput, "failed")
		assert.Contains(t, logOutput, "no such host")
		assert.NotContains(t, logOutput, "can be successfully pulled")
	})
}

func TestOneAgentCodeModulesImagePullable(t *testing.T) {
	dockerServer, secret, server, err := setupDockerMocker(
		[]string{
			"/v2/",
			"/v2/" + testOneAgentCodeModulesImage + "/manifests/latest",
			"/v2/" + testOneAgentCodeModulesImage + "/manifests/" + testVersion,
		})
	require.NoError(t, err)
	defer dockerServer.Close()

	troubleshootCtx := troubleshootContext{
		context:       context.TODO(),
		httpClient:    dockerServer.Client(),
		namespaceName: testNamespace,
		pullSecret:    *secret,
	}

	t.Run("OneAgent code modules image", func(t *testing.T) {
		troubleshootCtx.dynakube = *testNewDynakubeBuilder(testNamespace, testDynakube).
			withApiUrl(dockerServer.URL + "/api").
			withCloudNativeCodeModulesImage(server + "/" + testOneAgentCodeModulesImage + ":" + testVersion).
			build()

		logOutput := runWithTestLogger(func(log logr.Logger) {
			verifyImageIsAvailable(log, &troubleshootCtx, componentCodeModules, true)
		})
		assert.NotContains(t, logOutput, "failed")
		assert.Contains(t, logOutput, "can be successfully pulled")
	})
	t.Run("CloudNativeFullStack OneAgent code modules image", func(t *testing.T) {
		troubleshootCtx.dynakube = *testNewDynakubeBuilder(testNamespace, testDynakube).
			withApiUrl(dockerServer.URL + "/api").
			withCloudNativeCodeModulesImage(server + "/" + testOneAgentCodeModulesImage + ":" + testVersion).
			build()

		logOutput := runWithTestLogger(func(log logr.Logger) {
			verifyImageIsAvailable(log, &troubleshootCtx, componentCodeModules, true)
		})
		require.NotContains(t, logOutput, "failed")
		assert.Contains(t, logOutput, "can be successfully pulled")
	})
	t.Run("CloudNativeFullStack OneAgent code modules latest image", func(t *testing.T) {
		troubleshootCtx.dynakube = *testNewDynakubeBuilder(testNamespace, testDynakube).
			withApiUrl(dockerServer.URL + "/api").
			withCloudNativeCodeModulesImage(server + "/" + testOneAgentCodeModulesImage + ":latest").
			build()

		logOutput := runWithTestLogger(func(log logr.Logger) {
			verifyImageIsAvailable(log, &troubleshootCtx, componentCodeModules, true)
		})
		require.NotContains(t, logOutput, "failed")
		assert.Contains(t, logOutput, "can be successfully pulled")
	})
	t.Run("CloudNativeFullStack OneAgent code modules unversioned image", func(t *testing.T) {
		troubleshootCtx.dynakube = *testNewDynakubeBuilder(testNamespace, testDynakube).
			withApiUrl(dockerServer.URL + "/api").
			withCloudNativeCodeModulesImage(server + "/" + testOneAgentCodeModulesImage).
			build()

		logOutput := runWithTestLogger(func(log logr.Logger) {
			verifyImageIsAvailable(log, &troubleshootCtx, componentCodeModules, true)
		})
		require.NotContains(t, logOutput, "failed")
		assert.Contains(t, logOutput, "can be successfully pulled")
	})
	t.Run("ApplicationMonitoring OneAgent code modules image", func(t *testing.T) {
		troubleshootCtx.dynakube = *testNewDynakubeBuilder(testNamespace, testDynakube).
			withApiUrl(dockerServer.URL + "/api").
			withApplicationMonitoringCodeModulesImage(server + "/" + testOneAgentCodeModulesImage + ":" + testVersion).
			build()

		logOutput := runWithTestLogger(func(log logr.Logger) {
			verifyImageIsAvailable(log, &troubleshootCtx, componentCodeModules, true)
		})
		require.NotContains(t, logOutput, "failed")
		assert.Contains(t, logOutput, "can be successfully pulled")
	})
	t.Run("ApplicationMonitoring OneAgent code modules latest image", func(t *testing.T) {
		troubleshootCtx.dynakube = *testNewDynakubeBuilder(testNamespace, testDynakube).
			withApiUrl(dockerServer.URL + "/api").
			withApplicationMonitoringCodeModulesImage(server + "/" + testOneAgentCodeModulesImage + ":latest").
			build()

		logOutput := runWithTestLogger(func(log logr.Logger) {
			verifyImageIsAvailable(log, &troubleshootCtx, componentCodeModules, true)
		})
		require.NotContains(t, logOutput, "failed")
		assert.Contains(t, logOutput, "can be successfully pulled")
	})
	t.Run("ApplicationMonitoring OneAgent code modules unversioned image", func(t *testing.T) {
		troubleshootCtx.dynakube = *testNewDynakubeBuilder(testNamespace, testDynakube).
			withApiUrl(dockerServer.URL + "/api").
			withApplicationMonitoringCodeModulesImage(server + "/" + testOneAgentCodeModulesImage).
			build()

		logOutput := runWithTestLogger(func(log logr.Logger) {
			verifyImageIsAvailable(log, &troubleshootCtx, componentCodeModules, true)
		})
		require.NotContains(t, logOutput, "failed")
		assert.Contains(t, logOutput, "can be successfully pulled")
	})
	t.Run("OneAgent code modules with non-existing image", func(t *testing.T) {
		troubleshootCtx.dynakube = *testNewDynakubeBuilder(testNamespace, testDynakube).
			withApiUrl(dockerServer.URL + "/api").
			withCloudNativeCodeModulesImage(server + "/non-existing-image").
			build()

		logOutput := runWithTestLogger(func(log logr.Logger) {
			verifyImageIsAvailable(log, &troubleshootCtx, componentCodeModules, true)
		})
		assert.Contains(t, logOutput, "failed")
		assert.NotContains(t, logOutput, "can be successfully pulled")
	})
	t.Run("OneAgent code modules unreachable server", func(t *testing.T) {
		troubleshootCtx.dynakube = *testNewDynakubeBuilder(testNamespace, testDynakube).
			withApiUrl(dockerServer.URL + "/api").
			withCloudNativeCodeModulesImage("myunknownserver.com/myrepo/mymissingcodemodules").
			build()

		logOutput := runWithTestLogger(func(log logr.Logger) {
			verifyImageIsAvailable(log, &troubleshootCtx, componentCodeModules, true)
		})
		assert.Contains(t, logOutput, "failed")
		assert.Contains(t, logOutput, "no such host")
		assert.NotContains(t, logOutput, "can be successfully pulled")
	})
	t.Run("OneAgent code modules image with unset image", func(t *testing.T) {
		troubleshootCtx.dynakube = *testNewDynakubeBuilder(testNamespace, testDynakube).
			withApiUrl(dockerServer.URL + "/api").
			withCloudNativeCodeModulesImage("").
			build()

		logOutput := runWithTestLogger(func(log logr.Logger) {
			verifyImageIsAvailable(log, &troubleshootCtx, componentCodeModules, true)
		})
		assert.NotContains(t, logOutput, "Unknown OneAgentCodeModules image")
	})
}

func TestActiveGateImagePullable(t *testing.T) {
	dockerServer, secret, server, err := setupDockerMocker(
		[]string{
			"/v2/",
			"/v2/" + testActiveGateImage + "/manifests/" + "latest",
			"/v2/" + testActiveGateImage + "/manifests/" + testVersion,
			"/v2/" + testActiveGateCustomImage + "/manifests/" + "latest",
			"/v2/" + testActiveGateCustomImage + "/manifests/" + testVersion,
		})
	require.NoError(t, err)
	defer dockerServer.Close()

	troubleshootCtx := troubleshootContext{
		context:       context.TODO(),
		httpClient:    dockerServer.Client(),
		namespaceName: testNamespace,
		pullSecret:    *secret,
	}

	t.Run("ActiveGate image", func(t *testing.T) {
		troubleshootCtx.dynakube = *testNewDynakubeBuilder(testNamespace, testDynakube).
			withApiUrl(dockerServer.URL + "/api").
			build()

		logOutput := runWithTestLogger(func(log logr.Logger) {
			verifyImageIsAvailable(log, &troubleshootCtx, componentActiveGate, false)
		})
		assert.NotContains(t, logOutput, "failed")
		assert.Contains(t, logOutput, "can be successfully pulled")
	})
	t.Run("ActiveGate custom image", func(t *testing.T) {
		troubleshootCtx.dynakube = *testNewDynakubeBuilder(testNamespace, testDynakube).
			withApiUrl(dockerServer.URL + "/api").
			withActiveGateCustomImage(server + "/" + testActiveGateCustomImage).
			build()

		logOutput := runWithTestLogger(func(log logr.Logger) {
			verifyImageIsAvailable(log, &troubleshootCtx, componentActiveGate, false)
		})
		assert.NotContains(t, logOutput, "failed")
		assert.Contains(t, logOutput, "can be successfully pulled")
	})
	t.Run("ActiveGate latest custom image", func(t *testing.T) {
		troubleshootCtx.dynakube = *testNewDynakubeBuilder(testNamespace, testDynakube).
			withApiUrl(dockerServer.URL + "/api").
			withActiveGateCustomImage(server + "/" + testActiveGateCustomImage + ":latest").
			build()

		logOutput := runWithTestLogger(func(log logr.Logger) {
			verifyImageIsAvailable(log, &troubleshootCtx, componentActiveGate, false)
		})
		assert.NotContains(t, logOutput, "failed")
		assert.Contains(t, logOutput, "can be successfully pulled")
	})
	t.Run("ActiveGate versioned custom image", func(t *testing.T) {
		troubleshootCtx.dynakube = *testNewDynakubeBuilder(testNamespace, testDynakube).
			withApiUrl(dockerServer.URL + "/api").
			withActiveGateCustomImage(server + "/" + testActiveGateCustomImage + ":" + testVersion).
			build()

		logOutput := runWithTestLogger(func(log logr.Logger) {
			verifyImageIsAvailable(log, &troubleshootCtx, componentActiveGate, false)
		})
		assert.NotContains(t, logOutput, "failed")
		assert.Contains(t, logOutput, "can be successfully pulled")
	})
}

func TestActiveGateImageNotPullable(t *testing.T) {
	dockerServer, secret, server, err := setupDockerMocker(
		[]string{
			"/v2/",
		})
	require.NoError(t, err)
	defer dockerServer.Close()

	troubleshootCtx := troubleshootContext{
		context:       context.TODO(),
		httpClient:    dockerServer.Client(),
		namespaceName: testNamespace,
		pullSecret:    *secret,
	}

	t.Run("ActiveGate image", func(t *testing.T) {
		troubleshootCtx.dynakube = *testNewDynakubeBuilder(testNamespace, testDynakube).
			withApiUrl(dockerServer.URL + "/api").
			build()

		logOutput := runWithTestLogger(func(log logr.Logger) {
			verifyImageIsAvailable(log, &troubleshootCtx, componentActiveGate, false)
		})
		require.Contains(t, logOutput, "failed")
		assert.Contains(t, logOutput, "reading manifest")
		assert.NotContains(t, logOutput, "can be successfully pulled")
	})
	t.Run("ActiveGate custom image", func(t *testing.T) {
		troubleshootCtx.dynakube = *testNewDynakubeBuilder(testNamespace, testDynakube).
			withApiUrl(dockerServer.URL + "/api").
			withActiveGateCustomImage(server + "/" + testActiveGateCustomImage).
			withActiveGateCapability(v1beta1.RoutingCapability.DisplayName).
			build()

		logOutput := runWithTestLogger(func(log logr.Logger) {
			verifyImageIsAvailable(log, &troubleshootCtx, componentActiveGate, false)
		})
		require.Contains(t, logOutput, "failed")
		assert.Contains(t, logOutput, "reading manifest")
		assert.NotContains(t, logOutput, "can be successfully pulled")
	})
	t.Run("ActiveGate custom image non-existing server", func(t *testing.T) {
		troubleshootCtx.dynakube = *testNewDynakubeBuilder(testNamespace, testDynakube).
			withApiUrl(dockerServer.URL + "/api").
			withActiveGateCustomImage("myunknownserver.com/foobar/image").
			withActiveGateCapability(v1beta1.RoutingCapability.DisplayName).
			build()

		logOutput := runWithTestLogger(func(log logr.Logger) {
			verifyImageIsAvailable(log, &troubleshootCtx, componentActiveGate, false)
		})
		require.Contains(t, logOutput, "failed")
		assert.Contains(t, logOutput, "no such host")
		assert.NotContains(t, logOutput, "can be successfully pulled")
	})
}

func testDockerServerHandler(method string, urls []string) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		for _, url := range urls {
			if request.Method == method && request.URL.Path == url {
				writer.WriteHeader(http.StatusOK)
				return
			}
		}
		writer.WriteHeader(http.StatusBadRequest)
	}
}

func TestImagePullablePullSecret(t *testing.T) {
	t.Run("valid pull secret", func(t *testing.T) {
		troubleshootcontext := troubleshootContext{
			namespaceName: testNamespace,
			pullSecret:    *testNewSecretBuilder(testNamespace, testDynakube+pullSecretSuffix).dataAppend(dtpullsecret.DockerConfigJson, pullSecretFieldValue).build(),
		}
		secret, err := getPullSecretToken(&troubleshootcontext)
		require.NoErrorf(t, err, "unexpected error")
		assert.Equal(t, pullSecretFieldValue, secret, "invalid contents of pull secret")
	})

	t.Run("invalid pull secret", func(t *testing.T) {
		troubleshootcontext := troubleshootContext{
			namespaceName: testNamespace,
			pullSecret:    *testNewSecretBuilder(testNamespace, testDynakube+pullSecretSuffix).dataAppend("invalidToken", pullSecretFieldValue).build(),
		}
		secret, err := getPullSecretToken(&troubleshootcontext)
		require.Errorf(t, err, "expected error")
		assert.NotEqual(t, pullSecretFieldValue, secret, "valid contents of pull secret")
	})
}
