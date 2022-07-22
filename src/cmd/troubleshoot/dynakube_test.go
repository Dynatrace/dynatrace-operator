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
		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(
				testNewDynakubeBuilder(testNamespace, testDynakube).build(),
				testBuildNamespace(testNamespace),
			).
			Build()

		troubleshootCtx := troubleshootContext{apiReader: clt, namespaceName: testNamespace, dynakubeName: testDynakube}
		assert.NoErrorf(t, checkSelectedDynakubeExists(&troubleshootCtx), "no dynakube found")
	})
	t.Run("dynakube does not exist", func(t *testing.T) {
		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(
				testNewDynakubeBuilder(testNamespace, testDynakube).build(),
				testBuildNamespace(testNamespace),
			).
			Build()

		troubleshootCtx := troubleshootContext{apiReader: clt, namespaceName: testNamespace, dynakubeName: testOtherDynakube}
		assert.Errorf(t, checkSelectedDynakubeExists(&troubleshootCtx), "dynakube found")
	})
	t.Run("invalid namespace selected", func(t *testing.T) {
		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(
				testNewDynakubeBuilder(testNamespace, testDynakube).build(),
				testBuildNamespace(testNamespace),
			).
			Build()

		troubleshootCtx := troubleshootContext{apiReader: clt, namespaceName: testOtherNamespace, dynakubeName: testDynakube}
		assert.Errorf(t, checkSelectedDynakubeExists(&troubleshootCtx), "dynakube found")
	})
}

func TestApiUrl(t *testing.T) {
	t.Run("valid ApiUrl", func(t *testing.T) {
		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(
				testNewDynakubeBuilder(testNamespace, testDynakube).withApiUrl(testApiUrl).build(),
				testBuildNamespace(testNamespace),
			).
			Build()

		troubleshootCtx := troubleshootContext{apiReader: clt, namespaceName: testNamespace, dynakubeName: testDynakube}
		assert.NoErrorf(t, checkApiUrl(&troubleshootCtx), "invalid ApiUrl")
	})
	t.Run("invalid ApiUrl", func(t *testing.T) {
		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(
				testNewDynakubeBuilder(testNamespace, testDynakube).withApiUrl(testOtherApiUrl).build(),
				testBuildNamespace(testNamespace),
			).
			Build()

		troubleshootCtx := troubleshootContext{apiReader: clt, namespaceName: testNamespace, dynakubeName: testDynakube}
		assert.Errorf(t, checkApiUrl(&troubleshootCtx), "valid ApiUrl")
	})
}

