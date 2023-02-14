package activegate

import (
	"context"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/synthetic/autoscaler"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects/address"
	"github.com/Dynatrace/dynatrace-operator/src/scheme"
	"github.com/Dynatrace/dynatrace-operator/src/scheme/fake"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	scalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	testName        = "test-name"
	testNamespace   = "test-namespace"
	testProxyName   = "test-proxy"
	testServiceName = testName + "-activegate"
)

var (
	testKubeSystemNamespace = &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "kube-system",
			UID:  "01234-5678-9012-3456",
		},
	}

	syntheticCapabilityObjectMeta = metav1.ObjectMeta{
		Namespace: testNamespace,
		Name:      testName,
		Annotations: map[string]string{
			dynatracev1beta1.AnnotationFeatureSyntheticLocationEntityId: "imaginary",
			dynatracev1beta1.AnnotationFeatureSyntheticNodeType:         dynatracev1beta1.SyntheticNodeXs,
		},
	}
)

func TestReconciler_Reconcile(t *testing.T) {
	dtc := &dtclient.MockDynatraceClient{}
	dtc.On("GetActiveGateAuthToken", testName).Return(&dtclient.ActiveGateAuthTokenInfo{}, nil)

	t.Run(`Create works with minimal setup`, func(t *testing.T) {
		instance := &dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespace,
				Name:      testName,
			}}
		fakeClient := fake.NewClient()
		r := NewReconciler(context.TODO(), fakeClient, fakeClient, scheme.Scheme, instance, dtc)
		err := r.Reconcile()
		require.NoError(t, err)
	})
	t.Run(`Create AG proxy secret`, func(t *testing.T) {
		instance := &dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespace,
				Name:      testName,
			},
			Spec: dynatracev1beta1.DynaKubeSpec{
				Proxy: &dynatracev1beta1.DynaKubeProxy{Value: testProxyName},
			},
		}
		fakeClient := fake.NewClient()
		r := NewReconciler(context.TODO(), fakeClient, fakeClient, scheme.Scheme, instance, dtc)
		err := r.Reconcile()
		require.NoError(t, err)

		var proxySecret corev1.Secret
		err = fakeClient.Get(context.TODO(), types.NamespacedName{Name: "dynatrace-activegate-internal-proxy", Namespace: testNamespace}, &proxySecret)
		assert.NoError(t, err)
	})
	t.Run(`Create AG capability (creation and deletion)`, func(t *testing.T) {
		instance := &dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespace,
				Name:      testName,
			},
			Spec: dynatracev1beta1.DynaKubeSpec{
				ActiveGate: dynatracev1beta1.ActiveGateSpec{
					Capabilities: []dynatracev1beta1.CapabilityDisplayName{dynatracev1beta1.RoutingCapability.DisplayName},
				},
			},
		}
		fakeClient := fake.NewClient(testKubeSystemNamespace)
		r := NewReconciler(context.TODO(), fakeClient, fakeClient, scheme.Scheme, instance, dtc)
		err := r.Reconcile()
		require.NoError(t, err)

		var service corev1.Service
		err = fakeClient.Get(context.TODO(), types.NamespacedName{Name: testServiceName, Namespace: testNamespace}, &service)
		require.NoError(t, err)

		// remove AG from spec
		instance.Spec.ActiveGate = dynatracev1beta1.ActiveGateSpec{}
		err = r.Reconcile()
		require.NoError(t, err)

		err = fakeClient.Get(context.TODO(), types.NamespacedName{Name: testServiceName, Namespace: testNamespace}, &service)
		assert.True(t, k8serrors.IsNotFound(err))
	})
}

