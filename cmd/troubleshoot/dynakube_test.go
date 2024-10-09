package troubleshoot

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/value"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube/activegate"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
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
		require.ErrorContains(t, checkCRD(getNullLogger(t), err), "CRD for Dynakube missing")
	})
	t.Run("unrelated error", func(t *testing.T) {
		err := errors.New("fake error")
		require.ErrorContains(t, checkCRD(getNullLogger(t), err), "could not list Dynakube")
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
		require.NoErrorf(t, err, "no dynakube found")
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
		require.Errorf(t, err, "dynakube found")
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
		require.Errorf(t, err, "dynakube found")
	})
}

func TestDynatraceSecret(t *testing.T) {
	t.Run("Dynatrace secret exists", func(t *testing.T) {
		dk := testNewDynakubeBuilder(testNamespace, testDynakube).withTokens(testDynatraceSecret).build()
		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(
				dk,
				testBuildNamespace(testNamespace),
				testNewSecretBuilder(testNamespace, testDynatraceSecret).build(),
			).
			Build()

		_, err := getSelectedDynakube(context.Background(), clt, testNamespace, testDynakube)
		require.NoErrorf(t, err, "Dynatrace secret not found")
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

		dk := testNewDynakubeBuilder(testNamespace, testDynakube).build()
		_, err := checkIfDynatraceApiSecretHasApiToken(context.Background(), getNullLogger(t), clt, dk)
		require.Errorf(t, err, "Dynatrace secret found")
	})

	t.Run("Dynatrace secret has apiToken token", func(t *testing.T) {
		dk := testNewDynakubeBuilder(testNamespace, testDynakube).withTokens(testDynatraceSecret).build()
		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(
				dk,
				testBuildNamespace(testNamespace),
				testNewSecretBuilder(testNamespace, testDynatraceSecret).dataAppend("apiToken", testApiToken).dataAppend("paasToken", testPaasToken).build(),
			).
			Build()

		_, err := checkIfDynatraceApiSecretHasApiToken(context.Background(), getNullLogger(t), clt, dk)
		require.NoErrorf(t, err, "Dynatrace secret does not have required tokens")
	})
	t.Run("Dynatrace secret - apiToken is missing", func(t *testing.T) {
		dk := testNewDynakubeBuilder(testNamespace, testDynakube).withTokens(testDynatraceSecret).build()
		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(
				dk,
				testBuildNamespace(testNamespace),
				testNewSecretBuilder(testNamespace, testDynatraceSecret).dataAppend("paasToken", testPaasToken).build(),
			).
			Build()

		_, err := checkIfDynatraceApiSecretHasApiToken(context.Background(), getNullLogger(t), clt, dk)
		require.Errorf(t, err, "Dynatrace secret does not have apiToken")
	})
}

func TestPullSecret(t *testing.T) {
	t.Run("custom pull secret exists", func(t *testing.T) {
		dk := testNewDynakubeBuilder(testNamespace, testDynakube).withCustomPullSecret(testSecretName).build()
		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(
				dk,
				testBuildNamespace(testNamespace),
				testNewSecretBuilder(testNamespace, testSecretName).build(),
			).
			Build()

		_, err := checkPullSecretExists(context.Background(), getNullLogger(t), clt, dk)
		require.NoErrorf(t, err, "custom pull secret not found")
	})
	t.Run("custom pull secret does not exist", func(t *testing.T) {
		dk := testNewDynakubeBuilder(testNamespace, testDynakube).withCustomPullSecret(testSecretName).build()
		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(
				dk,
				testBuildNamespace(testNamespace),
			).
			Build()

		_, err := checkPullSecretExists(context.Background(), getNullLogger(t), clt, dk)
		require.Errorf(t, err, "custom pull secret found")
	})
	t.Run("custom pull secret has required tokens", func(t *testing.T) {
		require.NoErrorf(t, checkPullSecretHasRequiredTokens(getNullLogger(t), nil, *testNewSecretBuilder(testNamespace, testSecretName).dataAppend(".dockerconfigjson", testCustomPullSecretToken).build()), "custom pull secret does not have required tokens")
	})
	t.Run("custom pull secret does not have required tokens", func(t *testing.T) {
		require.Errorf(t, checkPullSecretHasRequiredTokens(getNullLogger(t), &dynakube.DynaKube{}, *testNewSecretBuilder(testNamespace, testSecretName).build()), "custom pull secret has required tokens")
	})
}