func TestDynatraceSecret(t *testing.T) {
	t.Run("default name of Dynatrace secret", func(t *testing.T) {
		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(
				testNewDynakubeBuilder(testNamespace, testDynakube).withTokens(testDynatraceSecret).build(),
				testBuildNamespace(testNamespace),
			).
			Build()

		troubleshootCtx := troubleshootContext{apiReader: clt, namespaceName: testNamespace, dynakubeName: testDynakube}
		assert.NoErrorf(t, evaluateDynatraceApiSecretName(&troubleshootCtx), "missing dynakube")
		assert.Equal(t, testDynakube, troubleshootCtx.dynatraceApiSecretName)
	})
	t.Run("custom name of Dynatrace secret", func(t *testing.T) {
		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(
				testNewDynakubeBuilder(testNamespace, testDynakube).withTokens(testOtherDynatraceSecret).build(),
				testBuildNamespace(testNamespace),
			).
			Build()

		troubleshootCtx := troubleshootContext{apiReader: clt, namespaceName: testNamespace, dynakubeName: testDynakube}
		assert.NoErrorf(t, evaluateDynatraceApiSecretName(&troubleshootCtx), "missing dynakube")
		assert.Equal(t, testOtherDynatraceSecret, troubleshootCtx.dynatraceApiSecretName)
	})

	t.Run("Dynatrace secret exists", func(t *testing.T) {
		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(
				testNewDynakubeBuilder(testNamespace, testDynakube).withTokens(testDynatraceSecret).build(),
				testBuildNamespace(testNamespace),
				testNewSecretBuilder(testNamespace, testDynatraceSecret).build(),
			).
			Build()

		troubleshootCtx := troubleshootContext{apiReader: clt, namespaceName: testNamespace, dynakubeName: testDynakube}
		assert.NoErrorf(t, checkSelectedDynakubeExists(&troubleshootCtx), "Dynatrace secret not found")
	})
	t.Run("Dynatrace secret does not exist", func(t *testing.T) {
		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(
				testNewDynakubeBuilder(testNamespace, testDynakube).withTokens(testDynatraceSecret).build(),
				testBuildNamespace(testNamespace),
				testNewSecretBuilder(testNamespace, testOtherDynatraceSecret).build(),
			).
			Build()

		troubleshootCtx := troubleshootContext{apiReader: clt, namespaceName: testNamespace, dynakubeName: testDynakube}
		assert.Errorf(t, checkIfDynatraceApiSecretWithTheGivenNameExists(&troubleshootCtx), "Dynatrace secret found")
	})

	t.Run("Dynatrace secret has apiToken token", func(t *testing.T) {
		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(
				testNewDynakubeBuilder(testNamespace, testDynakube).withTokens(testDynatraceSecret).build(),
				testBuildNamespace(testNamespace),
				testNewSecretBuilder(testNamespace, testDynatraceSecret).dataAppend("apiToken", testApiToken).dataAppend("paasToken", testPaasToken).build(),
			).
			Build()

		troubleshootCtx := troubleshootContext{apiReader: clt, namespaceName: testNamespace, dynakubeName: testDynakube, dynatraceApiSecretName: testDynatraceSecret}
		assert.NoErrorf(t, checkIfDynatraceApiSecretHasApiToken(&troubleshootCtx), "Dynatrace secret does not have required tokens")
	})
	t.Run("Dynatrace secret - apiToken is missing", func(t *testing.T) {
		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(
				testNewDynakubeBuilder(testNamespace, testDynakube).withTokens(testDynatraceSecret).build(),
				testBuildNamespace(testNamespace),
				testNewSecretBuilder(testNamespace, testDynatraceSecret).dataAppend("paasToken", testPaasToken).build(),
			).
			Build()

		troubleshootCtx := troubleshootContext{apiReader: clt, namespaceName: testNamespace, dynakubeName: testDynakube, dynatraceApiSecretName: testDynatraceSecret}
		assert.Errorf(t, checkIfDynatraceApiSecretHasApiToken(&troubleshootCtx), "Dynatrace secret does not have apiToken")
	})
	t.Run("Dynatrace secret has paasToken token", func(t *testing.T) {
		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(
				testNewDynakubeBuilder(testNamespace, testDynakube).withTokens(testDynatraceSecret).build(),
				testBuildNamespace(testNamespace),
				testNewSecretBuilder(testNamespace, testDynatraceSecret).dataAppend("apiToken", testApiToken).dataAppend("paasToken", testPaasToken).build(),
			).
			Build()

		troubleshootCtx := troubleshootContext{apiReader: clt, namespaceName: testNamespace, dynakubeName: testDynakube, dynatraceApiSecretName: testDynatraceSecret}
		assert.NoErrorf(t, checkIfDynatraceApiSecretHasPaasToken(&troubleshootCtx), "Dynatrace secret does not have required tokens")
	})
	t.Run("Dynatrace secret - paasToken is missing", func(t *testing.T) {
		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(
				testNewDynakubeBuilder(testNamespace, testDynakube).withTokens(testDynatraceSecret).build(),
				testBuildNamespace(testNamespace),
				testNewSecretBuilder(testNamespace, testDynatraceSecret).dataAppend("apiToken", testApiToken).build(),
			).
			Build()

		troubleshootCtx := troubleshootContext{apiReader: clt, namespaceName: testNamespace, dynakubeName: testDynakube, dynatraceApiSecretName: testDynatraceSecret}
		assert.Errorf(t, checkIfDynatraceApiSecretHasPaasToken(&troubleshootCtx), "Dynatrace secret does not have paasToken")
	})
}

