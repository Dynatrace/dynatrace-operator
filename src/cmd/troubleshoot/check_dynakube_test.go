package troubleshoot

import (
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/scheme"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	testRegistry              = "testing.dev.dynatracelabs.com"
	testApiUrl                = "https://" + testRegistry + "/api"
	testOtherApiUrl           = "https://" + testRegistry + "/otherapi"
	testDynatraceSecret       = testDynakube
	testOtherDynatraceSecret  = "otherDynatraceSecret"
	testApiToken              = "apiTokenValue"
	testPaasToken             = "passTokenValue"
	testSecretName            = "customSecret"
	testCustomPullSecretToken = "secretTokenValue"
)

func TestDynakube(t *testing.T) {
	// TODO checkDynakubeCrdExists test. How to mock apiReader.List

	t.Run("dynakube exists", func(t *testing.T) {
		troubleshootContext := TestData{namespaceName: testNamespace, dynakubeName: testDynakube}

		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(
				testDynakubeBuilder(testNamespace, testDynakube).build(),
				testBuildNamespace(testNamespace),
			).
			Build()

		assert.NoErrorf(t, checkSelectedDynakubeExists(clt, &troubleshootContext), "no dynakube found")
	})
	t.Run("dynakube does not exist", func(t *testing.T) {
		troubleshootContext := TestData{namespaceName: testNamespace, dynakubeName: testOtherDynakube}

		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(
				testDynakubeBuilder(testNamespace, testDynakube).build(),
				testBuildNamespace(testNamespace),
			).
			Build()

		assert.Errorf(t, checkSelectedDynakubeExists(clt, &troubleshootContext), "dynakube found")
	})
	t.Run("invalid namespace selected", func(t *testing.T) {
		troubleshootContext := TestData{namespaceName: testOtherNamespace, dynakubeName: testDynakube}

		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(
				testDynakubeBuilder(testNamespace, testDynakube).build(),
				testBuildNamespace(testNamespace),
			).
			Build()

		assert.Errorf(t, checkSelectedDynakubeExists(clt, &troubleshootContext), "dynakube found")
	})
}

func TestApiUrl(t *testing.T) {
	t.Run("valid ApiUrl", func(t *testing.T) {
		troubleshootContext := TestData{namespaceName: testNamespace, dynakubeName: testDynakube}

		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(
				testDynakubeBuilder(testNamespace, testDynakube).withApiUrl(testApiUrl).build(),
				testBuildNamespace(testNamespace),
			).
			Build()

		assert.NoErrorf(t, checkApiUrl(clt, &troubleshootContext), "invalid ApiUrl")
	})
	t.Run("invalid ApiUrl", func(t *testing.T) {
		troubleshootContext := TestData{namespaceName: testNamespace, dynakubeName: testDynakube}

		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(
				testDynakubeBuilder(testNamespace, testDynakube).withApiUrl(testOtherApiUrl).build(),
				testBuildNamespace(testNamespace),
			).
			Build()

		assert.Errorf(t, checkApiUrl(clt, &troubleshootContext), "valid ApiUrl")
	})
}