func TestServiceCreation(t *testing.T) {
	dynatraceClient := &dtclient.MockDynatraceClient{}
	dynakube := &dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      testName,
			Annotations: map[string]string{
				dynatracev1beta1.AnnotationFeatureActiveGateAuthToken: "false",
			},
		},
		Spec: dynatracev1beta1.DynaKubeSpec{
			ActiveGate: dynatracev1beta1.ActiveGateSpec{},
		},
	}

	t.Run("service exposes correct ports for single capabilities", func(t *testing.T) {
		expectedCapabilityPorts := map[dynatracev1beta1.CapabilityDisplayName][]string{
			dynatracev1beta1.RoutingCapability.DisplayName: {
				consts.HttpsServicePortName,
				consts.HttpServicePortName,
			},
			dynatracev1beta1.MetricsIngestCapability.DisplayName: {
				consts.HttpsServicePortName,
				consts.HttpServicePortName,
			},
			dynatracev1beta1.DynatraceApiCapability.DisplayName: {
				consts.HttpsServicePortName,
				consts.HttpServicePortName,
			},
			dynatracev1beta1.KubeMonCapability.DisplayName: {},
		}

		for capability, expectedPorts := range expectedCapabilityPorts {
			fakeClient := fake.NewClient(testKubeSystemNamespace)
			reconciler := NewReconciler(context.TODO(), fakeClient, fakeClient, scheme.Scheme, dynakube, dynatraceClient)
			dynakube.Spec.ActiveGate.Capabilities = []dynatracev1beta1.CapabilityDisplayName{
				capability,
			}

			err := reconciler.Reconcile()
			require.NoError(t, err)

			if len(expectedPorts) == 0 {
				err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: testServiceName, Namespace: testNamespace}, &corev1.Service{})

				assert.True(t, k8serrors.IsNotFound(err))
				continue
			}

			activegateService := getTestActiveGateService(t, fakeClient)
			assertContainsAllPorts(t, expectedPorts, activegateService.Spec.Ports)
		}
	})

	t.Run("service exposes correct ports for multiple capabilities", func(t *testing.T) {
		fakeClient := fake.NewClient(testKubeSystemNamespace)
		reconciler := NewReconciler(context.TODO(), fakeClient, fakeClient, scheme.Scheme, dynakube, dynatraceClient)
		dynakube.Spec.ActiveGate.Capabilities = []dynatracev1beta1.CapabilityDisplayName{
			dynatracev1beta1.RoutingCapability.DisplayName,
		}
		expectedPorts := []string{
			consts.HttpsServicePortName,
			consts.HttpServicePortName,
		}

		err := reconciler.Reconcile()
		require.NoError(t, err)

		activegateService := getTestActiveGateService(t, fakeClient)
		assertContainsAllPorts(t, expectedPorts, activegateService.Spec.Ports)
	})
}

func assertContainsAllPorts(t *testing.T, expectedPorts []string, servicePorts []corev1.ServicePort) {
	actualPorts := make([]string, 0, len(servicePorts))

	for _, servicePort := range servicePorts {
		actualPorts = append(actualPorts, servicePort.Name)
	}

	for _, expectedPort := range expectedPorts {
		assert.Contains(t, actualPorts, expectedPort)
	}
}

func getTestActiveGateService(t *testing.T, fakeClient client.Client) corev1.Service {
	var activegateService corev1.Service
	err := fakeClient.Get(context.TODO(), client.ObjectKey{Name: testServiceName, Namespace: testNamespace}, &activegateService)

	require.NoError(t, err)

	return activegateService
}

