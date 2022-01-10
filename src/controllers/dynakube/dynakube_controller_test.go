package dynakube

import (
	"context"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/activegate/capability"
	rcap "github.com/Dynatrace/dynatrace-operator/src/controllers/activegate/reconciler/capability"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/kubesystem"
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
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	testUID       = "test-uid"
	testPaasToken = "test-paas-token"
	testAPIToken  = "test-api-token"
	testVersion   = "1.217-12345-678910"

	testUUID = "test-uuid"

	testHost     = "test-host"
	testPort     = uint32(1234)
	testProtocol = "test-protocol"

	testAnotherHost     = "test-another-host"
	testAnotherPort     = uint32(5678)
	testAnotherProtocol = "test-another-protocol"
)

func TestReconcileActiveGate_Reconcile(t *testing.T) {
	t.Run(`Reconcile works with minimal setup`, func(t *testing.T) {
		r := &ReconcileDynaKube{
			client:    fake.NewClient(),
			apiReader: fake.NewClient(),
		}
		result, err := r.Reconcile(context.TODO(), reconcile.Request{})

		assert.NoError(t, err)
		assert.NotNil(t, result)
	})
	t.Run(`Reconcile works with minimal setup and interface`, func(t *testing.T) {
		mockClient := createDTMockClient(dtclient.TokenScopes{dtclient.TokenScopeInstallerDownload},
			dtclient.TokenScopes{dtclient.TokenScopeDataExport})

		instance := &dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			}}
		r := createFakeClientAndReconcile(mockClient, instance, testPaasToken, testAPIToken)

		result, err := r.Reconcile(context.TODO(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: testName},
		})

		assert.NoError(t, err)
		assert.NotNil(t, result)
	})
	t.Run(`Reconcile reconciles Kubernetes Monitoring if enabled`, func(t *testing.T) {
		mockClient := createDTMockClient(dtclient.TokenScopes{dtclient.TokenScopeInstallerDownload},
			dtclient.TokenScopes{dtclient.TokenScopeDataExport,
				dtclient.TokenScopeReadConfig,
				dtclient.TokenScopeWriteConfig,
			})
		instance := &dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			},
			Spec: dynatracev1beta1.DynaKubeSpec{
				KubernetesMonitoring: dynatracev1beta1.KubernetesMonitoringSpec{
					Enabled: true,
				}}}
		r := createFakeClientAndReconcile(mockClient, instance, testPaasToken, testAPIToken)

		result, err := r.Reconcile(context.TODO(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: testName},
		})

		assert.NoError(t, err)
		assert.NotNil(t, result)

		var statefulSet appsv1.StatefulSet

		kubeMonCapability := capability.NewKubeMonCapability(instance)
		name := capability.CalculateStatefulSetName(kubeMonCapability, instance.Name)
		err = r.client.Get(context.TODO(), client.ObjectKey{Name: name, Namespace: testNamespace}, &statefulSet)

		assert.NoError(t, err)
		assert.NotNil(t, statefulSet)
	})
}

func TestReconcileOnlyOneTokenProvided_Reconcile(t *testing.T) {
	t.Run(`Reconcile validates apiToken correctly if apiToken with "InstallerDownload"-scope is provided`, func(t *testing.T) {
		mockClient := createDTMockClient(dtclient.TokenScopes{},
			dtclient.TokenScopes{dtclient.TokenScopeDataExport, dtclient.TokenScopeInstallerDownload})

		instance := &dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			},
			Spec: dynatracev1beta1.DynaKubeSpec{}}
		r := createFakeClientAndReconcile(mockClient, instance, "", testAPIToken)

		result, err := r.Reconcile(context.TODO(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: testName},
		})

		assert.NoError(t, err)
		assert.NotNil(t, result)

		var secret corev1.Secret

		err = r.client.Get(context.TODO(), client.ObjectKey{Name: testName, Namespace: testNamespace}, &secret)

		assert.NoError(t, err)
		assert.NotNil(t, secret)
		assert.Equal(t, string(secret.Data[dtclient.DynatraceApiToken]), testAPIToken)
	})
}