func TestProxySecret(t *testing.T) {
	t.Run("proxy secret exists", func(t *testing.T) {
		dk := testNewDynakubeBuilder(testNamespace, testDynakube).withProxySecret(dynakube.ProxyKey).build()
		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(
				dk,
				testBuildNamespace(testNamespace),
				testNewSecretBuilder(testNamespace, dynakube.ProxyKey).build(),
			).
			Build()

		_, err := getProxyURL(context.Background(), clt, dk)
		require.NoErrorf(t, err, "proxy secret not found")
	})
	t.Run("proxy secret does not exist", func(t *testing.T) {
		dk := testNewDynakubeBuilder(testNamespace, testDynakube).withProxySecret(dynakube.ProxyKey).build()
		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(
				dk,
				testBuildNamespace(testNamespace),
			).
			Build()

		_, err := getProxyURL(context.Background(), clt, dk)
		require.Errorf(t, err, "proxy secret found, should not exist")
	})
	t.Run("proxy secret has required tokens", func(t *testing.T) {
		proxySecret := *testNewSecretBuilder(testNamespace, dynakube.ProxyKey).
			dataAppend(dynakube.ProxyKey, testCustomPullSecretToken).
			build()
		dk := testNewDynakubeBuilder(testNamespace, testDynakube).withProxySecret(dynakube.ProxyKey).build()
		clt := fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(
			dk,
			&proxySecret).
			Build()
		_, err := getProxyURL(context.Background(), clt, dk)
		require.NoErrorf(t, err, "proxy secret does not have required tokens")
	})
	t.Run("proxy secret does not have required tokens", func(t *testing.T) {
		secret := *testNewSecretBuilder(testNamespace, testSecretName).build()
		dk := testNewDynakubeBuilder(testNamespace, testDynakube).withProxySecret(dynakube.ProxyKey).build()
		clt := fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(
			dk,
			&secret).
			Build()
		_, err := getProxyURL(context.Background(), clt, dk)
		require.Errorf(t, err, "proxy secret has required tokens")
	})
}

type testDynaKubeBuilder struct {
	dynakube *dynakube.DynaKube
}

