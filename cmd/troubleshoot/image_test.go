package troubleshoot

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/dtpullsecret"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/oci/dockerkeychain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

const (
	testOneAgentCodeModulesImage = "custom_dir/customcodemodules"
	testActiveGateCustomImage    = "customag"
	testCustomOneAgentImage      = "customoa"
	testVersion                  = "1.248.3"
)

func defaultAuths(server string) Auths {
	return Auths{
		Auths: Endpoints{
			server: Credentials{
				Username: "ac",
				Password: "dt",
				Auth:     "dGVzdC10b2tlbjp0ZXN0LXBhc3N3b3Jk",
			},
		},
	}
}

func setupDockerMocker(handleUrls []string) (*httptest.Server, *corev1.Secret, string, error) { //nolint:revive // maximum number of return results per function exceeded; max 3 but got 4
	dockerServer := httptest.NewTLSServer(testDockerServerHandler(http.MethodGet, handleUrls))

	parsedServerUrl, err := url.Parse(dockerServer.URL)
	if err != nil {
		dockerServer.Close()

		return nil, nil, "", err
	}

	secret, err := createSecret(defaultAuths(parsedServerUrl.Host))
	if err != nil {
		dockerServer.Close()

		return nil, nil, "", err
	}

	return dockerServer, secret, parsedServerUrl.Host, nil
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

func dynakubeBuilder(dockerUrl string) *testDynaKubeBuilder {
	return testNewDynakubeBuilder(testNamespace, testDynakube).
		withApiUrl(dockerUrl + "/api")
}

func TestImagePullable(t *testing.T) {
	dockerServer, secret, server, err := setupDockerMocker(
		[]string{
			"/v2/",
			"/v2" + dynakube.DefaultOneAgentImageRegistrySubPath + "/manifests/" + testVersion + "-raw",
			"/v2/" + testCustomOneAgentImage + "/manifests/" + testVersion,
			"/v2/" + testOneAgentCodeModulesImage + "/manifests/" + testVersion,
			"/v2" + activegate.DefaultImageRegistrySubPath + "/manifests/" + testVersion + "-raw",
			"/v2/" + testActiveGateCustomImage + "/manifests/" + testVersion,
		})
	require.NoError(t, err)

	defer dockerServer.Close()

	tests := []struct {
		name         string
		dk           *dynakube.DynaKube
		component    component
		proxyWarning bool
	}{
		// standard OneAgent images
		{
			name:         "Default CloudNative OneAgent image",
			dk:           dynakubeBuilder(dockerServer.URL).withCloudNativeFullStack().build(),
			component:    componentOneAgent,
			proxyWarning: false,
		},
		{
			name:         "Default ClassicFullstack OneAgent image",
			dk:           dynakubeBuilder(dockerServer.URL).withClassicFullStack().build(),
			component:    componentOneAgent,
			proxyWarning: false,
		},
		{
			name:         "Default HostMonitoring OneAgent image",
			dk:           dynakubeBuilder(dockerServer.URL).withHostMonitoring().build(),
			component:    componentOneAgent,
			proxyWarning: false,
		},

		// custom OneAgent images
		{
			name:         "Custom CloudNative OneAgent image",
			dk:           dynakubeBuilder(dockerServer.URL).withCloudNativeFullStackCustomImage(server + "/" + testCustomOneAgentImage + ":" + testVersion).build(),
			component:    componentOneAgent,
			proxyWarning: false,
		},
		{
			name:         "Custom ClassicFullstack OneAgent image",
			dk:           dynakubeBuilder(dockerServer.URL).withClassicFullStackCustomImage(server + "/" + testCustomOneAgentImage + ":" + testVersion).build(),
			component:    componentOneAgent,
			proxyWarning: false,
		},
		{
			name:         "Custom HostMonitoring OneAgent image",
			dk:           dynakubeBuilder(dockerServer.URL).withHostMonitoringCustomImage(server + "/" + testCustomOneAgentImage + ":" + testVersion).build(),
			component:    componentOneAgent,
			proxyWarning: false,
		},

		// OneAgent code modules images
		{
			name:         "Custom CloudNative CodeModulesImage",
			dk:           dynakubeBuilder(dockerServer.URL).withCloudNativeCodeModulesImage(server + "/" + testOneAgentCodeModulesImage + ":" + testVersion).build(),
			component:    componentCodeModules,
			proxyWarning: true,
		},
		{
			name:         "Custom ApplicationMonitoring CodeModulesImage",
			dk:           dynakubeBuilder(dockerServer.URL).withApplicationMonitoringCodeModulesImage(server + "/" + testOneAgentCodeModulesImage + ":" + testVersion).build(),
			component:    componentCodeModules,
			proxyWarning: true,
		},
		// Active Gate images
		{
			name:         "ActiveGate default image",
			dk:           dynakubeBuilder(dockerServer.URL).withActiveGateCapability(activegate.RoutingCapability.DisplayName).build(),
			component:    componentActiveGate,
			proxyWarning: false,
		},
		{
			name:         "ActiveGate custom image",
			dk:           dynakubeBuilder(dockerServer.URL).withActiveGateCustomImage(server + "/" + testActiveGateCustomImage + ":" + testVersion).build(),
			component:    componentActiveGate,
			proxyWarning: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			logOutput := runWithTestLogger(func(log logd.Logger) {
				ctx := context.Background()
				clt := fake.NewClient(secret)
				pullSecret, _ := checkDynakube(ctx, log, clt, test.dk)
				keychain, _ := dockerkeychain.NewDockerKeychain(context.Background(), fake.NewClient(secret), pullSecret)

				transport, _ := createTransport(ctx, clt, test.dk, dockerServer.Client())
				pullImage := CreateImagePullFunc(ctx, keychain, transport)
				verifyImageIsAvailable(log, pullImage, test.dk, test.component, test.proxyWarning)
			})

			require.NotContains(t, logOutput, "failed")
			assert.Contains(t, logOutput, "can be successfully pulled")
		})
	}
}