func TestReconcile_RemoveRoutingIfDisabled(t *testing.T) {
	mockClient := createDTMockClient(dtclient.TokenScopes{dtclient.TokenScopeInstallerDownload},
		dtclient.TokenScopes{dtclient.TokenScopeDataExport,
			dtclient.TokenScopeReadConfig,
			dtclient.TokenScopeWriteConfig,
		})
	instance := &dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testName,
			Namespace: testNamespace,
		},
		Spec: dynatracev1beta1.DynaKubeSpec{
			Routing: dynatracev1beta1.RoutingSpec{
				Enabled: true,
			}}}
	r := createFakeClientAndReconcile(mockClient, instance, testPaasToken, testAPIToken)
	request := reconcile.Request{
		NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: testName},
	}

	_, err := r.Reconcile(context.TODO(), request)
	assert.NoError(t, err)

	// Reconcile twice since routing service is created before the stateful set
	_, err = r.Reconcile(context.TODO(), request)
	assert.NoError(t, err)

	routingCapability := capability.NewRoutingCapability(instance)
	stsName := capability.CalculateStatefulSetName(routingCapability, testName)

	routingSts := &appsv1.StatefulSet{}
	err = r.client.Get(context.TODO(), client.ObjectKey{
		Namespace: testNamespace,
		Name:      stsName,
	}, routingSts)
	assert.NoError(t, err)
	assert.NotNil(t, routingSts)

	routingSvc := &corev1.Service{}
	err = r.client.Get(context.TODO(), client.ObjectKey{
		Namespace: testNamespace,
		Name:      rcap.BuildServiceName(testName, routingCapability.ShortName()),
	}, routingSvc)
	assert.NoError(t, err)
	assert.NotNil(t, routingSvc)

	err = r.client.Get(context.TODO(), client.ObjectKey{Name: instance.Name, Namespace: instance.Namespace}, instance)
	require.NoError(t, err)

	instance.Spec.Routing.Enabled = false
	err = r.client.Update(context.TODO(), instance)
	require.NoError(t, err)

	_, err = r.Reconcile(context.TODO(), request)
	assert.NoError(t, err)

	err = r.client.Get(context.TODO(), client.ObjectKey{
		Namespace: testNamespace,
		Name:      stsName,
	}, routingSts)
	assert.Error(t, err)
	assert.True(t, k8serrors.IsNotFound(err))

	err = r.client.Get(context.TODO(), client.ObjectKey{
		Namespace: testNamespace,
		Name:      rcap.BuildServiceName(testName, routingCapability.ShortName()),
	}, routingSvc)
	assert.Error(t, err)
	assert.True(t, k8serrors.IsNotFound(err))
}

