package activegate

import (
	"context"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/connectioninfo"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/scheme"
	"github.com/Dynatrace/dynatrace-operator/src/scheme/fake"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
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
		err = fakeClient.Get(context.TODO(), types.NamespacedName{Name: testName + "-internal-proxy", Namespace: testNamespace}, &proxySecret)
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
	t.Run("Reconcile DynaKube without Proxy after a DynaKube with proxy must not interfere with the second DKs Proxy Secret", func(t *testing.T) {
		dynaKubeWithProxy := &dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespace,
				Name:      "proxyDk",
			},
			Spec: dynatracev1beta1.DynaKubeSpec{
				Proxy: &dynatracev1beta1.DynaKubeProxy{Value: testProxyName},
			},
		}
		dynaKubeNoProxy := &dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespace,
				Name:      "noProxyDk",
			},
			Spec: dynatracev1beta1.DynaKubeSpec{},
		}
		fakeClient := fake.NewClient()
		proxyReconciler := NewReconciler(context.TODO(), fakeClient, fakeClient, scheme.Scheme, dynaKubeWithProxy, dtc)
		err := proxyReconciler.Reconcile()
		require.NoError(t, err)

		noProxyReconciler := NewReconciler(context.TODO(), fakeClient, fakeClient, scheme.Scheme, dynaKubeNoProxy, dtc)
		err = noProxyReconciler.Reconcile()
		require.NoError(t, err)

		var proxySecret corev1.Secret
		err = fakeClient.Get(context.TODO(), types.NamespacedName{Name: "proxyDk-internal-proxy", Namespace: testNamespace}, &proxySecret)
		assert.NoError(t, err)
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
			},
			dynatracev1beta1.MetricsIngestCapability.DisplayName: {
				consts.HttpsServicePortName,
				consts.HttpServicePortName,
			},
			dynatracev1beta1.DynatraceApiCapability.DisplayName: {
				consts.HttpsServicePortName,
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
	mockDtClient := &dtclient.MockDynatraceClient{}
	mockDtClient.On("GetActiveGateAuthToken", testName).
		Return(&dtclient.ActiveGateAuthTokenInfo{}, nil)

	dynakube := &dynatracev1beta1.DynaKube{
		ObjectMeta: syntheticCapabilityObjectMeta,
	}
	fakeClient := fake.NewClient(testKubeSystemNamespace)
	reconciler := NewReconciler(context.TODO(), fakeClient, fakeClient, scheme.Scheme, dynakube, mockDtClient)
	err := reconciler.Reconcile()

	require.NoError(t, err, "successfully reconciled for syn-mon")

	var statefulSets appsv1.StatefulSetList
	err = fakeClient.List(
		context.TODO(),
		&statefulSets,
		client.InNamespace(testNamespace))
	require.NoError(t, err)
	require.Len(t, statefulSets.Items, 1)

	statefulSetCreated := false
	expectedName := capability.BuildServiceName(testName, capability.SyntheticName)
	for _, statefulSet := range statefulSets.Items {
		if statefulSet.GetName() == expectedName {
			statefulSetCreated = true
			break
		}
	}
	assert.True(t, statefulSetCreated, "unique StatefulSet for syn-mon")

	var services corev1.ServiceList
	err = fakeClient.List(
		context.TODO(),
		&services,
		client.InNamespace(testNamespace))
	require.NoError(t, err)
	require.Len(t, services.Items, 0)
}

func TestReconcile_ActivegateConfigMap(t *testing.T) {
	const (
		testName            = "test-name"
		testNamespace       = "test-namespace"
		testTenantToken     = "test-token"
		testTenantUUID      = "test-uuid"
		testTenantEndpoints = "test-endpoints"
	)

	dynakube := &dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      testName,
		},
		Status: dynatracev1beta1.DynaKubeStatus{
			ActiveGate: dynatracev1beta1.ActiveGateStatus{
				ConnectionInfoStatus: dynatracev1beta1.ActiveGateConnectionInfoStatus{
					ConnectionInfoStatus: dynatracev1beta1.ConnectionInfoStatus{
						TenantUUID:  testTenantUUID,
						Endpoints:   testTenantEndpoints,
						LastRequest: metav1.Time{},
					},
				},
			},
		},
	}

	mockDtClient := &dtclient.MockDynatraceClient{}

	t.Run(`create activegate ConfigMap`, func(t *testing.T) {
		fakeClient := fake.NewClient(testKubeSystemNamespace)
		r := NewReconciler(context.TODO(), fakeClient, fakeClient, scheme.Scheme, dynakube, mockDtClient)
		err := r.Reconcile()
		require.NoError(t, err)

		var actual corev1.ConfigMap
		err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: dynakube.ActiveGateConnectionInfoConfigMapName(), Namespace: testNamespace}, &actual)
		require.NoError(t, err)
		assert.Equal(t, testTenantUUID, actual.Data[connectioninfo.TenantUUIDName])
		assert.Equal(t, testTenantEndpoints, actual.Data[connectioninfo.CommunicationEndpointsName])
	})
}