func TestImageNotPullable(t *testing.T) {
	dockerServer, secret, server, err := setupDockerMocker(
		[]string{
			"/v2/",
		})
	require.NoError(t, err)
	defer dockerServer.Close()

	tests := []struct {
		name      string
		dk        *dynakube.DynaKube
		component component
	}{
		{
			name:      "OneAgent latest image for CloudNativeFullStack not available",
			dk:        dynakubeBuilder(dockerServer.URL).withCloudNativeFullStack().build(),
			component: componentOneAgent,
		},
		{
			name:      "OneAgent CloudNativeFullStack non-existing custom image",
			dk:        dynakubeBuilder(dockerServer.URL).withCloudNativeFullStackCustomImage(server + "/foobar").build(),
			component: componentOneAgent,
		},
		{
			name:      "OneAgent latest image for ClassicFullStack not available",
			dk:        dynakubeBuilder(dockerServer.URL).withClassicFullStack().build(),
			component: componentOneAgent,
		},
		{
			name:      "OneAgent ClassicFullStack non-existing custom image",
			dk:        dynakubeBuilder(dockerServer.URL).withClassicFullStackCustomImage(server + "/foobar").build(),
			component: componentOneAgent,
		},
		{
			name:      "OneAgent latest image for HostMonitoring not available",
			dk:        dynakubeBuilder(dockerServer.URL).withHostMonitoring().build(),
			component: componentOneAgent,
		},
		{
			name:      "OneAgent HostMonitoring non-existing custom image",
			dk:        dynakubeBuilder(dockerServer.URL).withHostMonitoringCustomImage(server + "/foobar").build(),
			component: componentOneAgent,
		},
		{
			name:      "OneAgent HostMonitoring non-existing server",
			dk:        dynakubeBuilder(dockerServer.URL).withHostMonitoringCustomImage("myunknownserver.com/foobar/image").build(),
			component: componentOneAgent,
		},
		{
			name: "ActiveGate image",
			dk: dynakubeBuilder(dockerServer.URL).
				withActiveGateCapability(activegate.RoutingCapability.DisplayName).
				build(),
			component: componentActiveGate,
		},
		{
			name: "ActiveGate custom image",
			dk: dynakubeBuilder(dockerServer.URL).
				withActiveGateCustomImage(server + "/" + testActiveGateCustomImage).
				withActiveGateCapability(activegate.RoutingCapability.DisplayName).
				build(),
			component: componentActiveGate,
		},
		{
			name: "ActiveGate custom image non-existing server",
			dk: dynakubeBuilder(dockerServer.URL).
				withActiveGateCustomImage("myunknownserver.com/foobar/image").
				withActiveGateCapability(activegate.RoutingCapability.DisplayName).
				build(),
			component: componentActiveGate,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			logOutput := runWithTestLogger(func(log logd.Logger) {
				ctx := context.Background()
				clt := fake.NewClient(secret)
				pullSecret, _ := checkDynakube(ctx, log, clt, test.dk)
				keychain, _ := dockerkeychain.NewDockerKeychain(context.Background(), fake.NewClient(secret), pullSecret)

				transport, _ := createTransport(ctx, clt, test.dk, dockerServer.Client())
				pullImage := CreateImagePullFunc(ctx, keychain, transport)
				verifyImageIsAvailable(log, pullImage, test.dk, test.component, false)
			})

			require.Contains(t, logOutput, "failed")

			if strings.Contains(test.name, "non-existing server") {
				assert.Contains(t, logOutput, "no such host")
			} else {
				assert.Contains(t, logOutput, "Bad Request")
			}

			assert.NotContains(t, logOutput, "can be successfully pulled")
		})
	}
}