func TestCustomPullSecret(t *testing.T) {
	t.Run("no custom pull secret", func(t *testing.T) {
		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(
				testNewDynakubeBuilder(testNamespace, testDynakube).build(),
				testBuildNamespace(testNamespace),
			).
			Build()

		troubleshootCtx := troubleshootContext{apiReader: clt, namespaceName: testNamespace, dynakubeName: testDynakube}
		assert.NoErrorf(t, checkCustomPullSecret(&troubleshootCtx), "missing dynakube")
		assert.Equal(t, "", troubleshootCtx.customPullSecretName)
	})
	t.Run("custom pull secret defined", func(t *testing.T) {
		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(
				testNewDynakubeBuilder(testNamespace, testDynakube).withCustomPullSecret(testSecretName).build(),
				testBuildNamespace(testNamespace),
			).
			Build()

		troubleshootCtx := troubleshootContext{apiReader: clt, namespaceName: testNamespace, dynakubeName: testDynakube}
		assert.NoErrorf(t, checkCustomPullSecret(&troubleshootCtx), "missing dynakube")
		assert.Equal(t, testSecretName, troubleshootCtx.customPullSecretName)
	})
	t.Run("custom pull secret exists", func(t *testing.T) {
		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(
				testNewDynakubeBuilder(testNamespace, testDynakube).withCustomPullSecret(testSecretName).build(),
				testBuildNamespace(testNamespace),
				testNewSecretBuilder(testNamespace, testSecretName).build(),
			).
			Build()

		troubleshootCtx := troubleshootContext{apiReader: clt, namespaceName: testNamespace, dynakubeName: testDynakube, customPullSecretName: testSecretName}
		assert.NoErrorf(t, checkIfCustomPullSecretWithTheGivenNameExists(&troubleshootCtx), "custom pull secret not found")
	})
	t.Run("custom pull secret does not exist", func(t *testing.T) {
		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(
				testNewDynakubeBuilder(testNamespace, testDynakube).withCustomPullSecret(testSecretName).build(),
				testBuildNamespace(testNamespace),
			).
			Build()

		troubleshootCtx := troubleshootContext{apiReader: clt, namespaceName: testNamespace, dynakubeName: testDynakube, customPullSecretName: testSecretName}
		assert.Errorf(t, checkIfCustomPullSecretWithTheGivenNameExists(&troubleshootCtx), "custom pull secret found")
	})
	t.Run("custom pull secret has required tokens", func(t *testing.T) {
		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(
				testNewDynakubeBuilder(testNamespace, testDynakube).withCustomPullSecret(testSecretName).build(),
				testBuildNamespace(testNamespace),
				testNewSecretBuilder(testNamespace, testSecretName).dataAppend(".dockerconfigjson", testCustomPullSecretToken).build(),
			).
			Build()

		troubleshootCtx := troubleshootContext{apiReader: clt, namespaceName: testNamespace, dynakubeName: testDynakube, customPullSecretName: testSecretName}
		assert.NoErrorf(t, checkCustomPullSecretHasRequiredTokens(&troubleshootCtx), "custom pull secret does not have required tokens")
	})
	t.Run("custom pull secret does not have required tokens", func(t *testing.T) {
		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(
				testNewDynakubeBuilder(testNamespace, testDynakube).withCustomPullSecret(testSecretName).build(),
				testBuildNamespace(testNamespace),
				testNewSecretBuilder(testNamespace, testSecretName).build(),
			).
			Build()

		troubleshootCtx := troubleshootContext{apiReader: clt, namespaceName: testNamespace, dynakubeName: testDynakube, customPullSecretName: testSecretName}
		assert.Errorf(t, checkCustomPullSecretHasRequiredTokens(&troubleshootCtx), "custom pull secret has required tokens")
	})
}

