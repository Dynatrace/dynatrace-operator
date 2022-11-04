package troubleshoot

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/dtpullsecret"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testImage                                      = "linux/activegate"
	testOneAgentImage                              = "linux/oneagent"
	testOneAgentCodeModulesImage                   = "customdir/customcodemodules"
	testCustomImage                                = "ag"
	testCustomOneAgentImage                        = "oa"
	testInvalidImage                               = "/beos/^activegate!@invalid"
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

		dockerServerUrl, err := url.Parse(dockerServer.URL)

		require.NoError(t, err)

		server := dockerServerUrl.Host
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
			httpClient:    dockerServer.Client(),
			namespaceName: testNamespace,
			dynakubeName:  testDynakube,
			dynakube:      *testNewDynakubeBuilder(testNamespace, testDynakube).withApiUrl(dockerServer.URL + "/api").build(),
			pullSecret:    *testNewSecretBuilder(testNamespace, testDynakube+pullSecretSuffix).dataAppend(dtpullsecret.DockerConfigJson, string(authsBytes)).build(),
		}

		err = checkOneAgentImagePullable(&troubleshootCtx)
		assert.NoErrorf(t, err, "unexpected error")
	})
}

func TestOneAgentCodeModulesImagePullable(t *testing.T) {
	dockerServer := httptest.NewTLSServer(
		testDockerServerHandler(
			"HEAD",
			[]string{
				"/v2/",
				"/v2/" + testOneAgentCodeModulesImage + "/manifests/" + testVersion,
			}))
	defer dockerServer.Close()

	dockerServerUrl, err := url.Parse(dockerServer.URL)

	require.NoError(t, err)

	server := dockerServerUrl.Host
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
	require.NoErrorf(t, err, "credentials could not be marshaled, test code needs some love")

	troubleshootCtx := troubleshootContext{
		httpClient:    dockerServer.Client(),
		namespaceName: testNamespace,
		dynakubeName:  testDynakube,
		pullSecret:    *testNewSecretBuilder(testNamespace, testDynakube+pullSecretSuffix).dataAppend(dtpullsecret.DockerConfigJson, string(authsBytes)).build(),
	}

	t.Run("OneAgent code modules image", func(t *testing.T) {
		troubleshootCtx.dynakube = *testNewDynakubeBuilder(testNamespace, testDynakube).
			withApiUrl(dockerServer.URL + "/api").
			withCloudNativeCodeModulesImage(server + "/" + testOneAgentCodeModulesImage + ":" + testVersion).
			build()

		err = checkOneAgentCodeModulesImagePullable(&troubleshootCtx)
		assert.NoErrorf(t, err, "unexpected error")
	})

	t.Run("OneAgent code modules with non-existing image", func(t *testing.T) {
		troubleshootCtx.dynakube = *testNewDynakubeBuilder(testNamespace, testDynakube).
			withApiUrl(dockerServer.URL + "/api").
			withCloudNativeCodeModulesImage(server + "/non-existing-image").
			build()

		err = checkOneAgentCodeModulesImagePullable(&troubleshootCtx)
		assert.Errorf(t, err, "expected an error")
		assert.Contains(t, err.Error(), "missing")
	})

	t.Run("OneAgent code modules unknown server", func(t *testing.T) {
		troubleshootCtx.dynakube = *testNewDynakubeBuilder(testNamespace, testDynakube).
			withApiUrl(dockerServer.URL + "/api").
			withCloudNativeCodeModulesImage("myunknownserver.com/myrepo/mymissingcodemodules").
			build()

		err = checkOneAgentCodeModulesImagePullable(&troubleshootCtx)
		assert.Errorf(t, err, "expected an error")
		assert.Contains(t, err.Error(), "registry 'myunknownserver.com' unreachable")
	})

	t.Run("OneAgent code modules unreachable server", func(t *testing.T) {
		troubleshootCtx.dynakube = *testNewDynakubeBuilder(testNamespace, testDynakube).
			withApiUrl(dockerServer.URL + "/api").
			withCloudNativeCodeModulesImage("myfailingserver.com/myrepo/mymissingcodemodules").
			build()

		newAuths := auths
		newAuths.Auths["myfailingserver.com"] = Credentials{
			Username: "foobar",
			Password: "foobar",
			Auth:     "ZW",
		}

		newAuthsBytes, err := json.Marshal(newAuths)
		require.NoErrorf(t, err, "credentials could not be marshaled, test code needs some love")

		troubleshootCtx.pullSecret = *testNewSecretBuilder(testNamespace, testDynakube+pullSecretSuffix).
			dataAppend(dtpullsecret.DockerConfigJson, string(newAuthsBytes)).
			build()

		err = checkOneAgentCodeModulesImagePullable(&troubleshootCtx)
		assert.Errorf(t, err, "expected an error")
		assert.Contains(t, err.Error(), "unreachable")
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

		dockerServerUrl, err := url.Parse(dockerServer.URL)

		require.NoError(t, err)

		server := dockerServerUrl.Host
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
			httpClient:    dockerServer.Client(),
			namespaceName: testNamespace,
			dynakubeName:  testDynakube,
			dynakube:      *testNewDynakubeBuilder(testNamespace, testDynakube).withApiUrl(dockerServer.URL + "/api").build(),
			pullSecret:    *testNewSecretBuilder(testNamespace, testDynakube+pullSecretSuffix).dataAppend(dtpullsecret.DockerConfigJson, string(authsBytes)).build(),
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

		statusCode, err := connectToDockerRegistry(dockerServer.Client(), dockerServer.URL+"/v2/", "basic")
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

		dockerServerUrl, err := url.Parse(dockerServer.URL)

		require.NoError(t, err)

		server := dockerServerUrl.Host
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
			namespaceName: testNamespace,
			dynakubeName:  testDynakube,
			pullSecret:    *testNewSecretBuilder(testNamespace, testDynakube+pullSecretSuffix).dataAppend(dtpullsecret.DockerConfigJson, pullSecretFieldValue).build(),
		}
		secret, err := getPullSecret(&troubleshootCtx)
		assert.NoErrorf(t, err, "unexpected error")
		assert.Equal(t, pullSecretFieldValue, secret, "invalid contents of pull secret")
	})
	t.Run("invalid pull secret", func(t *testing.T) {
		troubleshootCtx := troubleshootContext{
			namespaceName: testNamespace,
			dynakubeName:  testDynakube,
			pullSecret:    *testNewSecretBuilder(testNamespace, testDynakube+pullSecretSuffix).dataAppend("invalidToken", pullSecretFieldValue).build(),
		}
		secret, err := getPullSecret(&troubleshootCtx)
		assert.Errorf(t, err, "expected error")
		assert.NotEqual(t, pullSecretFieldValue, secret, "valid contents of pull secret")

	})
}