func TestDynatraceSecret(t *testing.T) {
	t.Run("default name of Dynatrace secret", func(t *testing.T) {
		troubleshootContext := TestData{namespaceName: testNamespace, dynakubeName: testDynakube}

		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(
				testDynakubeBuilder(testNamespace, testDynakube).withTokens(testDynatraceSecret).build(),
				testBuildNamespace(testNamespace),
			).
			Build()

		assert.NoErrorf(t, checkDynatraceApiSecret(clt, &troubleshootContext), "missing dynakube")
		assert.Equal(t, testDynakube, troubleshootContext.dynatraceApiSecretName)
	})
	t.Run("custom name of Dynatrace secret", func(t *testing.T) {
		troubleshootContext := TestData{namespaceName: testNamespace, dynakubeName: testDynakube}

		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(
				testDynakubeBuilder(testNamespace, testDynakube).withTokens(testOtherDynatraceSecret).build(),
				testBuildNamespace(testNamespace),
			).
			Build()

		assert.NoErrorf(t, checkDynatraceApiSecret(clt, &troubleshootContext), "missing dynakube")
		assert.Equal(t, testOtherDynatraceSecret, troubleshootContext.dynatraceApiSecretName)
	})

	t.Run("Dynatrace secret exists", func(t *testing.T) {
		troubleshootContext := TestData{namespaceName: testNamespace, dynakubeName: testDynakube}

		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(
				testDynakubeBuilder(testNamespace, testDynakube).withTokens(testDynatraceSecret).build(),
				testBuildNamespace(testNamespace),
				testSecretBuilder(testNamespace, testDynatraceSecret).build(),
			).
			Build()

		assert.NoErrorf(t, checkSelectedDynakubeExists(clt, &troubleshootContext), "Dynatrace secret not found")
	})
	t.Run("Dynatrace secret does not exist", func(t *testing.T) {
		troubleshootContext := TestData{namespaceName: testNamespace, dynakubeName: testDynakube}

		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(
				testDynakubeBuilder(testNamespace, testDynakube).withTokens(testDynatraceSecret).build(),
				testBuildNamespace(testNamespace),
				testSecretBuilder(testNamespace, testOtherDynatraceSecret).build(),
			).
			Build()

		assert.Errorf(t, checkIfDynatraceApiSecretWithTheGivenNameExists(clt, &troubleshootContext), "Dynatrace secret found")
	})

	t.Run("Dynatrace secret has apiToken token", func(t *testing.T) {
		troubleshootContext := TestData{namespaceName: testNamespace, dynakubeName: testDynakube, dynatraceApiSecretName: testDynatraceSecret}

		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(
				testDynakubeBuilder(testNamespace, testDynakube).withTokens(testDynatraceSecret).build(),
				testBuildNamespace(testNamespace),
				testSecretBuilder(testNamespace, testDynatraceSecret).dataAppend("apiToken", testApiToken).dataAppend("paasToken", testPaasToken).build(),
			).
			Build()

		assert.NoErrorf(t, checkIfDynatraceApiSecretHasApiToken(clt, &troubleshootContext), "Dynatrace secret does not have required tokens")
	})
	t.Run("Dynatrace secret - apiToken is missing", func(t *testing.T) {
		troubleshootContext := TestData{namespaceName: testNamespace, dynakubeName: testDynakube, dynatraceApiSecretName: testDynatraceSecret}

		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(
				testDynakubeBuilder(testNamespace, testDynakube).withTokens(testDynatraceSecret).build(),
				testBuildNamespace(testNamespace),
				testSecretBuilder(testNamespace, testDynatraceSecret).dataAppend("paasToken", testPaasToken).build(),
			).
			Build()

		assert.Errorf(t, checkIfDynatraceApiSecretHasApiToken(clt, &troubleshootContext), "Dynatrace secret does not have apiToken")
	})
	t.Run("Dynatrace secret has paasToken token", func(t *testing.T) {
		troubleshootContext := TestData{namespaceName: testNamespace, dynakubeName: testDynakube, dynatraceApiSecretName: testDynatraceSecret}

		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(
				testDynakubeBuilder(testNamespace, testDynakube).withTokens(testDynatraceSecret).build(),
				testBuildNamespace(testNamespace),
				testSecretBuilder(testNamespace, testDynatraceSecret).dataAppend("apiToken", testApiToken).dataAppend("paasToken", testPaasToken).build(),
			).
			Build()

		assert.NoErrorf(t, checkIfDynatraceApiSecretHasPaasToken(clt, &troubleshootContext), "Dynatrace secret does not have required tokens")
	})
	t.Run("Dynatrace secret - paasToken is missing", func(t *testing.T) {
		troubleshootContext := TestData{namespaceName: testNamespace, dynakubeName: testDynakube, dynatraceApiSecretName: testDynatraceSecret}

		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(
				testDynakubeBuilder(testNamespace, testDynakube).withTokens(testDynatraceSecret).build(),
				testBuildNamespace(testNamespace),
				testSecretBuilder(testNamespace, testDynatraceSecret).dataAppend("apiToken", testApiToken).build(),
			).
			Build()

		assert.Errorf(t, checkIfDynatraceApiSecretHasPaasToken(clt, &troubleshootContext), "Dynatrace secret does not have paasToken")
	})
}