func TestExclusiveSynMonitoring(t *testing.T) {
	dtRequests := &dtclient.MockDynatraceClient{}
	dtRequests.On(
		"GetActiveGateAuthToken",
		testName,
	).Return(
		&dtclient.ActiveGateAuthTokenInfo{},
		nil)
	dynaKube := &dynatracev1beta1.DynaKube{
		ObjectMeta: syntheticCapabilityObjectMeta,
	}
	k8sRequests := fake.NewClient(testKubeSystemNamespace)
	reconciler := NewReconciler(
		context.TODO(),
		k8sRequests,
		k8sRequests,
		scheme.Scheme,
		dynaKube,
		dtRequests)
	err := reconciler.Reconcile()

	toAssertReconciliation := func(t *testing.T) {
		require.NoError(t, err, "successfully reconciled for syn-mon")
	}
	t.Run("for-errorless-reconciliation", toAssertReconciliation)

	toAssertSingleStatefulSet := func(t *testing.T) {
		var sets appsv1.StatefulSetList
		err = k8sRequests.List(
			context.TODO(),
			&sets,
			client.InNamespace(testNamespace))
		require.NoError(t, err)
		require.Len(t, sets.Items, 1)
		assert.True(
			t,
			containsKubject(
				sets.Items,
				capability.BuildServiceName(testName, capability.SyntheticName)),
			"unique StatefulSet for syn-mon")
	}
	t.Run("for-unique-statefulset", toAssertSingleStatefulSet)

	toAssertSingleAutoScaler := func(t *testing.T) {
		var autoscalers scalingv2.HorizontalPodAutoscalerList
		err = k8sRequests.List(
			context.TODO(),
			&autoscalers,
			client.InNamespace(testNamespace))
		require.NoError(t, err)
		require.Len(t, autoscalers.Items, 1)
		assert.True(
			t,
			containsKubject(
				autoscalers.Items,
				capability.BuildServiceName(testName, autoscaler.SynAutoscaler)),
			"unique HorizontalPodAutoscaler for syn-mon")
	}
	t.Run("for-unique-autoscaler", toAssertSingleAutoScaler)

	toAssertServicelessActiveGate := func(t *testing.T) {
		var services corev1.ServiceList
		err = k8sRequests.List(
			context.TODO(),
			&services,
			client.InNamespace(testNamespace))
		require.NoError(t, err)
		require.Len(t, services.Items, 0)
	}
	t.Run("for-serviceless-activegate", toAssertServicelessActiveGate)
}

func containsKubject[
	BO any,
	O interface {
		*BO
		client.Object
	},
](toScan []BO, toContainName string) bool {
	for _, scanned := range toScan {
		if O(address.Of(scanned)).GetName() == toContainName {
			return true
		}
	}

	return false
}

func TestCombinedSynAndK8sMonitoring(t *testing.T) {
	dtRequests := &dtclient.MockDynatraceClient{}
	dtRequests.On(
		"GetActiveGateAuthToken",
		testName,
	).Return(
		&dtclient.ActiveGateAuthTokenInfo{},
		nil)
	dynakube := &dynatracev1beta1.DynaKube{
		ObjectMeta: syntheticCapabilityObjectMeta,
		Spec: dynatracev1beta1.DynaKubeSpec{
			ActiveGate: dynatracev1beta1.ActiveGateSpec{
				Capabilities: []dynatracev1beta1.CapabilityDisplayName{
					dynatracev1beta1.KubeMonCapability.DisplayName,
					dynatracev1beta1.RoutingCapability.DisplayName,
				},
			},
		},
	}
	k8sRequests := fake.NewClient(testKubeSystemNamespace)
	reconciler := NewReconciler(
		context.TODO(),
		k8sRequests,
		k8sRequests,
		scheme.Scheme,
		dynakube,
		dtRequests)
	err := reconciler.Reconcile()

	toAssertReconciliation := func(t *testing.T) {
		require.NoError(t, err, "successfully reconciled for syn-mon")
	}
	t.Run("for-errorless-reconciliation", toAssertReconciliation)

	toAssertStatefulSets := func(t *testing.T) {
		var sets appsv1.StatefulSetList
		err = k8sRequests.List(
			context.TODO(),
			&sets,
			client.InNamespace(testNamespace))
		require.NoError(t, err)
		require.Len(t, sets.Items, 2)
		assert.True(
			t,
			containsKubject(
				sets.Items,
				capability.BuildServiceName(testName, capability.SyntheticName)),
			"unique StatefulSet for syn-mon")
		assert.True(
			t,
			containsKubject(
				sets.Items,
				capability.BuildServiceName(testName, consts.MultiActiveGateName)),
			"unique StatefulSet for observability-specific ActiveGate")
	}
	t.Run("for-two-statefulsets", toAssertStatefulSets)

	toAssertService := func(t *testing.T) {
		var services corev1.ServiceList
		err = k8sRequests.List(
			context.TODO(),
			&services,
			client.InNamespace(testNamespace))
		require.NoError(t, err)
		require.Len(t, services.Items, 1)
		assert.True(
			t,
			containsKubject(
				services.Items,
				capability.BuildServiceName(testName, consts.MultiActiveGateName)),
			"unique Service for observability-specific ActiveGate")
	}
	t.Run("for-service-backed-activegate", toAssertService)
}