func TestReconcile_ActiveGateMultiCapability(t *testing.T) {
	mockClient := createDTMockClient(dtclient.TokenScopes{dtclient.TokenScopeInstallerDownload},
		dtclient.TokenScopes{dtclient.TokenScopeDataExport,
			dtclient.TokenScopeReadConfig,
			dtclient.TokenScopeWriteConfig,
		})
	instance := &dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testName,
			Namespace: testNamespace,
		},
		Spec: dynatracev1beta1.DynaKubeSpec{
			ActiveGate: dynatracev1beta1.ActiveGateSpec{
				Capabilities: []dynatracev1beta1.CapabilityDisplayName{
					dynatracev1beta1.DataIngestCapability.DisplayName,
					dynatracev1beta1.KubeMonCapability.DisplayName,
					dynatracev1beta1.RoutingCapability.DisplayName,
				},
			}}}
	r := createFakeClientAndReconcile(mockClient, instance, testPaasToken, testAPIToken)
	request := reconcile.Request{
		NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: testName},
	}

	_, err := r.Reconcile(context.TODO(), request)
	assert.NoError(t, err)

	// Reconcile twice since routing service is created before the stateful set
	_, err = r.Reconcile(context.TODO(), request)
	assert.NoError(t, err)

	multiCapability := capability.NewMultiCapability(instance)
	stsName := capability.CalculateStatefulSetName(multiCapability, testName)

	routingSts := &appsv1.StatefulSet{}
	err = r.client.Get(context.TODO(), client.ObjectKey{
		Namespace: testNamespace,
		Name:      stsName,
	}, routingSts)
	assert.NoError(t, err)
	assert.NotNil(t, routingSts)

	routingSvc := &corev1.Service{}
	err = r.client.Get(context.TODO(), client.ObjectKey{
		Namespace: testNamespace,
		Name:      rcap.BuildServiceName(testName, multiCapability.ShortName()),
	}, routingSvc)
	assert.NoError(t, err)
	assert.NotNil(t, routingSvc)

	err = r.client.Get(context.TODO(), client.ObjectKey{Name: instance.Name, Namespace: instance.Namespace}, instance)
	require.NoError(t, err)

	instance.Spec.ActiveGate.Capabilities = []dynatracev1beta1.CapabilityDisplayName{}
	err = r.client.Update(context.TODO(), instance)
	require.NoError(t, err)

	_, err = r.Reconcile(context.TODO(), request)
	assert.NoError(t, err)

	err = r.client.Get(context.TODO(), client.ObjectKey{
		Namespace: testNamespace,
		Name:      stsName,
	}, routingSts)
	assert.Error(t, err)
	assert.True(t, k8serrors.IsNotFound(err))

	err = r.client.Get(context.TODO(), client.ObjectKey{
		Namespace: testNamespace,
		Name:      rcap.BuildServiceName(testName, multiCapability.ShortName()),
	}, routingSvc)
	assert.Error(t, err)
	assert.True(t, k8serrors.IsNotFound(err))
}

func createDTMockClient(paasTokenScopes, apiTokenScopes dtclient.TokenScopes) *dtclient.MockDynatraceClient {
	mockClient := &dtclient.MockDynatraceClient{}

	mockClient.On("GetCommunicationHostForClient").Return(dtclient.CommunicationHost{
		Protocol: testProtocol,
		Host:     testHost,
		Port:     testPort,
	}, nil)
	mockClient.On("GetConnectionInfo").Return(dtclient.ConnectionInfo{
		CommunicationHosts: []dtclient.CommunicationHost{
			{
				Protocol: testProtocol,
				Host:     testHost,
				Port:     testPort,
			},
			{
				Protocol: testAnotherProtocol,
				Host:     testAnotherHost,
				Port:     testAnotherPort,
			},
		},
		TenantUUID: testUUID,
	}, nil)
	mockClient.On("GetTokenScopes", testPaasToken).Return(paasTokenScopes, nil)
	mockClient.On("GetTokenScopes", testAPIToken).Return(apiTokenScopes, nil)
	mockClient.On("GetConnectionInfo").Return(dtclient.ConnectionInfo{TenantUUID: "abc123456"}, nil)
	mockClient.On("GetLatestAgentVersion", dtclient.OsUnix, dtclient.InstallerTypeDefault).Return(testVersion, nil)
	mockClient.On("GetLatestAgentVersion", dtclient.OsUnix, dtclient.InstallerTypePaaS).Return(testVersion, nil)

	return mockClient
}

func createFakeClientAndReconcile(mockClient dtclient.Client, instance *dynatracev1beta1.DynaKube, paasToken, apiToken string) *ReconcileDynaKube {
	data := map[string][]byte{
		dtclient.DynatraceApiToken: []byte(apiToken),
	}
	if paasToken != "" {
		data[dtclient.DynatracePaasToken] = []byte(paasToken)
	}

	fakeClient := fake.NewClient(instance,
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			},
			Data: data},
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: kubesystem.Namespace,
				UID:  testUID,
			}})
	r := &ReconcileDynaKube{
		client:    fakeClient,
		apiReader: fakeClient,
		scheme:    scheme.Scheme,
		dtcBuildFunc: func(DynatraceClientProperties) (dtclient.Client, error) {
			return mockClient, nil
		},
	}

	return r
}