func TestCustomPullSecret(t *testing.T) {
	t.Run("no custom pull secret", func(t *testing.T) {
		troubleshootContext := TestData{namespaceName: testNamespace, dynakubeName: testDynakube}

		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(
				testDynakubeBuilder(testNamespace, testDynakube).build(),
				testBuildNamespace(testNamespace),
			).
			Build()

		assert.NoErrorf(t, checkCustomPullSecret(clt, &troubleshootContext), "missing dynakube")
		assert.Equal(t, "", troubleshootContext.customPullSecretName)
	})
	t.Run("custom pull secret defined", func(t *testing.T) {
		troubleshootContext := TestData{namespaceName: testNamespace, dynakubeName: testDynakube}

		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(
				testDynakubeBuilder(testNamespace, testDynakube).withCustomPullSecret(testSecretName).build(),
				testBuildNamespace(testNamespace),
			).
			Build()

		assert.NoErrorf(t, checkCustomPullSecret(clt, &troubleshootContext), "missing dynakube")
		assert.Equal(t, testSecretName, troubleshootContext.customPullSecretName)
	})
	t.Run("custom pull secret exists", func(t *testing.T) {
		troubleshootContext := TestData{namespaceName: testNamespace, dynakubeName: testDynakube, customPullSecretName: testSecretName}

		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(
				testDynakubeBuilder(testNamespace, testDynakube).withCustomPullSecret(testSecretName).build(),
				testBuildNamespace(testNamespace),
				testSecretBuilder(testNamespace, testSecretName).build(),
			).
			Build()

		assert.NoErrorf(t, checkIfCustomPullSecretWithTheGivenNameExists(clt, &troubleshootContext), "custom pull secret not found")
	})
	t.Run("custom pull secret does not exist", func(t *testing.T) {
		troubleshootContext := TestData{namespaceName: testNamespace, dynakubeName: testDynakube, customPullSecretName: testSecretName}

		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(
				testDynakubeBuilder(testNamespace, testDynakube).withCustomPullSecret(testSecretName).build(),
				testBuildNamespace(testNamespace),
			).
			Build()

		assert.Errorf(t, checkIfCustomPullSecretWithTheGivenNameExists(clt, &troubleshootContext), "custom pull secret found")
	})
	t.Run("custom pull secret has required tokens", func(t *testing.T) {
		troubleshootContext := TestData{namespaceName: testNamespace, dynakubeName: testDynakube, customPullSecretName: testSecretName}

		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(
				testDynakubeBuilder(testNamespace, testDynakube).withCustomPullSecret(testSecretName).build(),
				testBuildNamespace(testNamespace),
				testSecretBuilder(testNamespace, testSecretName).dataAppend(".dockerconfigjson", testCustomPullSecretToken).build(),
			).
			Build()

		assert.NoErrorf(t, checkCustomPullSecretHasRequiredTokens(clt, &troubleshootContext), "custom pull secret does not have required tokens")
	})
	t.Run("custom pull secret does not have required tokens", func(t *testing.T) {
		troubleshootContext := TestData{namespaceName: testNamespace, dynakubeName: testDynakube, customPullSecretName: testSecretName}

		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(
				testDynakubeBuilder(testNamespace, testDynakube).withCustomPullSecret(testSecretName).build(),
				testBuildNamespace(testNamespace),
				testSecretBuilder(testNamespace, testSecretName).build(),
			).
			Build()

		assert.Errorf(t, checkCustomPullSecretHasRequiredTokens(clt, &troubleshootContext), "custom pull secret has required tokens")
	})
}

func TestProxySecret(t *testing.T) {
	t.Run("no proxy secret", func(t *testing.T) {
		troubleshootContext := TestData{namespaceName: testNamespace, dynakubeName: testDynakube}

		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(
				testDynakubeBuilder(testNamespace, testDynakube).build(),
				testBuildNamespace(testNamespace),
			).
			Build()

		assert.NoErrorf(t, checkProxySecret(clt, &troubleshootContext), "missing dynakube")
		assert.Equal(t, "", troubleshootContext.proxySecretName)
	})
	t.Run("proxy secret defined", func(t *testing.T) {
		troubleshootContext := TestData{namespaceName: testNamespace, dynakubeName: testDynakube}

		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(
				testDynakubeBuilder(testNamespace, testDynakube).withProxySecret(testSecretName).build(),
				testBuildNamespace(testNamespace),
			).
			Build()

		assert.NoErrorf(t, checkProxySecret(clt, &troubleshootContext), "missing dynakube")
		assert.Equal(t, testSecretName, troubleshootContext.proxySecretName)
	})
	t.Run("proxy secret exists", func(t *testing.T) {
		troubleshootContext := TestData{namespaceName: testNamespace, dynakubeName: testDynakube, proxySecretName: testSecretName}

		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(
				testDynakubeBuilder(testNamespace, testDynakube).withProxySecret(testSecretName).build(),
				testBuildNamespace(testNamespace),
				testSecretBuilder(testNamespace, testSecretName).build(),
			).
			Build()

		assert.NoErrorf(t, checkIfProxySecretWithTheGivenNameExists(clt, &troubleshootContext), "proxy secret not found")
	})
	t.Run("proxy secret does not exist", func(t *testing.T) {
		troubleshootContext := TestData{namespaceName: testNamespace, dynakubeName: testDynakube, proxySecretName: testSecretName}

		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(
				testDynakubeBuilder(testNamespace, testDynakube).withProxySecret(testSecretName).build(),
				testBuildNamespace(testNamespace),
			).
			Build()

		assert.Errorf(t, checkIfProxySecretWithTheGivenNameExists(clt, &troubleshootContext), "proxy secret found")
	})
	t.Run("proxy secret has required tokens", func(t *testing.T) {
		troubleshootContext := TestData{namespaceName: testNamespace, dynakubeName: testDynakube, proxySecretName: testSecretName}

		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(
				testDynakubeBuilder(testNamespace, testDynakube).withProxySecret(testSecretName).build(),
				testBuildNamespace(testNamespace),
				testSecretBuilder(testNamespace, testSecretName).dataAppend(dtclient.CustomProxySecretKey, testCustomPullSecretToken).build(),
			).
			Build()

		assert.NoErrorf(t, checkProxySecretHasRequiredTokens(clt, &troubleshootContext), "proxy secret does not have required tokens")
	})
	t.Run("proxy secret does not have required tokens", func(t *testing.T) {
		troubleshootContext := TestData{namespaceName: testNamespace, dynakubeName: testDynakube, proxySecretName: testSecretName}

		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(
				testDynakubeBuilder(testNamespace, testDynakube).withProxySecret(testSecretName).build(),
				testBuildNamespace(testNamespace),
				testSecretBuilder(testNamespace, testSecretName).build(),
			).
			Build()

		assert.Errorf(t, checkProxySecretHasRequiredTokens(clt, &troubleshootContext), "proxy secret has required tokens")
	})
}

