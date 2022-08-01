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
	t.Run("dynakube exists", func(t *testing.T) {
		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(
				testNewDynakubeBuilder(testNamespace, testDynakube).build(),
				testBuildNamespace(testNamespace),
			).
			Build()

		troubleshootCtx := troubleshootContext{apiReader: clt, namespaceName: testNamespace, dynakubeName: testDynakube}
		assert.NoErrorf(t, getSelectedDynakubeIfItExists(&troubleshootCtx), "no dynakube found")
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
		assert.Errorf(t, getSelectedDynakubeIfItExists(&troubleshootCtx), "dynakube found")
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
		assert.Errorf(t, getSelectedDynakubeIfItExists(&troubleshootCtx), "dynakube found")
	})
}

func TestApiUrl(t *testing.T) {
	t.Run("valid ApiUrl", func(t *testing.T) {
		troubleshootCtx := troubleshootContext{namespaceName: testNamespace, dynakube: *testNewDynakubeBuilder(testNamespace, testDynakube).withApiUrl(testApiUrl).build()}
		assert.NoErrorf(t, checkApiUrl(&troubleshootCtx), "invalid ApiUrl")
	})
	t.Run("invalid ApiUrl", func(t *testing.T) {

		troubleshootCtx := troubleshootContext{namespaceName: testNamespace, dynakube: *testNewDynakubeBuilder(testNamespace, testDynakube).withApiUrl(testOtherApiUrl).build()}
		assert.Errorf(t, checkApiUrl(&troubleshootCtx), "valid ApiUrl")
	})
}

func TestDynatraceSecret(t *testing.T) {
	t.Run("default name of Dynatrace secret", func(t *testing.T) {
		troubleshootCtx := troubleshootContext{namespaceName: testNamespace, dynakube: *testNewDynakubeBuilder(testNamespace, testDynakube).withTokens(testDynatraceSecret).build()}
		assert.NoErrorf(t, evaluateDynatraceApiSecretName(&troubleshootCtx), "missing dynakube")
		assert.Equal(t, testDynakube, troubleshootCtx.dynatraceApiSecretName)
	})
	t.Run("custom name of Dynatrace secret", func(t *testing.T) {
		troubleshootCtx := troubleshootContext{namespaceName: testNamespace, dynakube: *testNewDynakubeBuilder(testNamespace, testDynakube).withTokens(testOtherDynatraceSecret).build()}
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
		assert.NoErrorf(t, getSelectedDynakubeIfItExists(&troubleshootCtx), "Dynatrace secret not found")
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
		assert.Errorf(t, getDynatraceApiSecretIfItExists(&troubleshootCtx), "Dynatrace secret found")
	})

	t.Run("Dynatrace secret has apiToken token", func(t *testing.T) {
		troubleshootCtx := troubleshootContext{namespaceName: testNamespace, dynatraceApiSecret: *testNewSecretBuilder(testNamespace, testDynatraceSecret).dataAppend("apiToken", testApiToken).dataAppend("paasToken", testPaasToken).build()}
		assert.NoErrorf(t, checkIfDynatraceApiSecretHasApiToken(&troubleshootCtx), "Dynatrace secret does not have required tokens")
	})
	t.Run("Dynatrace secret - apiToken is missing", func(t *testing.T) {
		troubleshootCtx := troubleshootContext{namespaceName: testNamespace, dynatraceApiSecret: *testNewSecretBuilder(testNamespace, testDynatraceSecret).dataAppend("paasToken", testPaasToken).build()}
		assert.Errorf(t, checkIfDynatraceApiSecretHasApiToken(&troubleshootCtx), "Dynatrace secret does not have apiToken")
	})
}

