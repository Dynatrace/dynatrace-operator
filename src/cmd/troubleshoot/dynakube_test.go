package troubleshoot

import (
	"context"
	"net/http"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects/address"
	"github.com/Dynatrace/dynatrace-operator/src/scheme"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

type errorClient struct {
	client.Client
}

func (errorClt *errorClient) List(_ context.Context, _ client.ObjectList, _ ...client.ListOption) error {
	return errors.New("fake error")
}

func TestDynakubeCRD(t *testing.T) {
	t.Run("crd does not exist", func(t *testing.T) {
		clt := fake.NewClientBuilder().Build()
		troubleshootCtx := troubleshootContext{
			apiReader:     clt,
			namespaceName: testNamespace,
			baseLog:       getNullLogger(t),
		}
		assert.ErrorContains(t, checkCRD(&troubleshootCtx), "CRD for Dynakube missing")
	})
	t.Run("unrelated error", func(t *testing.T) {
		troubleshootCtx := troubleshootContext{
			apiReader:     &errorClient{},
			namespaceName: testNamespace,
			baseLog:       getNullLogger(t),
		}
		assert.ErrorContains(t, checkCRD(&troubleshootCtx), "could not list Dynakube")
	})
}

func TestDynakube(t *testing.T) {
	t.Run("dynakube exists", func(t *testing.T) {
		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(
				testNewDynakubeBuilder(testNamespace, testDynakube).build(),
				testBuildNamespace(testNamespace),
			).
			Build()

		troubleshootCtx := troubleshootContext{
			apiReader:     clt,
			namespaceName: testNamespace,
			dynakube:      *testNewDynakubeBuilder(testNamespace, testDynakube).build(),
		}

		_, err := getSelectedDynakube(&troubleshootCtx)
		assert.NoErrorf(t, err, "no dynakube found")
	})
	t.Run("dynakube does not exist", func(t *testing.T) {
		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(
				testNewDynakubeBuilder(testNamespace, testDynakube).build(),
				testBuildNamespace(testNamespace),
			).
			Build()

		troubleshootCtx := troubleshootContext{
			apiReader:     clt,
			namespaceName: testNamespace,
			dynakube:      *testNewDynakubeBuilder(testNamespace, "doesnotexist").build(),
		}

		_, err := getSelectedDynakube(&troubleshootCtx)
		assert.Errorf(t, err, "dynakube found")
	})
	t.Run("invalid namespace selected", func(t *testing.T) {
		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(
				testNewDynakubeBuilder(testNamespace, testDynakube).build(),
				testBuildNamespace(testNamespace),
			).
			Build()

		troubleshootCtx := troubleshootContext{apiReader: clt, namespaceName: testOtherNamespace}
		_, err := getSelectedDynakube(&troubleshootCtx)
		assert.Errorf(t, err, "dynakube found")
	})
}

func TestApiUrl(t *testing.T) {
	t.Run("valid ApiUrl", func(t *testing.T) {
		troubleshootCtx := troubleshootContext{
			namespaceName: testNamespace,
			dynakube:      *testNewDynakubeBuilder(testNamespace, testDynakube).withApiUrl(testApiUrl).build(),
			baseLog:       getNullLogger(t),
		}
		assert.NoErrorf(t, checkApiUrlSyntax(&troubleshootCtx), "invalid ApiUrl")
	})
	t.Run("invalid ApiUrl", func(t *testing.T) {
		troubleshootCtx := troubleshootContext{
			namespaceName: testNamespace,
			dynakube:      *testNewDynakubeBuilder(testNamespace, testDynakube).withApiUrl(testOtherApiUrl).build(),
			baseLog:       getNullLogger(t),
		}
		assert.Errorf(t, checkApiUrlSyntax(&troubleshootCtx), "valid ApiUrl")
	})
}

func TestDynatraceSecret(t *testing.T) {
	t.Run("Dynatrace secret exists", func(t *testing.T) {
		dynakube := testNewDynakubeBuilder(testNamespace, testDynakube).withTokens(testDynatraceSecret).build()
		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(
				dynakube,
				testBuildNamespace(testNamespace),
				testNewSecretBuilder(testNamespace, testDynatraceSecret).build(),
			).
			Build()

		troubleshootCtx := troubleshootContext{
			apiReader:     clt,
			namespaceName: testNamespace,
			dynakube:      *dynakube,
		}
		_, err := getSelectedDynakube(&troubleshootCtx)
		assert.NoErrorf(t, err, "Dynatrace secret not found")
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

		troubleshootCtx := troubleshootContext{
			apiReader:     clt,
			namespaceName: testNamespace,
			dynakube:      *testNewDynakubeBuilder(testNamespace, testDynakube).build(),
			baseLog:       getNullLogger(t),
		}
		assert.Errorf(t, checkIfDynatraceApiSecretHasApiToken(&troubleshootCtx), "Dynatrace secret found")
	})

	t.Run("Dynatrace secret has apiToken token", func(t *testing.T) {
		dynakube := testNewDynakubeBuilder(testNamespace, testDynakube).withTokens(testDynatraceSecret).build()
		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(
				dynakube,
				testBuildNamespace(testNamespace),
				testNewSecretBuilder(testNamespace, testDynatraceSecret).dataAppend("apiToken", testApiToken).dataAppend("paasToken", testPaasToken).build(),
			).
			Build()

		troubleshootCtx := troubleshootContext{
			apiReader:     clt,
			namespaceName: testNamespace,
			dynakube:      *dynakube,
			baseLog:       getNullLogger(t),
		}
		assert.NoErrorf(t, checkIfDynatraceApiSecretHasApiToken(&troubleshootCtx), "Dynatrace secret does not have required tokens")
	})
	t.Run("Dynatrace secret - apiToken is missing", func(t *testing.T) {
		dynakube := testNewDynakubeBuilder(testNamespace, testDynakube).withTokens(testDynatraceSecret).build()
		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(
				dynakube,
				testBuildNamespace(testNamespace),
				testNewSecretBuilder(testNamespace, testDynatraceSecret).dataAppend("paasToken", testPaasToken).build(),
			).
			Build()

		troubleshootCtx := troubleshootContext{
			apiReader:     clt,
			namespaceName: testNamespace,
			dynakube:      *dynakube,
			baseLog:       getNullLogger(t),
		}
		assert.Errorf(t, checkIfDynatraceApiSecretHasApiToken(&troubleshootCtx), "Dynatrace secret does not have apiToken")
	})
}

func TestPullSecret(t *testing.T) {
	t.Run("custom pull secret exists", func(t *testing.T) {
		dynakube := testNewDynakubeBuilder(testNamespace, testDynakube).withCustomPullSecret(testSecretName).build()
		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(
				dynakube,
				testBuildNamespace(testNamespace),
				testNewSecretBuilder(testNamespace, testSecretName).build(),
			).
			Build()

		troubleshootCtx := troubleshootContext{
			apiReader:     clt,
			namespaceName: testNamespace,
			dynakube:      *dynakube,
			baseLog:       getNullLogger(t),
		}
		assert.NoErrorf(t, checkPullSecretExists(&troubleshootCtx), "custom pull secret not found")
	})
	t.Run("custom pull secret does not exist", func(t *testing.T) {
		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(
				testNewDynakubeBuilder(testNamespace, testDynakube).withCustomPullSecret(testSecretName).build(),
				testBuildNamespace(testNamespace),
			).
			Build()

		troubleshootCtx := troubleshootContext{
			apiReader:     clt,
			namespaceName: testNamespace,
			baseLog:       getNullLogger(t),
		}
		assert.Errorf(t, checkPullSecretExists(&troubleshootCtx), "custom pull secret found")
	})
	t.Run("custom pull secret has required tokens", func(t *testing.T) {
		troubleshootCtx := troubleshootContext{
			namespaceName: testNamespace,
			pullSecret:    *testNewSecretBuilder(testNamespace, testSecretName).dataAppend(".dockerconfigjson", testCustomPullSecretToken).build(),
			baseLog:       getNullLogger(t),
		}
		assert.NoErrorf(t, checkPullSecretHasRequiredTokens(&troubleshootCtx), "custom pull secret does not have required tokens")
	})
	t.Run("custom pull secret does not have required tokens", func(t *testing.T) {
		troubleshootCtx := troubleshootContext{
			namespaceName: testNamespace,
			pullSecret:    *testNewSecretBuilder(testNamespace, testSecretName).build(),
			baseLog:       getNullLogger(t),
		}
		assert.Errorf(t, checkPullSecretHasRequiredTokens(&troubleshootCtx), "custom pull secret has required tokens")
	})
}

func TestProxySecret(t *testing.T) {
	t.Run("proxy secret exists", func(t *testing.T) {
		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(
				testNewDynakubeBuilder(testNamespace, testDynakube).withProxySecret(testSecretName).build(),
				testBuildNamespace(testNamespace),
				testNewSecretBuilder(testNamespace, testSecretName).build(),
			).
			Build()

		troubleshootCtx := troubleshootContext{apiReader: clt, namespaceName: testNamespace}
		assert.NoErrorf(t, applyProxySettings(getNullLogger(t), &troubleshootCtx), "proxy secret not found")
	})
	t.Run("proxy secret does not exist", func(t *testing.T) {
		dynakube := testNewDynakubeBuilder(testNamespace, testDynakube).withProxySecret(testSecretName).build()
		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(
				dynakube,
				testBuildNamespace(testNamespace),
			).
			Build()

		troubleshootCtx := troubleshootContext{
			apiReader: clt, namespaceName: testNamespace, dynakube: *dynakube,
		}
		assert.Errorf(t, applyProxySettings(getNullLogger(t), &troubleshootCtx), "proxy secret found, should not exist")
	})
	t.Run("proxy secret has required tokens", func(t *testing.T) {
		proxySecret := *testNewSecretBuilder(testNamespace, testSecretName).
			dataAppend(dynatracev1beta1.ProxyKey, testCustomPullSecretToken).
			build()
		dynakube := *testNewDynakubeBuilder(testNamespace, testDynakube).withProxySecret(testSecretName).build()
		clt := fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(
			&dynakube,
			&proxySecret).
			Build()
		troubleshootCtx := troubleshootContext{
			namespaceName: testNamespace,
			httpClient:    &http.Client{},
			apiReader:     clt,
		}
		assert.NoErrorf(t, applyProxySettings(getNullLogger(t), &troubleshootCtx), "proxy secret does not have required tokens")
	})
	t.Run("proxy secret does not have required tokens", func(t *testing.T) {
		secret := *testNewSecretBuilder(testNamespace, testSecretName).build()
		dynakube := *testNewDynakubeBuilder(testNamespace, testDynakube).withProxySecret(testSecretName).build()
		clt := fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(
			&dynakube,
			&secret).
			Build()
		troubleshootCtx := troubleshootContext{
			namespaceName: testNamespace,
			dynakube:      dynakube,
			apiReader:     clt}
		assert.Errorf(t, applyProxySettings(getNullLogger(t), &troubleshootCtx), "proxy secret has required tokens")
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

func (builder *testDynaKubeBuilder) withProxy(proxyURL string) *testDynaKubeBuilder {
	builder.dynakube.Spec.Proxy = &dynatracev1beta1.DynaKubeProxy{
		Value: proxyURL,
	}
	return builder
}

func (builder *testDynaKubeBuilder) withProxySecret(secretName string) *testDynaKubeBuilder {
	builder.dynakube.Spec.Proxy = &dynatracev1beta1.DynaKubeProxy{
		ValueFrom: secretName,
	}
	return builder
}

func (builder *testDynaKubeBuilder) withActiveGateCapability(capability dynatracev1beta1.CapabilityDisplayName) *testDynaKubeBuilder {
	if builder.dynakube.Spec.ActiveGate.Capabilities == nil {
		builder.dynakube.Spec.ActiveGate.Capabilities = make([]dynatracev1beta1.CapabilityDisplayName, 0)
	}

	builder.dynakube.Spec.ActiveGate.Capabilities = append(builder.dynakube.Spec.ActiveGate.Capabilities, capability)
	return builder
}

func (builder *testDynaKubeBuilder) withActiveGateCustomImage(image string) *testDynaKubeBuilder {
	builder.dynakube.Spec.ActiveGate.Image = image
	return builder
}

func (builder *testDynaKubeBuilder) withCloudNativeFullStack() *testDynaKubeBuilder {
	builder.dynakube.Spec.OneAgent.CloudNativeFullStack = &dynatracev1beta1.CloudNativeFullStackSpec{
		HostInjectSpec: dynatracev1beta1.HostInjectSpec{},
	}
	return builder
}

func (builder *testDynaKubeBuilder) withClassicFullStack() *testDynaKubeBuilder {
	builder.dynakube.Spec.OneAgent.ClassicFullStack = &dynatracev1beta1.HostInjectSpec{}
	return builder
}

func (builder *testDynaKubeBuilder) withHostMonitoring() *testDynaKubeBuilder {
	builder.dynakube.Spec.OneAgent.HostMonitoring = &dynatracev1beta1.HostInjectSpec{}
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

func (builder *testDynaKubeBuilder) withCloudNativeCodeModulesImage(image string) *testDynaKubeBuilder {
	if builder.dynakube.Spec.OneAgent.CloudNativeFullStack != nil {
		builder.dynakube.Spec.OneAgent.CloudNativeFullStack.CodeModulesImage = image
	} else {
		builder.dynakube.Spec.OneAgent.CloudNativeFullStack = &dynatracev1beta1.CloudNativeFullStackSpec{
			AppInjectionSpec: dynatracev1beta1.AppInjectionSpec{
				InitResources:    &corev1.ResourceRequirements{},
				CodeModulesImage: image,
			},
		}
	}
	return builder
}

func (builder *testDynaKubeBuilder) withApplicationMonitoringCodeModulesImage(image string) *testDynaKubeBuilder {
	if builder.dynakube.Spec.OneAgent.ApplicationMonitoring != nil {
		builder.dynakube.Spec.OneAgent.ApplicationMonitoring.CodeModulesImage = image
		builder.dynakube.Spec.OneAgent.ApplicationMonitoring.UseCSIDriver = address.Of(true)
	} else {
		builder.dynakube.Spec.OneAgent.ApplicationMonitoring = &dynatracev1beta1.ApplicationMonitoringSpec{
			AppInjectionSpec: dynatracev1beta1.AppInjectionSpec{
				InitResources:    &corev1.ResourceRequirements{},
				CodeModulesImage: image,
			},
			UseCSIDriver: address.Of(true),
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
