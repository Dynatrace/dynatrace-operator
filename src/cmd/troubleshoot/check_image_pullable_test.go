package troubleshoot

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/scheme"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	testImage                                = "linux/activegate"
	testOneAgentImage                        = "linux/oneagent"
	testInvalidImage                         = "/beos/activegate"
	testVersion                              = "1.248"
	testValidImageNameWithVersion            = testRegistry + "/" + testImage + ":" + testVersion
	testValidImageNameWithoutVersion         = testRegistry + "/" + testImage
	testValidOneAgentImageNameWithVersion    = testRegistry + "/" + testOneAgentImage + ":" + testVersion
	testValidOneAgentImageNameWithoutVersion = testRegistry + "/" + testOneAgentImage
	testInvalidImageName                     = testRegistry + testInvalidImage + ":" + testVersion
)

func TestOneAgentImagePullable(t *testing.T) {
	t.Run("OneAgent image", func(t *testing.T) {
		troubleshootContext := TestData{namespaceName: testNamespace, dynakubeName: testDynakube}

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

		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(
				testDynakubeBuilder(testNamespace, testDynakube).withApiUrl(dockerServer.URL+"/api").build(),
				testBuildNamespace(testNamespace),
				testSecretBuilder(testNamespace, testDynakube+pullSecretSuffix).dataAppend(pullSecretFieldName, string(authsBytes)).build(),
			).
			Build()

		err = checkOneAgentImagePullable(dockerServer.Client(), clt, &troubleshootContext)
		assert.NoErrorf(t, err, "unexpected error")
	})
}

