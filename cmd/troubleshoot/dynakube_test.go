package troubleshoot

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/address"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
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

func TestDynakubeCRD(t *testing.T) {
	t.Run("crd does not exist", func(t *testing.T) {
		err := runtime.NewNotRegisteredErrForKind("dynakube", schema.GroupVersionKind{})
		assert.ErrorContains(t, checkCRD(getNullLogger(t), err), "CRD for Dynakube missing")
	})
	t.Run("unrelated error", func(t *testing.T) {
		err := errors.New("fake error")
		assert.ErrorContains(t, checkCRD(getNullLogger(t), err), "could not list Dynakube")
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

		_, err := getSelectedDynakube(context.Background(), clt, testNamespace, testDynakube)
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

		_, err := getSelectedDynakube(context.Background(), clt, testNamespace, "doesnotexist")
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

		_, err := getSelectedDynakube(context.Background(), clt, testOtherNamespace, testDynakube)
		assert.Errorf(t, err, "dynakube found")
	})
}

func TestApiUrl(t *testing.T) {
	t.Run("valid ApiUrl", func(t *testing.T) {
		assert.NoErrorf(t, checkApiUrlSyntax(context.Background(), getNullLogger(t), testNewDynakubeBuilder(testNamespace, testDynakube).withApiUrl(testApiUrl).build()), "invalid ApiUrl")
	})
	t.Run("invalid ApiUrl", func(t *testing.T) {
		assert.Errorf(t, checkApiUrlSyntax(context.Background(), getNullLogger(t), testNewDynakubeBuilder(testNamespace, testDynakube).withApiUrl(testOtherApiUrl).build()), "valid ApiUrl")
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

		_, err := getSelectedDynakube(context.Background(), clt, testNamespace, testDynakube)
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

		dynakube := testNewDynakubeBuilder(testNamespace, testDynakube).build()
		_, err := checkIfDynatraceApiSecretHasApiToken(context.Background(), getNullLogger(t), clt, dynakube)
		assert.Errorf(t, err, "Dynatrace secret found")
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

		_, err := checkIfDynatraceApiSecretHasApiToken(context.Background(), getNullLogger(t), clt, dynakube)
		assert.NoErrorf(t, err, "Dynatrace secret does not have required tokens")
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

		_, err := checkIfDynatraceApiSecretHasApiToken(context.Background(), getNullLogger(t), clt, dynakube)
		assert.Errorf(t, err, "Dynatrace secret does not have apiToken")
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

		_, err := checkPullSecretExists(context.Background(), getNullLogger(t), clt, dynakube)
		assert.NoErrorf(t, err, "custom pull secret not found")
	})
	t.Run("custom pull secret does not exist", func(t *testing.T) {
		dynakube := testNewDynakubeBuilder(testNamespace, testDynakube).withCustomPullSecret(testSecretName).build()
		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(
				dynakube,
				testBuildNamespace(testNamespace),
			).
			Build()

		_, err := checkPullSecretExists(context.Background(), getNullLogger(t), clt, dynakube)
		assert.Errorf(t, err, "custom pull secret found")
	})
	t.Run("custom pull secret has required tokens", func(t *testing.T) {
		assert.NoErrorf(t, checkPullSecretHasRequiredTokens(getNullLogger(t), nil, *testNewSecretBuilder(testNamespace, testSecretName).dataAppend(".dockerconfigjson", testCustomPullSecretToken).build()), "custom pull secret does not have required tokens")
	})
	t.Run("custom pull secret does not have required tokens", func(t *testing.T) {
		assert.Errorf(t, checkPullSecretHasRequiredTokens(getNullLogger(t), &dynatracev1beta1.DynaKube{}, *testNewSecretBuilder(testNamespace, testSecretName).build()), "custom pull secret has required tokens")
	})
}

func TestProxySecret(t *testing.T) {
	t.Run("proxy secret exists", func(t *testing.T) {
		dynakube := testNewDynakubeBuilder(testNamespace, testDynakube).withProxySecret(dynatracev1beta1.ProxyKey).build()
		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(
				dynakube,
				testBuildNamespace(testNamespace),
				testNewSecretBuilder(testNamespace, dynatracev1beta1.ProxyKey).build(),
			).
			Build()

		_, err := getProxyURL(context.Background(), clt, dynakube)
		assert.NoErrorf(t, err, "proxy secret not found")
	})
	t.Run("proxy secret does not exist", func(t *testing.T) {
		dynakube := testNewDynakubeBuilder(testNamespace, testDynakube).withProxySecret(dynatracev1beta1.ProxyKey).build()
		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(
				dynakube,
				testBuildNamespace(testNamespace),
			).
			Build()

		_, err := getProxyURL(context.Background(), clt, dynakube)
		assert.Errorf(t, err, "proxy secret found, should not exist")
	})
	t.Run("proxy secret has required tokens", func(t *testing.T) {
		proxySecret := *testNewSecretBuilder(testNamespace, dynatracev1beta1.ProxyKey).
			dataAppend(dynatracev1beta1.ProxyKey, testCustomPullSecretToken).
			build()
		dynakube := testNewDynakubeBuilder(testNamespace, testDynakube).withProxySecret(dynatracev1beta1.ProxyKey).build()
		clt := fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(
			dynakube,
			&proxySecret).
			Build()
		_, err := getProxyURL(context.Background(), clt, dynakube)
		assert.NoErrorf(t, err, "proxy secret does not have required tokens")
	})
	t.Run("proxy secret does not have required tokens", func(t *testing.T) {
		secret := *testNewSecretBuilder(testNamespace, testSecretName).build()
		dynakube := testNewDynakubeBuilder(testNamespace, testDynakube).withProxySecret(dynatracev1beta1.ProxyKey).build()
		clt := fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(
			dynakube,
			&secret).
			Build()
		_, err := getProxyURL(context.Background(), clt, dynakube)
		assert.Errorf(t, err, "proxy secret has required tokens")
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
	secretData := map[string][]byte{
		name: []byte("topsecretstringhere"),
	}
	return &testSecretBuilder{
		secret: &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      name,
			},
			Data: secretData,
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