func TestOneAgentCodeModulesImageNotPullable(t *testing.T) {
	dockerServer, secret, _, err := setupDockerMocker(
		[]string{
			"/v2/",
			"/v2/" + testOneAgentCodeModulesImage + "/manifests/latest",
			"/v2/" + testOneAgentCodeModulesImage + "/manifests/" + testVersion,
		})
	require.NoError(t, err)

	defer dockerServer.Close()

	t.Run("OneAgent code modules unreachable server", func(t *testing.T) {
		dk := *testNewDynakubeBuilder(testNamespace, testDynakube).
			withApiUrl(dockerServer.URL + "/api").
			withCloudNativeCodeModulesImage("myunknownserver.com/myrepo/mymissingcodemodules").
			build()
		logOutput := runWithTestLogger(func(log logd.Logger) {
			ctx := context.Background()
			clt := fake.NewClient(secret)
			pullSecret, _ := checkDynakube(ctx, log, clt, &dk)
			keychain, _ := dockerkeychain.NewDockerKeychain(context.Background(), fake.NewClient(secret), pullSecret)

			transport, _ := createTransport(ctx, clt, &dk, dockerServer.Client())
			pullImage := CreateImagePullFunc(ctx, keychain, transport)
			verifyImageIsAvailable(log, pullImage, &dk, componentCodeModules, true)
		})
		assert.Contains(t, logOutput, "failed")
		assert.Contains(t, logOutput, "no such host")
		assert.NotContains(t, logOutput, "can be successfully pulled")
	})

	t.Run("OneAgent code modules image with unset image", func(t *testing.T) {
		dk := *testNewDynakubeBuilder(testNamespace, testDynakube).
			withApiUrl(dockerServer.URL + "/api").
			withCloudNativeCodeModulesImage("").
			build()

		logOutput := runWithTestLogger(func(log logd.Logger) {
			ctx := context.Background()
			clt := fake.NewClient(secret)
			pullSecret, _ := checkDynakube(ctx, log, clt, &dk)
			keychain, _ := dockerkeychain.NewDockerKeychain(context.Background(), fake.NewClient(secret), pullSecret)
			transport, _ := createTransport(ctx, clt, &dk, dockerServer.Client())
			pullImage := CreateImagePullFunc(ctx, keychain, transport)
			verifyImageIsAvailable(log, pullImage, &dk, componentCodeModules, false)
		})
		assert.NotContains(t, logOutput, "Unknown OneAgentCodeModules image")
	})
}

func testDockerServerHandler(method string, serverUrls []string) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		for _, serverUrl := range serverUrls {
			if request.Method == method && request.URL.Path == serverUrl {
				writer.WriteHeader(http.StatusOK)

				return
			}
		}

		writer.WriteHeader(http.StatusBadRequest)
	}
}

func TestImagePullablePullSecret(t *testing.T) {
	t.Run("valid pull secret", func(t *testing.T) {
		pullSecret := testNewSecretBuilder(testNamespace, testDynakube+pullSecretSuffix).dataAppend(dtpullsecret.DockerConfigJson, pullSecretFieldValue).build()
		secret, err := getPullSecretToken(pullSecret)
		require.NoErrorf(t, err, "unexpected error")
		assert.Equal(t, pullSecretFieldValue, secret, "invalid contents of pull secret")
	})

	t.Run("invalid pull secret", func(t *testing.T) {
		pullSecret := testNewSecretBuilder(testNamespace, testDynakube+pullSecretSuffix).dataAppend("invalidToken", pullSecretFieldValue).build()
		secret, err := getPullSecretToken(pullSecret)
		require.Errorf(t, err, "expected error")
		assert.NotEqual(t, pullSecretFieldValue, secret, "valid contents of pull secret")
	})
}

func getPullSecretToken(pullSecret *corev1.Secret) (string, error) {
	secretBytes, hasPullSecret := pullSecret.Data[dtpullsecret.DockerConfigJson]
	if !hasPullSecret {
		return "", fmt.Errorf("token .dockerconfigjson does not exist in secret '%s'", pullSecret.Name)
	}

	secretStr := string(secretBytes)

	return secretStr, nil
}