func testNewDynakubeBuilder(namespace string, dynakubeName string) *testDynaKubeBuilder {
	return &testDynaKubeBuilder{
		dynakube: &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      dynakubeName,
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
	builder.dynakube.Spec.Proxy = &value.Source{
		Value: proxyURL,
	}

	return builder
}

func (builder *testDynaKubeBuilder) withProxySecret(secretName string) *testDynaKubeBuilder {
	builder.dynakube.Spec.Proxy = &value.Source{
		ValueFrom: secretName,
	}

	return builder
}

func (builder *testDynaKubeBuilder) withActiveGateCapability(capability activegate.CapabilityDisplayName) *testDynaKubeBuilder {
	if builder.dynakube.Spec.ActiveGate.Capabilities == nil {
		builder.dynakube.Spec.ActiveGate.Capabilities = make([]activegate.CapabilityDisplayName, 0)
	}

	builder.dynakube.Spec.ActiveGate.Capabilities = append(builder.dynakube.Spec.ActiveGate.Capabilities, capability)
	builder.dynakube.Status.ActiveGate.ImageID = builder.dynakube.ActiveGate().GetDefaultImage(testVersion)

	return builder
}

func (builder *testDynaKubeBuilder) withActiveGateCustomImage(image string) *testDynaKubeBuilder {
	builder.dynakube.Spec.ActiveGate.Image = image
	builder.dynakube.Status.ActiveGate.ImageID = image

	return builder
}

func (builder *testDynaKubeBuilder) withCloudNativeFullStack() *testDynaKubeBuilder {
	builder.dynakube.Spec.OneAgent.CloudNativeFullStack = &dynakube.CloudNativeFullStackSpec{
		HostInjectSpec: dynakube.HostInjectSpec{},
	}
	builder.dynakube.Status.OneAgent.ImageID = builder.dynakube.DefaultOneAgentImage(testVersion)

	return builder
}

func (builder *testDynaKubeBuilder) withClassicFullStack() *testDynaKubeBuilder {
	builder.dynakube.Spec.OneAgent.ClassicFullStack = &dynakube.HostInjectSpec{}
	builder.dynakube.Status.OneAgent.ImageID = builder.dynakube.DefaultOneAgentImage(testVersion)

	return builder
}

func (builder *testDynaKubeBuilder) withHostMonitoring() *testDynaKubeBuilder {
	builder.dynakube.Spec.OneAgent.HostMonitoring = &dynakube.HostInjectSpec{}
	builder.dynakube.Status.OneAgent.ImageID = builder.dynakube.DefaultOneAgentImage(testVersion)

	return builder
}

func (builder *testDynaKubeBuilder) withClassicFullStackCustomImage(image string) *testDynaKubeBuilder {
	if builder.dynakube.Spec.OneAgent.ClassicFullStack != nil {
		builder.dynakube.Spec.OneAgent.ClassicFullStack.Image = image
	} else {
		builder.dynakube.Spec.OneAgent.ClassicFullStack = &dynakube.HostInjectSpec{
			Image: image,
		}
	}
	builder.dynakube.Status.OneAgent.ImageID = image

	return builder
}

func (builder *testDynaKubeBuilder) withCloudNativeFullStackCustomImage(image string) *testDynaKubeBuilder {
	if builder.dynakube.Spec.OneAgent.CloudNativeFullStack != nil {
		builder.dynakube.Spec.OneAgent.CloudNativeFullStack.Image = image
	} else {
		builder.dynakube.Spec.OneAgent.CloudNativeFullStack = &dynakube.CloudNativeFullStackSpec{
			HostInjectSpec: dynakube.HostInjectSpec{
				Image: image,
			},
		}
	}
	builder.dynakube.Status.OneAgent.ImageID = image

	return builder
}

func (builder *testDynaKubeBuilder) withHostMonitoringCustomImage(image string) *testDynaKubeBuilder {
	if builder.dynakube.Spec.OneAgent.HostMonitoring != nil {
		builder.dynakube.Spec.OneAgent.HostMonitoring.Image = image
	} else {
		builder.dynakube.Spec.OneAgent.HostMonitoring = &dynakube.HostInjectSpec{
			Image: image,
		}
	}
	builder.dynakube.Status.OneAgent.ImageID = image

	return builder
}

func (builder *testDynaKubeBuilder) withCloudNativeCodeModulesImage(image string) *testDynaKubeBuilder {
	if builder.dynakube.Spec.OneAgent.CloudNativeFullStack != nil {
		builder.dynakube.Spec.OneAgent.CloudNativeFullStack.CodeModulesImage = image
	} else {
		builder.dynakube.Spec.OneAgent.CloudNativeFullStack = &dynakube.CloudNativeFullStackSpec{
			AppInjectionSpec: dynakube.AppInjectionSpec{
				InitResources:    &corev1.ResourceRequirements{},
				CodeModulesImage: image,
			},
		}
	}
	builder.dynakube.Status.CodeModules.ImageID = image

	return builder
}

func (builder *testDynaKubeBuilder) withApplicationMonitoringCodeModulesImage(image string) *testDynaKubeBuilder {
	if builder.dynakube.Spec.OneAgent.ApplicationMonitoring != nil {
		builder.dynakube.Spec.OneAgent.ApplicationMonitoring.CodeModulesImage = image
		builder.dynakube.Spec.OneAgent.ApplicationMonitoring.UseCSIDriver = true
	} else {
		builder.dynakube.Spec.OneAgent.ApplicationMonitoring = &dynakube.ApplicationMonitoringSpec{
			AppInjectionSpec: dynakube.AppInjectionSpec{
				InitResources:    &corev1.ResourceRequirements{},
				CodeModulesImage: image,
			},
			UseCSIDriver: true,
		}
	}
	builder.dynakube.Status.CodeModules.ImageID = image

	return builder
}

func (builder *testDynaKubeBuilder) build() *dynakube.DynaKube {
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
