package troubleshoot

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/oci/dockerkeychain"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/dtpullsecret"
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
			"/v2/" + testOneAgentImage + "/manifests/" + "latest",
			"/v2/" + testOneAgentImage + "/manifests/" + testVersion,
			"/v2/" + testCustomOneAgentImage + "/manifests/" + "latest",
			"/v2/" + testCustomOneAgentImage + "/manifests/" + testVersion,
			"/v2/" + testOneAgentCodeModulesImage + "/manifests/latest",
			"/v2/" + testOneAgentCodeModulesImage + "/manifests/" + testVersion,
			"/v2/" + testActiveGateImage + "/manifests/" + "latest",
			"/v2/" + testActiveGateImage + "/manifests/" + testVersion,
			"/v2/" + testActiveGateCustomImage + "/manifests/" + "latest",
			"/v2/" + testActiveGateCustomImage + "/manifests/" + testVersion,
		})
	require.NoError(t, err)
	defer dockerServer.Close()

	tests := []struct {
		name         string
		dynaKube     *dynatracev1beta1.DynaKube
		component    component
		proxyWarning bool
	}{
		// standard OneAgent images
		{
			name:         "TestOneAgentImagePullable/OneAgent image",
			dynaKube:     dynakubeBuilder(dockerServer.URL).withCloudNativeFullStack().build(),
			component:    componentOneAgent,
			proxyWarning: false,
		},
		{
			name:         "TestOneAgentImagePullable/OneAgent versioned image for CloudNativeFullStack",
			dynaKube:     dynakubeBuilder(dockerServer.URL).withCloudNativeFullStackImageVersion(testVersion).build(),
			component:    componentOneAgent,
			proxyWarning: false,
		},
		{
			name:         "TestOneAgentImagePullable/OneAgent latest image for ClassicFullStack",
			dynaKube:     dynakubeBuilder(dockerServer.URL).withClassicFullStack().build(),
			component:    componentOneAgent,
			proxyWarning: false,
		},
		{
			name:         "TestOneAgentImagePullable/OneAgent versioned image for ClassicFullStack",
			dynaKube:     dynakubeBuilder(dockerServer.URL).withClassicFullStackImageVersion(testVersion).build(),
			component:    componentOneAgent,
			proxyWarning: false,
		},
		{
			name:         "TestOneAgentImagePullable/OneAgent latest image for HostMonitoring",
			dynaKube:     dynakubeBuilder(dockerServer.URL).withHostMonitoring().build(),
			component:    componentOneAgent,
			proxyWarning: false,
		},
		{
			name:         "TestOneAgentImagePullable/OneAgent versioned image for HostMonitoring",
			dynaKube:     dynakubeBuilder(dockerServer.URL).withHostMonitoringImageVersion(testVersion).build(),
			component:    componentOneAgent,
			proxyWarning: false,
		},

		// custom OneAgent images
		{
			name:         "TestCustomOneAgentImagePullable/OneAgent CloudNativeFullStack unversioned custom image",
			dynaKube:     dynakubeBuilder(dockerServer.URL).withCloudNativeFullStackCustomImage(server + "/" + testCustomOneAgentImage).build(),
			component:    componentOneAgent,
			proxyWarning: false,
		},
		{
			name:         "TestCustomOneAgentImagePullable/OneAgent CloudNativeFullStack latest custom image",
			dynaKube:     dynakubeBuilder(dockerServer.URL).withCloudNativeFullStackCustomImage(server + "/" + testCustomOneAgentImage + ":latest").build(),
			component:    componentOneAgent,
			proxyWarning: false,
		},
		{
			name:         "TestCustomOneAgentImagePullable/OneAgent CloudNativeFullStack versioned custom image",
			dynaKube:     dynakubeBuilder(dockerServer.URL).withCloudNativeFullStackCustomImage(server + "/" + testCustomOneAgentImage + ":" + testVersion).build(),
			component:    componentOneAgent,
			proxyWarning: false,
		},
		{
			name:         "TestCustomOneAgentImagePullable/OneAgent ClassicFullStack custom image",
			dynaKube:     dynakubeBuilder(dockerServer.URL).withClassicFullStackCustomImage(server + "/" + testCustomOneAgentImage).build(),
			component:    componentOneAgent,
			proxyWarning: false,
		},
		{
			name:         "TestCustomOneAgentImagePullable/OneAgent ClassicFullStack latest custom image",
			dynaKube:     dynakubeBuilder(dockerServer.URL).withClassicFullStackCustomImage(server + "/" + testCustomOneAgentImage + ":latest").build(),
			component:    componentOneAgent,
			proxyWarning: false,
		},
		{
			name:         "TestCustomOneAgentImagePullable/OneAgent ClassicFullStack versioned custom image",
			dynaKube:     dynakubeBuilder(dockerServer.URL).withClassicFullStackCustomImage(server + "/" + testCustomOneAgentImage + ":" + testVersion).build(),
			component:    componentOneAgent,
			proxyWarning: false,
		},
		{
			name:         "TestCustomOneAgentImagePullable/OneAgent HostMonitoring custom image",
			dynaKube:     dynakubeBuilder(dockerServer.URL).withHostMonitoringCustomImage(server + "/" + testCustomOneAgentImage).build(),
			component:    componentOneAgent,
			proxyWarning: false,
		},
		{
			name:         "TestCustomOneAgentImagePullable/OneAgent HostMonitoring latest custom image",
			dynaKube:     dynakubeBuilder(dockerServer.URL).withHostMonitoringCustomImage(server + "/" + testCustomOneAgentImage + ":latest").build(),
			component:    componentOneAgent,
			proxyWarning: false,
		},
		{
			name:         "TestCustomOneAgentImagePullable/OneAgent HostMonitoring versioned custom image",
			dynaKube:     dynakubeBuilder(dockerServer.URL).withHostMonitoringCustomImage(server + "/" + testCustomOneAgentImage + ":" + testVersion).build(),
			component:    componentOneAgent,
			proxyWarning: false,
		},

		// OneAgent code modules images
		{
			name:         "TestOneAgentCodeModulesImagePullable/OneAgent code modules image",
			dynaKube:     dynakubeBuilder(dockerServer.URL).withCloudNativeCodeModulesImage(server + "/" + testOneAgentCodeModulesImage + ":" + testVersion).build(),
			component:    componentOneAgent,
			proxyWarning: true,
		},
		{
			name:         "TestOneAgentCodeModulesImagePullable/CloudNativeFullStack OneAgent code modules image",
			dynaKube:     dynakubeBuilder(dockerServer.URL).withCloudNativeCodeModulesImage(server + "/" + testOneAgentCodeModulesImage + ":" + testVersion).build(),
			component:    componentOneAgent,
			proxyWarning: true,
		},
		{
			name:         "TestOneAgentCodeModulesImagePullable/CloudNativeFullStack OneAgent code modules latest image",
			dynaKube:     dynakubeBuilder(dockerServer.URL).withCloudNativeCodeModulesImage(server + "/" + testOneAgentCodeModulesImage + ":latest").build(),
			component:    componentOneAgent,
			proxyWarning: true,
		},
		{
			name:         "TestOneAgentCodeModulesImagePullable/CloudNativeFullStack OneAgent code modules unversioned image",
			dynaKube:     dynakubeBuilder(dockerServer.URL).withCloudNativeCodeModulesImage(server + "/" + testOneAgentCodeModulesImage).build(),
			component:    componentOneAgent,
			proxyWarning: true,
		},
		{
			name:         "TestOneAgentCodeModulesImagePullable/ApplicationMonitoring OneAgent code modules image",
			dynaKube:     dynakubeBuilder(dockerServer.URL).withApplicationMonitoringCodeModulesImage(server + "/" + testOneAgentCodeModulesImage + ":" + testVersion).build(),
			component:    componentOneAgent,
			proxyWarning: true,
		},
		{
			name:         "TestOneAgentCodeModulesImagePullable/ApplicationMonitoring OneAgent code modules latest image",
			dynaKube:     dynakubeBuilder(dockerServer.URL).withApplicationMonitoringCodeModulesImage(server + "/" + testOneAgentCodeModulesImage + ":latest").build(),
			component:    componentOneAgent,
			proxyWarning: true,
		},
		{
			name:         "TestOneAgentCodeModulesImagePullable/ApplicationMonitoring OneAgent code modules unversioned image",
			dynaKube:     dynakubeBuilder(dockerServer.URL).withApplicationMonitoringCodeModulesImage(server + "/" + testOneAgentCodeModulesImage).build(),
			component:    componentOneAgent,
			proxyWarning: true,
		},
		{
			name:         "TestOneAgentCodeModulesImagePullable/OneAgent code modules with non-existing image",
			dynaKube:     dynakubeBuilder(dockerServer.URL).withCloudNativeCodeModulesImage(server + "/non-existing-image").build(),
			component:    componentOneAgent,
			proxyWarning: true,
		},
		// Active Gate images
		{
			name:         "TestActiveGateImagePullable/ActiveGate image",
			dynaKube:     dynakubeBuilder(dockerServer.URL).build(),
			component:    componentActiveGate,
			proxyWarning: false,
		},
		{
			name:         "TestActiveGateImagePullable/ActiveGate custom image",
			dynaKube:     dynakubeBuilder(dockerServer.URL).withActiveGateCustomImage(server + "/" + testActiveGateCustomImage).build(),
			component:    componentActiveGate,
			proxyWarning: false,
		},
		{
			name:         "TestActiveGateImagePullable/ActiveGate latest custom image",
			dynaKube:     dynakubeBuilder(dockerServer.URL).withActiveGateCustomImage(server + "/" + testActiveGateCustomImage + ":latest").build(),
			component:    componentActiveGate,
			proxyWarning: false,
		},
		{
			name:         "TestActiveGateImagePullable/ActiveGate versioned custom image",
			dynaKube:     dynakubeBuilder(dockerServer.URL).withActiveGateCustomImage(server + "/" + testActiveGateCustomImage + ":" + testVersion).build(),
			component:    componentActiveGate,
			proxyWarning: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			logOutput := runWithTestLogger(func(log logr.Logger) {
				ctx := context.Background()
				clt := fake.NewClient(secret)
				pullSecret, _ := checkDynakube(ctx, log, clt, test.dynaKube)
				keychain, _ := dockerkeychain.NewDockerKeychain(context.Background(), fake.NewClient(secret), pullSecret)

				transport, _ := createTransport(ctx, clt, test.dynaKube, dockerServer.Client())
				pullImage := CreateImagePullFunc(ctx, keychain, transport)
				verifyImageIsAvailable(log, pullImage, test.dynaKube, test.component, test.proxyWarning)
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
		dynaKube  *dynatracev1beta1.DynaKube
		component component
	}{
		{
			name:      "OneAgent latest image for CloudNativeFullStack not available",
			dynaKube:  dynakubeBuilder(dockerServer.URL).withCloudNativeFullStack().build(),
			component: componentOneAgent,
		},
		{
			name:      "OneAgent CloudNativeFullStack non-existing custom image",
			dynaKube:  dynakubeBuilder(dockerServer.URL).withCloudNativeFullStackCustomImage(server + "/foobar").build(),
			component: componentOneAgent,
		},
		{
			name:      "OneAgent latest image for ClassicFullStack not available",
			dynaKube:  dynakubeBuilder(dockerServer.URL).withClassicFullStack().build(),
			component: componentOneAgent,
		},
		{
			name:      "OneAgent ClassicFullStack non-existing custom image",
			dynaKube:  dynakubeBuilder(dockerServer.URL).withClassicFullStackCustomImage(server + "/foobar").build(),
			component: componentOneAgent,
		},
		{
			name:      "OneAgent latest image for HostMonitoring not available",
			dynaKube:  dynakubeBuilder(dockerServer.URL).withHostMonitoring().build(),
			component: componentOneAgent,
		},
		{
			name:      "OneAgent HostMonitoring non-existing custom image",
			dynaKube:  dynakubeBuilder(dockerServer.URL).withHostMonitoringCustomImage(server + "/foobar").build(),
			component: componentOneAgent,
		},
		{
			name:      "OneAgent HostMonitoring non-existing server",
			dynaKube:  dynakubeBuilder(dockerServer.URL).withHostMonitoringCustomImage("myunknownserver.com/foobar/image").build(),
			component: componentOneAgent,
		},
		{
			name:      "ActiveGate image",
			dynaKube:  dynakubeBuilder(dockerServer.URL).build(),
			component: componentActiveGate,
		},
		{
			name: "ActiveGate custom image",
			dynaKube: dynakubeBuilder(dockerServer.URL).
				withActiveGateCustomImage(server + "/" + testActiveGateCustomImage).
				withActiveGateCapability(dynatracev1beta1.RoutingCapability.DisplayName).
				build(),
			component: componentActiveGate,
		},
		{
			name: "ActiveGate custom image non-existing server",
			dynaKube: dynakubeBuilder(dockerServer.URL).
				withActiveGateCustomImage("myunknownserver.com/foobar/image").
				withActiveGateCapability(dynatracev1beta1.RoutingCapability.DisplayName).
				build(),
			component: componentActiveGate,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			logOutput := runWithTestLogger(func(log logr.Logger) {
				ctx := context.Background()
				clt := fake.NewClient(secret)
				pullSecret, _ := checkDynakube(ctx, log, clt, test.dynaKube)
				keychain, _ := dockerkeychain.NewDockerKeychain(context.Background(), fake.NewClient(secret), pullSecret)

				transport, _ := createTransport(ctx, clt, test.dynaKube, dockerServer.Client())
				pullImage := CreateImagePullFunc(ctx, keychain, transport)
				verifyImageIsAvailable(log, pullImage, test.dynaKube, test.component, false)
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
		dynakube := *testNewDynakubeBuilder(testNamespace, testDynakube).
			withApiUrl(dockerServer.URL + "/api").
			withCloudNativeCodeModulesImage("myunknownserver.com/myrepo/mymissingcodemodules").
			build()
		logOutput := runWithTestLogger(func(log logr.Logger) {
			ctx := context.Background()
			clt := fake.NewClient(secret)
			pullSecret, _ := checkDynakube(ctx, log, clt, &dynakube)
			keychain, _ := dockerkeychain.NewDockerKeychain(context.Background(), fake.NewClient(secret), pullSecret)

			transport, _ := createTransport(ctx, clt, &dynakube, dockerServer.Client())
			pullImage := CreateImagePullFunc(ctx, keychain, transport)
			verifyImageIsAvailable(log, pullImage, &dynakube, componentCodeModules, true)
		})
		assert.Contains(t, logOutput, "failed")
		assert.Contains(t, logOutput, "no such host")
		assert.NotContains(t, logOutput, "can be successfully pulled")
	})

	t.Run("OneAgent code modules image with unset image", func(t *testing.T) {
		dynakube := *testNewDynakubeBuilder(testNamespace, testDynakube).
			withApiUrl(dockerServer.URL + "/api").
			withCloudNativeCodeModulesImage("").
			build()

		logOutput := runWithTestLogger(func(log logr.Logger) {
			ctx := context.Background()
			clt := fake.NewClient(secret)
			pullSecret, _ := checkDynakube(ctx, log, clt, &dynakube)
			keychain, _ := dockerkeychain.NewDockerKeychain(context.Background(), fake.NewClient(secret), pullSecret)
			transport, _ := createTransport(ctx, clt, &dynakube, dockerServer.Client())
			pullImage := CreateImagePullFunc(ctx, keychain, transport)
			verifyImageIsAvailable(log, pullImage, &dynakube, componentCodeModules, false)
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