func TestPullSecret(t *testing.T) {
	t.Run("no custom pull secret", func(t *testing.T) {
		troubleshootCtx := troubleshootContext{namespaceName: testNamespace, dynakubeName: testDynakube, dynakube: *testNewDynakubeBuilder(testNamespace, testDynakube).build()}
		_ = evaluatePullSecret(&troubleshootCtx)
		assert.Equal(t, testDynakube+pullSecretSuffix, troubleshootCtx.pullSecretName)
	})
	t.Run("custom pull secret defined", func(t *testing.T) {
		troubleshootCtx := troubleshootContext{namespaceName: testNamespace, dynakubeName: testDynakube, dynakube: *testNewDynakubeBuilder(testNamespace, testDynakube).withCustomPullSecret(testSecretName).build()}
		_ = evaluatePullSecret(&troubleshootCtx)
		assert.Equal(t, testSecretName, troubleshootCtx.pullSecretName)
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

		troubleshootCtx := troubleshootContext{apiReader: clt, namespaceName: testNamespace, pullSecretName: testSecretName}
		assert.NoErrorf(t, getPullSecretIfItExists(&troubleshootCtx), "custom pull secret not found")
	})
	t.Run("custom pull secret does not exist", func(t *testing.T) {
		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(
				testNewDynakubeBuilder(testNamespace, testDynakube).withCustomPullSecret(testSecretName).build(),
				testBuildNamespace(testNamespace),
			).
			Build()

		troubleshootCtx := troubleshootContext{apiReader: clt, namespaceName: testNamespace, pullSecretName: testSecretName}
		assert.Errorf(t, getPullSecretIfItExists(&troubleshootCtx), "custom pull secret found")
	})
	t.Run("custom pull secret has required tokens", func(t *testing.T) {
		troubleshootCtx := troubleshootContext{namespaceName: testNamespace, pullSecretName: testSecretName, pullSecret: *testNewSecretBuilder(testNamespace, testSecretName).dataAppend(".dockerconfigjson", testCustomPullSecretToken).build()}
		assert.NoErrorf(t, checkPullSecretHasRequiredTokens(&troubleshootCtx), "custom pull secret does not have required tokens")
	})
	t.Run("custom pull secret does not have required tokens", func(t *testing.T) {
		troubleshootCtx := troubleshootContext{namespaceName: testNamespace, pullSecretName: testSecretName, pullSecret: *testNewSecretBuilder(testNamespace, testSecretName).build()}
		assert.Errorf(t, checkPullSecretHasRequiredTokens(&troubleshootCtx), "custom pull secret has required tokens")
	})
}

func TestProxySecret(t *testing.T) {
	t.Run("no proxy secret", func(t *testing.T) {
		troubleshootCtx := troubleshootContext{namespaceName: testNamespace, dynakube: *testNewDynakubeBuilder(testNamespace, testDynakube).build()}
		_ = evaluateProxySecret(&troubleshootCtx)
		assert.Equal(t, "", troubleshootCtx.proxySecretName)
	})
	t.Run("proxy secret defined", func(t *testing.T) {
		troubleshootCtx := troubleshootContext{namespaceName: testNamespace, dynakube: *testNewDynakubeBuilder(testNamespace, testDynakube).withProxySecret(testSecretName).build()}
		_ = evaluateProxySecret(&troubleshootCtx)
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
		assert.NoErrorf(t, getProxySecretIfItExists(&troubleshootCtx), "proxy secret not found")
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
		assert.Errorf(t, getProxySecretIfItExists(&troubleshootCtx), "proxy secret found")
	})
	t.Run("proxy secret has required tokens", func(t *testing.T) {
		troubleshootCtx := troubleshootContext{namespaceName: testNamespace, proxySecretName: testSecretName, proxySecret: *testNewSecretBuilder(testNamespace, testSecretName).dataAppend(dtclient.CustomProxySecretKey, testCustomPullSecretToken).build()}
		assert.NoErrorf(t, checkProxySecretHasRequiredTokens(&troubleshootCtx), "proxy secret does not have required tokens")
	})
	t.Run("proxy secret does not have required tokens", func(t *testing.T) {
		troubleshootCtx := troubleshootContext{namespaceName: testNamespace, proxySecretName: testSecretName, proxySecret: *testNewSecretBuilder(testNamespace, testSecretName).build()}
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