func TestActiveGateImagePullable(t *testing.T) {
	t.Run("ActiveGate image", func(t *testing.T) {
		troubleshootContext := TestData{namespaceName: testNamespace, dynakubeName: testDynakube}

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

		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(
				testDynakubeBuilder(testNamespace, testDynakube).withApiUrl(dockerServer.URL+"/api").build(),
				testBuildNamespace(testNamespace),
				testSecretBuilder(testNamespace, testDynakube+pullSecretSuffix).dataAppend(pullSecretFieldName, string(authsBytes)).build(),
			).
			Build()

		err = checkActiveGateImagePullable(dockerServer.Client(), clt, &troubleshootContext)
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
	t.Run("pull secret name", func(t *testing.T) {
		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(
				testDynakubeBuilder(testNamespace, testDynakube).build(),
				testBuildNamespace(testNamespace),
			).
			Build()

		secretName, err := getPullSecretName(clt, testDynakube, testNamespace)
		assert.Equal(t, testDynakube+pullSecretSuffix, secretName, "invalid pull secret name")
		assert.NoErrorf(t, err, "unexpected error")
	})
	t.Run("custom pull secret name", func(t *testing.T) {
		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(
				testDynakubeBuilder(testNamespace, testDynakube).withCustomPullSecret(testSecretName).build(),
				testBuildNamespace(testNamespace),
			).
			Build()

		secretName, err := getPullSecretName(clt, testDynakube, testNamespace)
		assert.Equal(t, testSecretName, secretName, "invalid pull secret name")
		assert.NoErrorf(t, err, "unexpected error")
	})

	t.Run("pull secret", func(t *testing.T) {
		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(
				testDynakubeBuilder(testNamespace, testDynakube).build(),
				testBuildNamespace(testNamespace),
				testSecretBuilder(testNamespace, testDynakube+pullSecretSuffix).dataAppend(pullSecretFieldName, pullSecretFieldValue).build(),
			).
			Build()

		secret, err := getPullSecret(clt, testDynakube+pullSecretSuffix, testNamespace)
		assert.Equal(t, pullSecretFieldValue, secret, "invalid contents of pull secret")
		assert.NoErrorf(t, err, "unexpected error")
	})
}

func TestImagePullableActiveGateEndpoint(t *testing.T) {
	t.Run("ActiveGate image", func(t *testing.T) {
		troubleshootContext := TestData{namespaceName: testNamespace, dynakubeName: testDynakube}

		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(
				testDynakubeBuilder(testNamespace, testDynakube).withApiUrl(testApiUrl).build(),
				testBuildNamespace(testNamespace),
			).
			Build()

		endpoint, err := getActiveGateImageEndpoint(clt, &troubleshootContext)
		assert.Equal(t, testValidImageNameWithoutVersion, endpoint, "invalid image")
		assert.NoErrorf(t, err, "unexpected error")
	})
	t.Run("ActiveGate custom image with version", func(t *testing.T) {
		troubleshootContext := TestData{namespaceName: testNamespace, dynakubeName: testDynakube}

		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(
				testDynakubeBuilder(testNamespace, testDynakube).withApiUrl(testApiUrl).withActiveGateImage(testValidImageNameWithVersion).build(),
				testBuildNamespace(testNamespace),
			).
			Build()

		endpoint, err := getActiveGateImageEndpoint(clt, &troubleshootContext)
		assert.Equal(t, testValidImageNameWithVersion, endpoint, "invalid image")
		assert.NoErrorf(t, err, "unexpected error")
	})
	t.Run("ActiveGate custom image without version", func(t *testing.T) {
		troubleshootContext := TestData{namespaceName: testNamespace, dynakubeName: testDynakube}

		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(
				testDynakubeBuilder(testNamespace, testDynakube).withApiUrl(testApiUrl).withActiveGateImage(testValidImageNameWithoutVersion).build(),
				testBuildNamespace(testNamespace),
			).
			Build()

		endpoint, err := getActiveGateImageEndpoint(clt, &troubleshootContext)
		assert.Equal(t, testValidImageNameWithoutVersion, endpoint, "invalid image")
		assert.NoErrorf(t, err, "unexpected error")
	})
}

func TestImagePullableOneAgentEndpoint(t *testing.T) {
	t.Run("OneAgent image", func(t *testing.T) {
		troubleshootContext := TestData{namespaceName: testNamespace, dynakubeName: testDynakube}

		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(
				testDynakubeBuilder(testNamespace, testDynakube).withApiUrl(testApiUrl).build(),
				testBuildNamespace(testNamespace),
			).
			Build()

		endpoint, err := getOneAgentImageEndpoint(clt, &troubleshootContext)
		assert.Equal(t, testValidOneAgentImageNameWithoutVersion, endpoint, "invalid image")
		assert.NoErrorf(t, err, "unexpected error")
	})

	t.Run("Classic Full Stack OneAgent custom image with version", func(t *testing.T) {
		troubleshootContext := TestData{namespaceName: testNamespace, dynakubeName: testDynakube}

		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(
				testDynakubeBuilder(testNamespace, testDynakube).withApiUrl(testApiUrl).withClassicFullStackImage(testValidOneAgentImageNameWithVersion).build(),
				testBuildNamespace(testNamespace),
			).
			Build()

		endpoint, err := getOneAgentImageEndpoint(clt, &troubleshootContext)
		assert.Equal(t, testValidOneAgentImageNameWithVersion, endpoint, "invalid image")
		assert.NoErrorf(t, err, "unexpected error")
	})
	t.Run("Classic Full Stack OneAgent custom image without version", func(t *testing.T) {
		troubleshootContext := TestData{namespaceName: testNamespace, dynakubeName: testDynakube}

		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(
				testDynakubeBuilder(testNamespace, testDynakube).withApiUrl(testApiUrl).withClassicFullStackImage(testValidOneAgentImageNameWithoutVersion).build(),
				testBuildNamespace(testNamespace),
			).
			Build()

		endpoint, err := getOneAgentImageEndpoint(clt, &troubleshootContext)
		assert.Equal(t, testValidOneAgentImageNameWithoutVersion, endpoint, "invalid image")
		assert.NoErrorf(t, err, "unexpected error")
	})

	t.Run("Cloud Native OneAgent custom image with version", func(t *testing.T) {
		troubleshootContext := TestData{namespaceName: testNamespace, dynakubeName: testDynakube}

		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(
				testDynakubeBuilder(testNamespace, testDynakube).withApiUrl(testApiUrl).withCloudNativeFullStackImage(testValidOneAgentImageNameWithVersion).build(),
				testBuildNamespace(testNamespace),
			).
			Build()

		endpoint, err := getOneAgentImageEndpoint(clt, &troubleshootContext)
		assert.Equal(t, testValidOneAgentImageNameWithVersion, endpoint, "invalid image")
		assert.NoErrorf(t, err, "unexpected error")
	})
	t.Run("Cloud Native OneAgent custom image without version", func(t *testing.T) {
		troubleshootContext := TestData{namespaceName: testNamespace, dynakubeName: testDynakube}

		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(
				testDynakubeBuilder(testNamespace, testDynakube).withApiUrl(testApiUrl).withCloudNativeFullStackImage(testValidOneAgentImageNameWithoutVersion).build(),
				testBuildNamespace(testNamespace),
			).
			Build()

		endpoint, err := getOneAgentImageEndpoint(clt, &troubleshootContext)
		assert.Equal(t, testValidOneAgentImageNameWithoutVersion, endpoint, "invalid image")
		assert.NoErrorf(t, err, "unexpected error")
	})

	t.Run("Host Monitoring OneAgent custom image with version", func(t *testing.T) {
		troubleshootContext := TestData{namespaceName: testNamespace, dynakubeName: testDynakube}

		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(
				testDynakubeBuilder(testNamespace, testDynakube).withApiUrl(testApiUrl).withHostMonitoringImage(testValidOneAgentImageNameWithVersion).build(),
				testBuildNamespace(testNamespace),
			).
			Build()

		endpoint, err := getOneAgentImageEndpoint(clt, &troubleshootContext)
		assert.Equal(t, testValidOneAgentImageNameWithVersion, endpoint, "invalid image")
		assert.NoErrorf(t, err, "unexpected error")
	})
	t.Run("Host Monitoring OneAgent custom image without version", func(t *testing.T) {
		troubleshootContext := TestData{namespaceName: testNamespace, dynakubeName: testDynakube}

		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(
				testDynakubeBuilder(testNamespace, testDynakube).withApiUrl(testApiUrl).withHostMonitoringImage(testValidOneAgentImageNameWithoutVersion).build(),
				testBuildNamespace(testNamespace),
			).
			Build()

		endpoint, err := getOneAgentImageEndpoint(clt, &troubleshootContext)
		assert.Equal(t, testValidOneAgentImageNameWithoutVersion, endpoint, "invalid image")
		assert.NoErrorf(t, err, "unexpected error")
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