type TestDynaKubeBuilder struct {
	dynakube *dynatracev1beta1.DynaKube
}

func testDynakubeBuilder(namespace string, dynakube string) *TestDynaKubeBuilder {
	return &TestDynaKubeBuilder{
		dynakube: &dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      dynakube,
			},
		},
	}
}

func (builder *TestDynaKubeBuilder) withApiUrl(apiUrl string) *TestDynaKubeBuilder {
	builder.dynakube.Spec.APIURL = apiUrl
	return builder
}

func (builder *TestDynaKubeBuilder) withTokens(secretName string) *TestDynaKubeBuilder {
	builder.dynakube.Spec.Tokens = secretName
	return builder
}

func (builder *TestDynaKubeBuilder) withCustomPullSecret(secretName string) *TestDynaKubeBuilder {
	builder.dynakube.Spec.CustomPullSecret = secretName
	return builder
}

func (builder *TestDynaKubeBuilder) withProxySecret(secretName string) *TestDynaKubeBuilder {
	builder.dynakube.Spec.Proxy = &dynatracev1beta1.DynaKubeProxy{
		ValueFrom: secretName,
	}
	return builder
}

func (builder *TestDynaKubeBuilder) withActiveGateImage(image string) *TestDynaKubeBuilder {
	builder.dynakube.Spec.ActiveGate.Image = image
	return builder
}

func (builder *TestDynaKubeBuilder) withClassicFullStackImage(image string) *TestDynaKubeBuilder {
	builder.dynakube.Spec.OneAgent.ClassicFullStack = &dynatracev1beta1.HostInjectSpec{
		Image: image,
	}
	return builder
}

func (builder *TestDynaKubeBuilder) withCloudNativeFullStackImage(image string) *TestDynaKubeBuilder {
	builder.dynakube.Spec.OneAgent.CloudNativeFullStack = &dynatracev1beta1.CloudNativeFullStackSpec{
		HostInjectSpec: dynatracev1beta1.HostInjectSpec{
			Image: image,
		},
	}
	return builder
}

func (builder *TestDynaKubeBuilder) withHostMonitoringImage(image string) *TestDynaKubeBuilder {
	builder.dynakube.Spec.OneAgent.HostMonitoring = &dynatracev1beta1.HostInjectSpec{
		Image: image,
	}
	return builder
}

func (builder *TestDynaKubeBuilder) build() *dynatracev1beta1.DynaKube {
	return builder.dynakube
}

type TestSecretBuilder struct {
	secret *corev1.Secret
}

func testSecretBuilder(namespace string, name string) *TestSecretBuilder {
	return &TestSecretBuilder{
		secret: &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      name,
			},
		},
	}
}

func (builder *TestSecretBuilder) dataAppend(key string, value string) *TestSecretBuilder {
	if builder.secret.Data == nil {
		builder.secret.Data = make(map[string][]byte)
		builder.secret.Data[key] = []byte(value)
	} else {
		builder.secret.Data[key] = []byte(value)
	}
	return builder
}

func (builder *TestSecretBuilder) build() *corev1.Secret {
	return builder.secret
}

func testBuildNamespace(namespace string) *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
			UID:  testUID,
		},
	}
}