func TestProxySecret(t *testing.T) {
	t.Run("no proxy secret", func(t *testing.T) {
		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(
				testNewDynakubeBuilder(testNamespace, testDynakube).build(),
				testBuildNamespace(testNamespace),
			).
			Build()

		troubleshootCtx := troubleshootContext{apiReader: clt, namespaceName: testNamespace, dynakubeName: testDynakube}
		assert.NoErrorf(t, checkProxySecret(&troubleshootCtx), "missing dynakube")
		assert.Equal(t, "", troubleshootCtx.proxySecretName)
	})
	t.Run("proxy secret defined", func(t *testing.T) {
		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(
				testNewDynakubeBuilder(testNamespace, testDynakube).withProxySecret(testSecretName).build(),
				testBuildNamespace(testNamespace),
			).
			Build()

		troubleshootCtx := troubleshootContext{apiReader: clt, namespaceName: testNamespace, dynakubeName: testDynakube}
		assert.NoErrorf(t, checkProxySecret(&troubleshootCtx), "missing dynakube")
		assert.Equal(t, testSecretName, troubleshootCtx.proxySecretName)
	})
	t.Run("proxy secret exists", func(t *testing.T) {
		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(
				testNewDynakubeBuilder(testNamespace, testDynakube).withProxySecret(testSecretName).build(),
				testBuildNamespace(testNamespace),
				testNewSecretBuilder(testNamespace, testSecretName).build(),
			).
			Build()

		troubleshootCtx := troubleshootContext{apiReader: clt, namespaceName: testNamespace, dynakubeName: testDynakube, proxySecretName: testSecretName}
		assert.NoErrorf(t, checkIfProxySecretWithTheGivenNameExists(&troubleshootCtx), "proxy secret not found")
	})
	t.Run("proxy secret does not exist", func(t *testing.T) {
		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(
				testNewDynakubeBuilder(testNamespace, testDynakube).withProxySecret(testSecretName).build(),
				testBuildNamespace(testNamespace),
			).
			Build()

		troubleshootCtx := troubleshootContext{apiReader: clt, namespaceName: testNamespace, dynakubeName: testDynakube, proxySecretName: testSecretName}
		assert.Errorf(t, checkIfProxySecretWithTheGivenNameExists(&troubleshootCtx), "proxy secret found")
	})
	t.Run("proxy secret has required tokens", func(t *testing.T) {
		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(
				testNewDynakubeBuilder(testNamespace, testDynakube).withProxySecret(testSecretName).build(),
				testBuildNamespace(testNamespace),
				testNewSecretBuilder(testNamespace, testSecretName).dataAppend(dtclient.CustomProxySecretKey, testCustomPullSecretToken).build(),
			).
			Build()

		troubleshootCtx := troubleshootContext{apiReader: clt, namespaceName: testNamespace, dynakubeName: testDynakube, proxySecretName: testSecretName}
		assert.NoErrorf(t, checkProxySecretHasRequiredTokens(&troubleshootCtx), "proxy secret does not have required tokens")
	})
	t.Run("proxy secret does not have required tokens", func(t *testing.T) {
		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(
				testNewDynakubeBuilder(testNamespace, testDynakube).withProxySecret(testSecretName).build(),
				testBuildNamespace(testNamespace),
				testNewSecretBuilder(testNamespace, testSecretName).build(),
			).
			Build()

		troubleshootCtx := troubleshootContext{apiReader: clt, namespaceName: testNamespace, dynakubeName: testDynakube, proxySecretName: testSecretName}
		assert.Errorf(t, checkProxySecretHasRequiredTokens(&troubleshootCtx), "proxy secret has required tokens")
	})
}

type testDynaKubeBuilder struct {
	dynakube *dynatracev1beta1.DynaKube
}

func testNewDynakubeBuilder(namespace string, dynakube string) *testDynaKubeBuilder {
	return &testDynaKubeBuilder{
		dynakube: &dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      dynakube,
			},
		},
	}
}

func (builder *testDynaKubeBuilder) withApiUrl(apiUrl string) *testDynaKubeBuilder {
	builder.dynakube.Spec.APIURL = apiUrl
	return builder
}

func (builder *testDynaKubeBuilder) withTokens(secretName string) *testDynaKubeBuilder {
	builder.dynakube.Spec.Tokens = secretName
	return builder
}

func (builder *testDynaKubeBuilder) withCustomPullSecret(secretName string) *testDynaKubeBuilder {
	builder.dynakube.Spec.CustomPullSecret = secretName
	return builder
}

