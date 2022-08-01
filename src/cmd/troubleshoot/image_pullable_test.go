package troubleshoot

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	testImage                                      = "linux/activegate"
	testOneAgentImage                              = "linux/oneagent"
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
	testInvalidImageName                           = testRegistry + testInvalidImage + ":" + testVersion
)

func TestOneAgentImagePullable(t *testing.T) {
	t.Run("OneAgent image", func(t *testing.T) {
		dockerServer := httptest.NewTLSServer(
			testDockerServerHandler(
				"HEAD",
				[]string{
					"/v2/",
					"/v2/" + testOneAgentImage + "/manifests/" + "latest",
				}))
		defer dockerServer.Close()

		server := removeSchemaRegex.FindStringSubmatch(dockerServer.URL)[1]

		auths := Auths{
			Auths: Endpoints{
				server: Credentials{
					Username: "ac",
					Password: "dt",
					Auth:     "ZW",
				},
			},
		}

		authsBytes, err := json.Marshal(auths)
		assert.NoErrorf(t, err, "fix it please")

		troubleshootCtx := troubleshootContext{
			httpClient:     dockerServer.Client(),
			namespaceName:  testNamespace,
			dynakubeName:   testDynakube,
			dynakube:       *testNewDynakubeBuilder(testNamespace, testDynakube).withApiUrl(dockerServer.URL + "/api").build(),
			pullSecretName: testDynakube + pullSecretSuffix,
			pullSecret:     *testNewSecretBuilder(testNamespace, testDynakube+pullSecretSuffix).dataAppend(pullSecretFieldName, string(authsBytes)).build(),
		}

		err = checkOneAgentImagePullable(&troubleshootCtx)
		assert.NoErrorf(t, err, "unexpected error")
	})
}

func TestActiveGateImagePullable(t *testing.T) {
	t.Run("ActiveGate image", func(t *testing.T) {
		dockerServer := httptest.NewTLSServer(
			testDockerServerHandler(
				"HEAD",
				[]string{
					"/v2/",
					"/v2/" + testImage + "/manifests/" + "latest",
				}))
		defer dockerServer.Close()

		server := removeSchemaRegex.FindStringSubmatch(dockerServer.URL)[1]

		auths := Auths{
			Auths: Endpoints{
				server: Credentials{
					Username: "ac",
					Password: "dt",
					Auth:     "ZW",
				},
			},
		}

		authsBytes, err := json.Marshal(auths)
		assert.NoErrorf(t, err, "fix it please")

		troubleshootCtx := troubleshootContext{
			httpClient:     dockerServer.Client(),
			namespaceName:  testNamespace,
			dynakubeName:   testDynakube,
			dynakube:       *testNewDynakubeBuilder(testNamespace, testDynakube).withApiUrl(dockerServer.URL + "/api").build(),
			pullSecretName: testDynakube + pullSecretSuffix,
			pullSecret:     *testNewSecretBuilder(testNamespace, testDynakube+pullSecretSuffix).dataAppend(pullSecretFieldName, string(authsBytes)).build(),
		}

		err = checkActiveGateImagePullable(&troubleshootCtx)
		assert.NoErrorf(t, err, "unexpected error")
	})
}

func TestImagePullableDockerClient(t *testing.T) {
	t.Run("connect to docker registry", func(t *testing.T) {
		dockerServer := httptest.NewTLSServer(
			testDockerServerHandler(
				"HEAD",
				[]string{
					"/v2/",
				}))
		defer dockerServer.Close()

		statusCode, err := connectToDockerRegistry(dockerServer.Client(), "HEAD", dockerServer.URL+"/v2/", "Basic", "basic")
		assert.Equal(t, http.StatusOK, statusCode, "connection not established")
		assert.NoErrorf(t, err, "unexpected error")
	})

	t.Run("component image", func(t *testing.T) {
		dockerServer := httptest.NewTLSServer(
			testDockerServerHandler(
				"HEAD",
				[]string{
					"/v2/",
					"/v2/" + testImage + "/manifests/" + testVersion,
				}))
		defer dockerServer.Close()

		server := removeSchemaRegex.FindStringSubmatch(dockerServer.URL)[1]

		auths := Auths{
			Auths: Endpoints{
				server: Credentials{
					Username: "ac",
					Password: "dt",
					Auth:     "ZW",
				},
			},
		}

		authsBytes, err := json.Marshal(auths)
		assert.NoErrorf(t, err, "fix it please")

		err = checkComponentImagePullable(dockerServer.Client(), "ActiveGate", string(authsBytes), server+"/"+testImage+":"+testVersion)
		assert.NoErrorf(t, err, "unexpected error")
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
			pullSecret:     *testNewSecretBuilder(testNamespace, testDynakube+pullSecretSuffix).dataAppend(pullSecretFieldName, pullSecretFieldValue).build(),
		}
		secret, err := getPullSecretToken(&troubleshootCtx)
		assert.NoErrorf(t, err, "unexpected error")
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
		assert.Errorf(t, err, "expected error")
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
			dynakube:      *testNewDynakubeBuilder(testNamespace, testDynakube).withApiUrl(testApiUrl).withActiveGateImage(testValidCustomImageNameWithVersion).build(),
		}
		endpoint := getActiveGateImageEndpoint(&troubleshootCtx)
		assert.Equal(t, testValidCustomImageNameWithVersion, endpoint, "invalid image")
	})
	t.Run("ActiveGate custom image without version", func(t *testing.T) {
		troubleshootCtx := troubleshootContext{
			namespaceName: testNamespace,
			dynakubeName:  testDynakube,
			dynakube:      *testNewDynakubeBuilder(testNamespace, testDynakube).withApiUrl(testApiUrl).withActiveGateImage(testValidCustomImageNameWithoutVersion).build(),
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

func TestImagePullableSplitImage(t *testing.T) {
	t.Run("valid image name with version", func(t *testing.T) {
		registry, image, version, err := splitImageName(testValidImageNameWithVersion)
		assert.Equal(t, testRegistry, registry, "invalid registry")
		assert.Equal(t, testImage, image, "invalid image")
		assert.Equal(t, testVersion, version, "invalid version")
		assert.NoErrorf(t, err, "error unexpected")
	})
	t.Run("valid image name without version", func(t *testing.T) {
		registry, image, version, err := splitImageName(testValidImageNameWithoutVersion)
		assert.Equal(t, testRegistry, registry, "invalid registry")
		assert.Equal(t, testImage, image, "invalid image")
		assert.Equal(t, "latest", version, "invalid version")
		assert.NoErrorf(t, err, "error unexpected")
	})
	t.Run("invalid image name", func(t *testing.T) {
		registry, image, version, err := splitImageName(testInvalidImageName)
		assert.NotEqual(t, testRegistry, registry, "valid registry")
		assert.NotEqual(t, testInvalidImage, image, "valid image")
		assert.NotEqual(t, testVersion, version, "valid version")
		assert.Errorf(t, err, "error not raised")
	})
}
