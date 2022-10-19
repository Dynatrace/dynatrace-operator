package troubleshoot

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/dtpullsecret"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

const (
	testImage                                      = "linux/activegate"
	testOneAgentImage                              = "linux/oneagent"
	testOneAgentCodeModulesImage                   = "custom_dir/customcodemodules"
	testCustomImage                                = "ag"
	testCustomOneAgentImage                        = "oa"
	testInvalidImage                               = "/beos/activegate"
	testVersion                                    = "1.248"
	testCustomRegistry                             = "testing.domain.com"
	testValidImageNameWithVersion                  = testRegistry + "/" + testImage + ":" + testVersion
	testValidImageNameWithoutVersion               = testRegistry + "/" + testImage
	testValidCustomImageNameWithVersion            = testCustomRegistry + "/" + testCustomImage + ":" + testVersion
	testValidCustomImageNameWithoutVersion         = testCustomRegistry + "/" + testCustomImage
	testValidOneAgentImageNameWithVersion          = testRegistry + "/" + testOneAgentImage + ":" + testVersion
	testValidOneAgentImageNameWithoutVersion       = testRegistry + "/" + testOneAgentImage
	testValidOneAgentCustomImageNameWithVersion    = testCustomRegistry + "/" + testCustomOneAgentImage + ":" + testVersion
	testValidOneAgentCustomImageNameWithoutVersion = testCustomRegistry + "/" + testCustomOneAgentImage
	testValidOneAgentCodeModulesImageName          = testCustomRegistry + "/" + testOneAgentCodeModulesImage
	testInvalidImageName                           = testRegistry + testInvalidImage + ":" + testVersion
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

func setupDockerMocker(handleUrls []string) (*httptest.Server, *corev1.Secret, string, error) {
	dockerServer := httptest.NewTLSServer(testDockerServerHandler("GET", handleUrls))

	server := removeSchemaRegex.FindStringSubmatch(dockerServer.URL)[1]

	secret, err := createSecret(defaultAuths(server))
	if err != nil {
		dockerServer.Close()
		return nil, nil, "", err
	}

	return dockerServer, secret, server, nil
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

func runWithTestLogger(function func()) string {
	logBuffer := bytes.Buffer{}
	logger := newTroubleshootLoggerToWriter("imagepullable_test", &logBuffer)

	oldLog := log
	log = logger
	function()
	log = oldLog
	return logBuffer.String()
}

func TestOneAgentImagePullable(t *testing.T) {
	t.Run("OneAgent image", func(t *testing.T) {

		dockerServer, secret, _, err := setupDockerMocker(
			[]string{
				"/v2/",
				"/v2/" + testOneAgentImage + "/manifests/" + "latest",
			})
		require.NoError(t, err)
		defer dockerServer.Close()

		troubleshootCtx := troubleshootContext{
			httpClient:    dockerServer.Client(),
			namespaceName: testNamespace,
			dynakubeName:  testDynakube,
			dynakube: *testNewDynakubeBuilder(testNamespace, testDynakube).
				withApiUrl(dockerServer.URL + "/api").
				withCloudNativeFullStack().
				build(),
			pullSecretName: testDynakube + pullSecretSuffix,
			pullSecret:     *secret,
			ctx:            context.TODO(),
		}

		logOutput := runWithTestLogger(func() {
			verifyImageIsAvailable("OneAgent", getOneAgentImageEndpoint(&troubleshootCtx), &troubleshootCtx)
		})
		assert.NotContains(t, logOutput, "failed")
		assert.Contains(t, logOutput, "can be successfully pulled")
	})
}

func TestOneAgentCodeModulesImagePullable(t *testing.T) {

	dockerServer, secret, server, err := setupDockerMocker(
		[]string{
			"/v2/",
			"/v2/" + testOneAgentCodeModulesImage + "/manifests/" + testVersion,
		})
	require.NoError(t, err)
	defer dockerServer.Close()

	troubleshootCtx := troubleshootContext{
		httpClient:     dockerServer.Client(),
		namespaceName:  testNamespace,
		dynakubeName:   testDynakube,
		pullSecretName: testDynakube + pullSecretSuffix,
		pullSecret:     *secret,
		ctx:            context.TODO(),
	}

	t.Run("OneAgent code modules image", func(t *testing.T) {
		troubleshootCtx.dynakube = *testNewDynakubeBuilder(testNamespace, testDynakube).
			withApiUrl(dockerServer.URL + "/api").
			withCloudNativeCodeModulesImage(server + "/" + testOneAgentCodeModulesImage + ":" + testVersion).
			build()

		logOutput := runWithTestLogger(func() {
			verifyImageIsAvailable("OneAgentCodeModules", getOneAgentCodeModulesImageEndpoint(&troubleshootCtx), &troubleshootCtx)
		})
		assert.NotContains(t, logOutput, "failed")
		assert.Contains(t, logOutput, "can be successfully pulled")
	})

	t.Run("OneAgent code modules with non-existing image", func(t *testing.T) {
		troubleshootCtx.dynakube = *testNewDynakubeBuilder(testNamespace, testDynakube).
			withApiUrl(dockerServer.URL + "/api").
			withCloudNativeCodeModulesImage(server + "/non-existing-image").
			build()

		logOutput := runWithTestLogger(func() {
			verifyImageIsAvailable("OneAgentCodeModules", getOneAgentCodeModulesImageEndpoint(&troubleshootCtx), &troubleshootCtx)
		})
		assert.Contains(t, logOutput, "failed")
		assert.NotContains(t, logOutput, "can be successfully pulled")
	})

	t.Run("OneAgent code modules unreachable server", func(t *testing.T) {
		troubleshootCtx.dynakube = *testNewDynakubeBuilder(testNamespace, testDynakube).
			withApiUrl(dockerServer.URL + "/api").
			withCloudNativeCodeModulesImage("myunknownserver.com/myrepo/mymissingcodemodules").
			build()

		logOutput := runWithTestLogger(func() {
			verifyImageIsAvailable("OneAgentCodeModules", getOneAgentCodeModulesImageEndpoint(&troubleshootCtx), &troubleshootCtx)
		})
		assert.Contains(t, logOutput, "failed")
		assert.Contains(t, logOutput, "no such host")
		assert.NotContains(t, logOutput, "can be successfully pulled")
	})
}

func TestActiveGateImagePullable(t *testing.T) {
	t.Run("ActiveGate image", func(t *testing.T) {
		dockerServer, secret, _, err := setupDockerMocker(
			[]string{
				"/v2/",
				"/v2/" + testImage + "/manifests/" + "latest",
			})
		require.NoError(t, err)
		defer dockerServer.Close()

		troubleshootCtx := troubleshootContext{
			httpClient:     dockerServer.Client(),
			namespaceName:  testNamespace,
			dynakubeName:   testDynakube,
			dynakube:       *testNewDynakubeBuilder(testNamespace, testDynakube).withApiUrl(dockerServer.URL + "/api").build(),
			pullSecretName: testDynakube + pullSecretSuffix,
			pullSecret:     *secret,
			ctx:            context.TODO(),
		}

		logOutput := runWithTestLogger(func() {
			verifyImageIsAvailable("ActiveGate", getActiveGateImageEndpoint(&troubleshootCtx), &troubleshootCtx)
		})
		assert.NotContains(t, logOutput, "failed")
		assert.Contains(t, logOutput, "can be successfully pulled")
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
		troubleshootCtx := troubleshootContext{
			namespaceName:  testNamespace,
			dynakubeName:   testDynakube,
			pullSecretName: testDynakube + pullSecretSuffix,
			pullSecret:     *testNewSecretBuilder(testNamespace, testDynakube+pullSecretSuffix).dataAppend(dtpullsecret.DockerConfigJson, pullSecretFieldValue).build(),
		}
		secret, err := getPullSecretToken(&troubleshootCtx)
		require.NoErrorf(t, err, "unexpected error")
		assert.Equal(t, pullSecretFieldValue, secret, "invalid contents of pull secret")
	})

	t.Run("invalid pull secret", func(t *testing.T) {
		troubleshootCtx := troubleshootContext{
			namespaceName:  testNamespace,
			dynakubeName:   testDynakube,
			pullSecretName: testDynakube + pullSecretSuffix,
			pullSecret:     *testNewSecretBuilder(testNamespace, testDynakube+pullSecretSuffix).dataAppend("invalidToken", pullSecretFieldValue).build(),
		}
		secret, err := getPullSecretToken(&troubleshootCtx)
		require.Errorf(t, err, "expected error")
		assert.NotEqual(t, pullSecretFieldValue, secret, "valid contents of pull secret")
	})
}

func TestImagePullableActiveGateEndpoint(t *testing.T) {
	t.Run("ActiveGate image", func(t *testing.T) {
		troubleshootCtx := troubleshootContext{
			namespaceName: testNamespace,
			dynakubeName:  testDynakube,
			dynakube:      *testNewDynakubeBuilder(testNamespace, testDynakube).withApiUrl(testApiUrl).build(),
		}
		endpoint := getActiveGateImageEndpoint(&troubleshootCtx)
		assert.Equal(t, testValidImageNameWithoutVersion, endpoint, "invalid image")
	})
	t.Run("ActiveGate custom image with version", func(t *testing.T) {
		troubleshootCtx := troubleshootContext{
			namespaceName: testNamespace,
			dynakubeName:  testDynakube,
			dynakube: *testNewDynakubeBuilder(testNamespace, testDynakube).
				withApiUrl(testApiUrl).
				withActiveGateImage(testValidCustomImageNameWithVersion).
				withActiveGateCapability("routing").
				build(),
		}
		endpoint := getActiveGateImageEndpoint(&troubleshootCtx)
		assert.Equal(t, testValidCustomImageNameWithVersion, endpoint, "invalid image")
	})
	t.Run("ActiveGate custom image without version", func(t *testing.T) {
		troubleshootCtx := troubleshootContext{
			namespaceName: testNamespace,
			dynakubeName:  testDynakube,
			dynakube: *testNewDynakubeBuilder(testNamespace, testDynakube).
				withApiUrl(testApiUrl).
				withActiveGateImage(testValidCustomImageNameWithoutVersion).
				withActiveGateCapability("routing").
				build(),
		}
		endpoint := getActiveGateImageEndpoint(&troubleshootCtx)
		assert.Equal(t, testValidCustomImageNameWithoutVersion, endpoint, "invalid image")
	})
}

func TestImagePullableOneAgentEndpoint(t *testing.T) {
	t.Run("OneAgent image", func(t *testing.T) {
		troubleshootCtx := troubleshootContext{
			namespaceName: testNamespace,
			dynakubeName:  testDynakube,
			dynakube:      *testNewDynakubeBuilder(testNamespace, testDynakube).withApiUrl(testApiUrl).build(),
		}
		endpoint := getOneAgentImageEndpoint(&troubleshootCtx)
		assert.Equal(t, testValidOneAgentImageNameWithoutVersion, endpoint, "invalid image")
	})

	t.Run("Classic Full Stack OneAgent custom image with version", func(t *testing.T) {
		troubleshootCtx := troubleshootContext{
			namespaceName: testNamespace,
			dynakubeName:  testDynakube,
			dynakube:      *testNewDynakubeBuilder(testNamespace, testDynakube).withApiUrl(testApiUrl).withClassicFullStackCustomImage(testValidOneAgentCustomImageNameWithVersion).build(),
		}
		endpoint := getOneAgentImageEndpoint(&troubleshootCtx)
		assert.Equal(t, testValidOneAgentCustomImageNameWithVersion, endpoint, "invalid image")
	})
	t.Run("Classic Full Stack OneAgent custom image without version", func(t *testing.T) {
		troubleshootCtx := troubleshootContext{
			namespaceName: testNamespace,
			dynakubeName:  testDynakube,
			dynakube:      *testNewDynakubeBuilder(testNamespace, testDynakube).withApiUrl(testApiUrl).withClassicFullStackCustomImage(testValidOneAgentCustomImageNameWithoutVersion).build(),
		}
		endpoint := getOneAgentImageEndpoint(&troubleshootCtx)
		assert.Equal(t, testValidOneAgentCustomImageNameWithoutVersion, endpoint, "invalid image")
	})
	t.Run("Classic Full Stack OneAgent regular image with version", func(t *testing.T) {
		troubleshootCtx := troubleshootContext{
			namespaceName: testNamespace,
			dynakubeName:  testDynakube,
			dynakube:      *testNewDynakubeBuilder(testNamespace, testDynakube).withApiUrl(testApiUrl).withClassicFullStackImageVersion(testVersion).build(),
		}
		endpoint := getOneAgentImageEndpoint(&troubleshootCtx)
		assert.Equal(t, testValidOneAgentImageNameWithVersion, endpoint, "invalid image")
	})
	t.Run("Classic Full Stack OneAgent custom image and Version", func(t *testing.T) {
		troubleshootCtx := troubleshootContext{
			namespaceName: testNamespace,
			dynakubeName:  testDynakube,
			dynakube:      *testNewDynakubeBuilder(testNamespace, testDynakube).withApiUrl(testApiUrl).withClassicFullStackCustomImage(testValidOneAgentCustomImageNameWithoutVersion).withClassicFullStackImageVersion(testVersion).build(),
		}
		endpoint := getOneAgentImageEndpoint(&troubleshootCtx)
		assert.Equal(t, testValidOneAgentCustomImageNameWithoutVersion, endpoint, "invalid image")
	})

	t.Run("Cloud Native OneAgent custom image with version", func(t *testing.T) {
		troubleshootCtx := troubleshootContext{
			namespaceName: testNamespace,
			dynakubeName:  testDynakube,
			dynakube:      *testNewDynakubeBuilder(testNamespace, testDynakube).withApiUrl(testApiUrl).withCloudNativeFullStackCustomImage(testValidOneAgentCustomImageNameWithVersion).build(),
		}

		endpoint := getOneAgentImageEndpoint(&troubleshootCtx)
		assert.Equal(t, testValidOneAgentCustomImageNameWithVersion, endpoint, "invalid image")
	})
	t.Run("Cloud Native OneAgent custom image without version", func(t *testing.T) {
		troubleshootCtx := troubleshootContext{
			namespaceName: testNamespace,
			dynakubeName:  testDynakube,
			dynakube:      *testNewDynakubeBuilder(testNamespace, testDynakube).withApiUrl(testApiUrl).withCloudNativeFullStackCustomImage(testValidOneAgentCustomImageNameWithoutVersion).build(),
		}
		endpoint := getOneAgentImageEndpoint(&troubleshootCtx)
		assert.Equal(t, testValidOneAgentCustomImageNameWithoutVersion, endpoint, "invalid image")
	})
	t.Run("Cloud Native OneAgent regular image with version", func(t *testing.T) {
		troubleshootCtx := troubleshootContext{
			namespaceName: testNamespace,
			dynakubeName:  testDynakube,
			dynakube:      *testNewDynakubeBuilder(testNamespace, testDynakube).withApiUrl(testApiUrl).withCloudNativeFullStackImageVersion(testVersion).build(),
		}
		endpoint := getOneAgentImageEndpoint(&troubleshootCtx)
		assert.Equal(t, testValidOneAgentImageNameWithVersion, endpoint, "invalid image")
	})
	t.Run("Classic Full Stack OneAgent custom image and Version", func(t *testing.T) {
		troubleshootCtx := troubleshootContext{
			namespaceName: testNamespace,
			dynakubeName:  testDynakube,
			dynakube:      *testNewDynakubeBuilder(testNamespace, testDynakube).withApiUrl(testApiUrl).withCloudNativeFullStackCustomImage(testValidOneAgentCustomImageNameWithoutVersion).withCloudNativeFullStackImageVersion(testVersion).build(),
		}
		endpoint := getOneAgentImageEndpoint(&troubleshootCtx)
		assert.Equal(t, testValidOneAgentCustomImageNameWithoutVersion, endpoint, "invalid image")
	})

	t.Run("Host Monitoring OneAgent custom image with version", func(t *testing.T) {
		troubleshootCtx := troubleshootContext{
			namespaceName: testNamespace,
			dynakubeName:  testDynakube,
			dynakube:      *testNewDynakubeBuilder(testNamespace, testDynakube).withApiUrl(testApiUrl).withHostMonitoringCustomImage(testValidOneAgentImageNameWithVersion).build(),
		}
		endpoint := getOneAgentImageEndpoint(&troubleshootCtx)
		assert.Equal(t, testValidOneAgentImageNameWithVersion, endpoint, "invalid image")
	})
	t.Run("Host Monitoring OneAgent custom image without version", func(t *testing.T) {
		troubleshootCtx := troubleshootContext{
			namespaceName: testNamespace,
			dynakubeName:  testDynakube,
			dynakube:      *testNewDynakubeBuilder(testNamespace, testDynakube).withApiUrl(testApiUrl).withHostMonitoringCustomImage(testValidOneAgentImageNameWithoutVersion).build(),
		}
		endpoint := getOneAgentImageEndpoint(&troubleshootCtx)
		assert.Equal(t, testValidOneAgentImageNameWithoutVersion, endpoint, "invalid image")
	})
	t.Run("Host Monitoring OneAgent regular image with version", func(t *testing.T) {
		troubleshootCtx := troubleshootContext{
			namespaceName: testNamespace,
			dynakubeName:  testDynakube,
			dynakube:      *testNewDynakubeBuilder(testNamespace, testDynakube).withApiUrl(testApiUrl).withHostMonitoringImageVersion(testVersion).build(),
		}
		endpoint := getOneAgentImageEndpoint(&troubleshootCtx)
		assert.Equal(t, testValidOneAgentImageNameWithVersion, endpoint, "invalid image")
	})
	t.Run("HostMonitoring OneAgent custom image and Version", func(t *testing.T) {
		troubleshootCtx := troubleshootContext{
			namespaceName: testNamespace,
			dynakubeName:  testDynakube,
			dynakube:      *testNewDynakubeBuilder(testNamespace, testDynakube).withApiUrl(testApiUrl).withHostMonitoringCustomImage(testValidOneAgentCustomImageNameWithoutVersion).withHostMonitoringImageVersion(testVersion).build(),
		}
		endpoint := getOneAgentImageEndpoint(&troubleshootCtx)
		assert.Equal(t, testValidOneAgentCustomImageNameWithoutVersion, endpoint, "invalid image")
	})
}

func TestImagePullableOneAgentCodeModulesEndpoint(t *testing.T) {
	t.Run("CloudNative codeModules image", func(t *testing.T) {
		troubleshootCtx := troubleshootContext{
			namespaceName: testNamespace,
			dynakubeName:  testDynakube,
			dynakube:      *testNewDynakubeBuilder(testNamespace, testDynakube).withApiUrl(testApiUrl).withCloudNativeCodeModulesImage(testValidOneAgentCodeModulesImageName).build(),
		}
		endpoint := getOneAgentCodeModulesImageEndpoint(&troubleshootCtx)
		assert.Equal(t, testValidOneAgentCodeModulesImageName, endpoint, "invalid image")
	})
	t.Run("ApplicationMonitoring codeModules image", func(t *testing.T) {
		troubleshootCtx := troubleshootContext{
			namespaceName: testNamespace,
			dynakubeName:  testDynakube,
			dynakube: *testNewDynakubeBuilder(testNamespace, testDynakube).
				withApiUrl(testApiUrl).
				withApplicationMonitoringCodeModulesImage(testValidOneAgentCodeModulesImageName).
				withApplicationMonitoringUseCSIDriver(true).
				build(),
		}
		endpoint := getOneAgentCodeModulesImageEndpoint(&troubleshootCtx)
		assert.Equal(t, testValidOneAgentCodeModulesImageName, endpoint, "invalid image")
	})
	t.Run("No codeModules image", func(t *testing.T) {
		troubleshootCtx := troubleshootContext{
			namespaceName: testNamespace,
			dynakubeName:  testDynakube,
			dynakube:      *testNewDynakubeBuilder(testNamespace, testDynakube).withApiUrl(testApiUrl).build(),
		}
		endpoint := getOneAgentCodeModulesImageEndpoint(&troubleshootCtx)
		assert.Equal(t, "", endpoint, "expected empty image")
	})
}