func (builder *testDynaKubeBuilder) withProxySecret(secretName string) *testDynaKubeBuilder {
	builder.dynakube.Spec.Proxy = &dynatracev1beta1.DynaKubeProxy{
		ValueFrom: secretName,
	}
	return builder
}

func (builder *testDynaKubeBuilder) withActiveGateImage(image string) *testDynaKubeBuilder {
	builder.dynakube.Spec.ActiveGate.Image = image
	return builder
}

func (builder *testDynaKubeBuilder) withClassicFullStackCustomImage(image string) *testDynaKubeBuilder {
	if builder.dynakube.Spec.OneAgent.ClassicFullStack != nil {
		builder.dynakube.Spec.OneAgent.ClassicFullStack.Image = image
	} else {
		builder.dynakube.Spec.OneAgent.ClassicFullStack = &dynatracev1beta1.HostInjectSpec{
			Image: image,
		}
	}
	return builder
}

func (builder *testDynaKubeBuilder) withCloudNativeFullStackCustomImage(image string) *testDynaKubeBuilder {
	if builder.dynakube.Spec.OneAgent.CloudNativeFullStack != nil {
		builder.dynakube.Spec.OneAgent.CloudNativeFullStack.Image = image
	} else {
		builder.dynakube.Spec.OneAgent.CloudNativeFullStack = &dynatracev1beta1.CloudNativeFullStackSpec{
			HostInjectSpec: dynatracev1beta1.HostInjectSpec{
				Image: image,
			},
		}
	}
	return builder
}

func (builder *testDynaKubeBuilder) withHostMonitoringCustomImage(image string) *testDynaKubeBuilder {
	if builder.dynakube.Spec.OneAgent.HostMonitoring != nil {
		builder.dynakube.Spec.OneAgent.HostMonitoring.Image = image
	} else {
		builder.dynakube.Spec.OneAgent.HostMonitoring = &dynatracev1beta1.HostInjectSpec{
			Image: image,
		}
	}
	return builder
}

func (builder *testDynaKubeBuilder) withClassicFullStackImageVersion(version string) *testDynaKubeBuilder {
	if builder.dynakube.Spec.OneAgent.ClassicFullStack != nil {
		builder.dynakube.Spec.OneAgent.ClassicFullStack.Version = version
	} else {
		builder.dynakube.Spec.OneAgent.ClassicFullStack = &dynatracev1beta1.HostInjectSpec{
			Version: version,
		}
	}
	return builder
}

func (builder *testDynaKubeBuilder) withCloudNativeFullStackImageVersion(version string) *testDynaKubeBuilder {
	if builder.dynakube.Spec.OneAgent.CloudNativeFullStack != nil {
		builder.dynakube.Spec.OneAgent.CloudNativeFullStack.Version = version
	} else {
		builder.dynakube.Spec.OneAgent.CloudNativeFullStack = &dynatracev1beta1.CloudNativeFullStackSpec{
			HostInjectSpec: dynatracev1beta1.HostInjectSpec{
				Version: version,
			},
		}
	}
	return builder
}

func (builder *testDynaKubeBuilder) withHostMonitoringImageVersion(version string) *testDynaKubeBuilder {
	if builder.dynakube.Spec.OneAgent.HostMonitoring != nil {
		builder.dynakube.Spec.OneAgent.HostMonitoring.Version = version
	} else {
		builder.dynakube.Spec.OneAgent.HostMonitoring = &dynatracev1beta1.HostInjectSpec{
			Version: version,
		}
	}
	return builder
}

func (builder *testDynaKubeBuilder) build() *dynatracev1beta1.DynaKube {
	return builder.dynakube
}

type testSecretBuilder struct {
	secret *corev1.Secret
}

func testNewSecretBuilder(namespace string, name string) *testSecretBuilder {
	return &testSecretBuilder{
		secret: &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      name,
			},
		},
	}
}

func (builder *testSecretBuilder) dataAppend(key string, value string) *testSecretBuilder {
	if builder.secret.Data == nil {
		builder.secret.Data = make(map[string][]byte)
		builder.secret.Data[key] = []byte(value)
	} else {
		builder.secret.Data[key] = []byte(value)
	}
	return builder
}

func (builder *testSecretBuilder) build() *corev1.Secret {
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
