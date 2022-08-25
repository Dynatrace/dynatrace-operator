package dynakube

import (
	"context"
	"fmt"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects/address"
	"github.com/Dynatrace/dynatrace-operator/src/kubesystem"
	"github.com/Dynatrace/dynatrace-operator/src/scheme"
	"github.com/Dynatrace/dynatrace-operator/src/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/src/version"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/src/webhook"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	testUID              = "test-uid"
	testPaasToken        = "test-paas-token"
	testAPIToken         = "test-api-token"
	testVersion          = "1.217-12345-678910"
	testComponentVersion = "test-component-version"

	testUUID     = "test-uuid"
	testObjectID = "test-object-id"

	testHost     = "test-host"
	testPort     = uint32(1234)
	testProtocol = "test-protocol"

	testAnotherHost     = "test-another-host"
	testAnotherPort     = uint32(5678)
	testAnotherProtocol = "test-another-protocol"
)

func TestMonitoringModesDynakube_Reconcile(t *testing.T) {
	deploymentModes := map[string]dynatracev1beta1.OneAgentSpec{
		"hostMonitoring":        {HostMonitoring: &dynatracev1beta1.HostInjectSpec{AutoUpdate: address.Of(false)}},
		"classicFullStack":      {ClassicFullStack: &dynatracev1beta1.HostInjectSpec{AutoUpdate: address.Of(false)}},
		"cloudNativeFullStack":  {CloudNativeFullStack: &dynatracev1beta1.CloudNativeFullStackSpec{HostInjectSpec: dynatracev1beta1.HostInjectSpec{AutoUpdate: address.Of(false)}}},
		"applicationMonitoring": {ApplicationMonitoring: &dynatracev1beta1.ApplicationMonitoringSpec{}},
	}

	for mode := range deploymentModes {
		t.Run(fmt.Sprintf(`Reconcile dynakube with %s mode`, mode), func(t *testing.T) {
			mockClient := createDTMockClient(dtclient.TokenScopes{dtclient.TokenScopeInstallerDownload},
				dtclient.TokenScopes{dtclient.TokenScopeDataExport})

			instance := &dynatracev1beta1.DynaKube{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testName,
					Namespace: testNamespace,
				},
				Spec: dynatracev1beta1.DynaKubeSpec{
					APIURL:   testHost,
					OneAgent: deploymentModes[mode],
				},
			}
			controller := createFakeClientAndReconciler(mockClient, instance, testPaasToken, testAPIToken)

			result, err := controller.Reconcile(context.TODO(), reconcile.Request{
				NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: testName},
			})

			assert.NoError(t, err)
			assert.NotNil(t, result)

			err = controller.client.Get(context.TODO(), types.NamespacedName{Namespace: testNamespace, Name: testName}, instance)
			require.NoError(t, err)
			assert.Equal(t, dynatracev1beta1.Running, instance.Status.Phase)
		})
	}
}

func TestReconcileActiveGate_Reconcile(t *testing.T) {
	t.Run(`Reconcile works with minimal setup`, func(t *testing.T) {
		controller := &DynakubeController{
			client:    fake.NewClient(),
			apiReader: fake.NewClient(),
		}
		result, err := controller.Reconcile(context.TODO(), reconcile.Request{})

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
		controller := createFakeClientAndReconciler(mockClient, instance, testPaasToken, testAPIToken)

		result, err := controller.Reconcile(context.TODO(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: testName},
		})

		assert.NoError(t, err)
		assert.NotNil(t, result)
	})
	t.Run(`Reconcile reconciles Kubernetes Monitoring if enabled`, func(t *testing.T) {
		mockClient := createDTMockClient(dtclient.TokenScopes{dtclient.TokenScopeInstallerDownload},
			dtclient.TokenScopes{dtclient.TokenScopeDataExport})
		instance := &dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			},
			Spec: dynatracev1beta1.DynaKubeSpec{
				KubernetesMonitoring: dynatracev1beta1.KubernetesMonitoringSpec{
					Enabled: true,
				}}}
		controller := createFakeClientAndReconciler(mockClient, instance, testPaasToken, testAPIToken)

		result, err := controller.Reconcile(context.TODO(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: testName},
		})

		assert.NoError(t, err)
		assert.NotNil(t, result)

		var statefulSet appsv1.StatefulSet

		kubeMonCapability := capability.NewKubeMonCapability(instance)
		name := capability.CalculateStatefulSetName(kubeMonCapability, instance.Name)
		err = controller.client.Get(context.TODO(), client.ObjectKey{Name: name, Namespace: testNamespace}, &statefulSet)

		assert.NoError(t, err)
		assert.NotNil(t, statefulSet)
	})
	t.Run(`Reconcile reconciles automatic kubernetes api monitoring`, func(t *testing.T) {
		mockClient := createDTMockClient(dtclient.TokenScopes{dtclient.TokenScopeInstallerDownload},
			dtclient.TokenScopes{dtclient.TokenScopeDataExport, dtclient.TokenScopeEntitiesRead, dtclient.TokenScopeSettingsRead, dtclient.TokenScopeSettingsWrite})
		instance := &dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
				Annotations: map[string]string{
					dynatracev1beta1.AnnotationFeatureAutomaticK8sApiMonitoring: "true",
				},
			},
			Spec: dynatracev1beta1.DynaKubeSpec{
				ActiveGate: dynatracev1beta1.ActiveGateSpec{
					Capabilities: []dynatracev1beta1.CapabilityDisplayName{
						dynatracev1beta1.KubeMonCapability.DisplayName,
					},
				},
			}}
		controller := createFakeClientAndReconciler(mockClient, instance, testPaasToken, testAPIToken)

		result, err := controller.Reconcile(context.TODO(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: testName},
		})

		mockClient.AssertCalled(t, "CreateOrUpdateKubernetesSetting",
			testName,
			testUID,
			mock.AnythingOfType("string"))
		assert.NoError(t, err)
		assert.Equal(t, false, result.Requeue)
	})
	t.Run(`Reconcile reconciles automatic kubernetes api monitoring with custom cluster name`, func(t *testing.T) {
		const clusterLabel = "..blabla..;.🙃"

		mockClient := createDTMockClient(dtclient.TokenScopes{dtclient.TokenScopeInstallerDownload},
			dtclient.TokenScopes{dtclient.TokenScopeDataExport, dtclient.TokenScopeEntitiesRead, dtclient.TokenScopeSettingsRead, dtclient.TokenScopeSettingsWrite})
		mockClient.On("CreateOrUpdateKubernetesSetting",
			mock.AnythingOfType("string"),
			mock.AnythingOfType("string"),
			mock.AnythingOfType("string")).Return(testUID, nil)

		instance := &dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
				Annotations: map[string]string{
					dynatracev1beta1.AnnotationFeatureAutomaticK8sApiMonitoring:            "true",
					dynatracev1beta1.AnnotationFeatureAutomaticK8sApiMonitoringClusterName: clusterLabel,
				},
			},
			Spec: dynatracev1beta1.DynaKubeSpec{
				ActiveGate: dynatracev1beta1.ActiveGateSpec{
					Capabilities: []dynatracev1beta1.CapabilityDisplayName{
						dynatracev1beta1.KubeMonCapability.DisplayName,
					},
				},
			}}
		controller := createFakeClientAndReconciler(mockClient, instance, testPaasToken, testAPIToken)

		result, err := controller.Reconcile(context.TODO(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: testName},
		})

		mockClient.AssertCalled(t, "CreateOrUpdateKubernetesSetting",
			clusterLabel,
			testUID,
			mock.AnythingOfType("string"))

		assert.NoError(t, err)
		assert.Equal(t, false, result.Requeue)
	})
	t.Run(`Reconcile reconciles proxy secret`, func(t *testing.T) {
		mockClient := createDTMockClient(dtclient.TokenScopes{dtclient.TokenScopeInstallerDownload},
			dtclient.TokenScopes{dtclient.TokenScopeDataExport})
		instance := &dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			},
			Spec: dynatracev1beta1.DynaKubeSpec{
				Proxy: &dynatracev1beta1.DynaKubeProxy{
					Value:     "https://proxy:1234",
					ValueFrom: "",
				}}}
		controller := createFakeClientAndReconciler(mockClient, instance, testPaasToken, testAPIToken)

		result, err := controller.Reconcile(context.TODO(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: testName},
		})

		assert.NoError(t, err)
		assert.NotNil(t, result)

		var proxySecret corev1.Secret
		name := capability.BuildProxySecretName()
		err = controller.client.Get(context.TODO(), client.ObjectKey{Name: name, Namespace: testNamespace}, &proxySecret)

		assert.NoError(t, err)
		assert.NotNil(t, proxySecret)
	})
	t.Run(`has proxy secret but feature flag disables proxy`, func(t *testing.T) {
		mockClient := createDTMockClient(dtclient.TokenScopes{dtclient.TokenScopeInstallerDownload},
			dtclient.TokenScopes{dtclient.TokenScopeDataExport})
		instance := &dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			},
			Spec: dynatracev1beta1.DynaKubeSpec{
				Proxy: &dynatracev1beta1.DynaKubeProxy{
					Value:     "https://proxy:1234",
					ValueFrom: "",
				}}}
		instance.Annotations = map[string]string{dynatracev1beta1.AnnotationFeatureActiveGateIgnoreProxy: "true"}
		controller := createFakeClientAndReconciler(mockClient, instance, testPaasToken, testAPIToken)

		result, err := controller.Reconcile(context.TODO(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: testName},
		})

		assert.NoError(t, err)
		assert.NotNil(t, result)

		var proxySecret corev1.Secret
		name := capability.BuildProxySecretName()
		err = controller.client.Get(context.TODO(), client.ObjectKey{Name: name, Namespace: testNamespace}, &proxySecret)

		assert.Error(t, err)
		assert.True(t, k8serrors.IsNotFound(err))
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
		controller := createFakeClientAndReconciler(mockClient, instance, "", testAPIToken)

		result, err := controller.Reconcile(context.TODO(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: testName},
		})

		assert.NoError(t, err)
		assert.NotNil(t, result)

		var secret corev1.Secret

		err = controller.client.Get(context.TODO(), client.ObjectKey{Name: testName, Namespace: testNamespace}, &secret)

		assert.NoError(t, err)
		assert.NotNil(t, secret)
		assert.Equal(t, string(secret.Data[dtclient.DynatraceApiToken]), testAPIToken)
	})
}

func TestRemoveOneAgentDaemonset(t *testing.T) {
	t.Run(`Reconcile validates apiToken correctly if apiToken with "InstallerDownload"-scope is provided`, func(t *testing.T) {
		mockClient := createDTMockClient(dtclient.TokenScopes{},
			dtclient.TokenScopes{
				dtclient.TokenScopeDataExport,
				dtclient.TokenScopeInstallerDownload})
		instance := &dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			},
			Spec: dynatracev1beta1.DynaKubeSpec{}}
		data := map[string][]byte{
			dtclient.DynatraceApiToken: []byte(testAPIToken),
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
				},
			},
			&appsv1.DaemonSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      instance.OneAgentDaemonsetName(),
					Namespace: testNamespace,
				},
			},
		)
		controller := &DynakubeController{
			client:    fakeClient,
			apiReader: fakeClient,
			scheme:    scheme.Scheme,
			dtcBuildFunc: func(DynatraceClientProperties) (dtclient.Client, error) {
				return mockClient, nil
			},
		}

		result, err := controller.Reconcile(context.TODO(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: testName},
		})

		assert.NoError(t, err)
		assert.NotNil(t, result)

		var daemonSet appsv1.DaemonSet

		err = controller.client.Get(context.TODO(), client.ObjectKey{Name: (instance.OneAgentDaemonsetName()), Namespace: testNamespace}, &daemonSet)

		assert.Error(t, err)
	})
}

func TestReconcile_RemoveRoutingIfDisabled(t *testing.T) {
	mockClient := createDTMockClient(dtclient.TokenScopes{dtclient.TokenScopeInstallerDownload},
		dtclient.TokenScopes{dtclient.TokenScopeDataExport})
	instance := &dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testName,
			Namespace: testNamespace,
		},
		Spec: dynatracev1beta1.DynaKubeSpec{
			Routing: dynatracev1beta1.RoutingSpec{
				Enabled: true,
			}}}
	controller := createFakeClientAndReconciler(mockClient, instance, testPaasToken, testAPIToken)
	request := reconcile.Request{
		NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: testName},
	}

	_, err := controller.Reconcile(context.TODO(), request)
	assert.NoError(t, err)

	// Reconcile twice since routing service is created before the stateful set
	_, err = controller.Reconcile(context.TODO(), request)
	assert.NoError(t, err)

	routingCapability := capability.NewRoutingCapability(instance)
	stsName := capability.CalculateStatefulSetName(routingCapability, testName)

	routingSts := &appsv1.StatefulSet{}
	err = controller.client.Get(context.TODO(), client.ObjectKey{
		Namespace: testNamespace,
		Name:      stsName,
	}, routingSts)
	assert.NoError(t, err)
	assert.NotNil(t, routingSts)

	routingSvc := &corev1.Service{}
	err = controller.client.Get(context.TODO(), client.ObjectKey{
		Namespace: testNamespace,
		Name:      testName + "-" + routingCapability.ShortName(),
	}, routingSvc)
	assert.NoError(t, err)
	assert.NotNil(t, routingSvc)

	err = controller.client.Get(context.TODO(), client.ObjectKey{Name: instance.Name, Namespace: instance.Namespace}, instance)
	require.NoError(t, err)

	instance.Spec.Routing.Enabled = false
	err = controller.client.Update(context.TODO(), instance)
	require.NoError(t, err)

	_, err = controller.Reconcile(context.TODO(), request)
	assert.NoError(t, err)

	err = controller.client.Get(context.TODO(), client.ObjectKey{
		Namespace: testNamespace,
		Name:      stsName,
	}, routingSts)
	assert.Error(t, err)
	assert.True(t, k8serrors.IsNotFound(err))

	err = controller.client.Get(context.TODO(), client.ObjectKey{
		Namespace: testNamespace,
		Name:      testName + "-" + routingCapability.ShortName(),
	}, routingSvc)
	assert.Error(t, err)
	assert.True(t, k8serrors.IsNotFound(err))
}

func TestReconcile_ActiveGateMultiCapability(t *testing.T) {
	mockClient := createDTMockClient(dtclient.TokenScopes{dtclient.TokenScopeInstallerDownload},
		dtclient.TokenScopes{
			dtclient.TokenScopeDataExport,
			dtclient.TokenScopeMetricsIngest,
		})
	instance := &dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testName,
			Namespace: testNamespace,
		},
		Spec: dynatracev1beta1.DynaKubeSpec{
			ActiveGate: dynatracev1beta1.ActiveGateSpec{
				Capabilities: []dynatracev1beta1.CapabilityDisplayName{
					dynatracev1beta1.MetricsIngestCapability.DisplayName,
					dynatracev1beta1.KubeMonCapability.DisplayName,
					dynatracev1beta1.RoutingCapability.DisplayName,
				},
			}},
		Status: dynatracev1beta1.DynaKubeStatus{
			ActiveGate: dynatracev1beta1.ActiveGateStatus{
				VersionStatus: dynatracev1beta1.VersionStatus{
					Version: testComponentVersion,
				},
			},
		}}
	r := createFakeClientAndReconciler(mockClient, instance, testPaasToken, testAPIToken)
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
		Name:      testName + "-" + multiCapability.ShortName(),
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
		Name:      testName + "-" + multiCapability.ShortName(),
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
	mockClient.On("GetMonitoredEntitiesForKubeSystemUUID", mock.AnythingOfType("string")).
		Return([]dtclient.MonitoredEntity{}, nil)
	mockClient.On("GetSettingsForMonitoredEntities", []dtclient.MonitoredEntity{}).
		Return(dtclient.GetSettingsResponse{}, nil)
	mockClient.On("CreateOrUpdateKubernetesSetting", testName, testUID, mock.AnythingOfType("string")).
		Return(testObjectID, nil)
	mockClient.On("GetAgentTenantInfo").Return(&dtclient.AgentTenantInfo{}, nil)
	mockClient.On("GetActiveGateTenantInfo").Return(&dtclient.ActiveGateTenantInfo{}, nil)

	return mockClient
}

func createFakeClientAndReconciler(mockClient dtclient.Client, instance *dynatracev1beta1.DynaKube, paasToken, apiToken string) *DynakubeController {
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
			},
		},
		generateStatefulSetForTesting(testName, testNamespace, "activegate", testUID),
	)

	controller := &DynakubeController{
		client:    fakeClient,
		apiReader: fakeClient,
		scheme:    scheme.Scheme,
		dtcBuildFunc: func(DynatraceClientProperties) (dtclient.Client, error) {
			return mockClient, nil
		},
	}

	return controller
}

// generateStatefulSetForTesting prepares an ActiveGate StatefulSet after a Reconciliation of the Dynakube with a specific feature enabled
func generateStatefulSetForTesting(name, namespace, feature, kubeSystemUUID string) *appsv1.StatefulSet {
	expectedLabels := map[string]string{
		kubeobjects.AppNameLabel:      kubeobjects.ActiveGateComponentLabel,
		kubeobjects.AppVersionLabel:   testComponentVersion,
		kubeobjects.AppComponentLabel: feature,
		kubeobjects.AppCreatedByLabel: name,
		kubeobjects.AppManagedByLabel: version.AppName,
	}
	expectedMatchLabels := map[string]string{
		kubeobjects.AppNameLabel:      kubeobjects.ActiveGateComponentLabel,
		kubeobjects.AppManagedByLabel: version.AppName,
		kubeobjects.AppCreatedByLabel: name,
	}
	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name + "-" + feature,
			Namespace: namespace,
			Labels:    expectedLabels,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "dynatrace.com/v1beta1",
					Kind:       "DynaKube",
					Name:       name,
				},
			},
		},
		Spec: appsv1.StatefulSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: expectedMatchLabels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: expectedLabels,
					Annotations: map[string]string{
						"internal.operator.dynatrace.com/custom-properties-hash": "",
						"internal.operator.dynatrace.com/version":                "",
					},
				},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: "truststore-volume",
						},
					},
					InitContainers: []corev1.Container{
						{
							Name: "certificate-loader",
							Command: []string{
								"/bin/bash",
							},
							Args: []string{
								"-c",
								"/opt/dynatrace/gateway/k8scrt2jks.sh",
							},
							WorkingDir: "/var/lib/dynatrace/gateway",
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:             "truststore-volume",
									MountPath:        "/var/lib/dynatrace/gateway/ssl",
									MountPropagation: (*corev1.MountPropagationMode)(nil),
								},
							},
							ImagePullPolicy: "Always",
						},
					},
					Containers: []corev1.Container{
						{
							Name: feature,
							Env: []corev1.EnvVar{
								{
									Name:  "DT_CAPABILITIES",
									Value: "kubernetes_monitoring",
								},
								{
									Name:  "DT_ID_SEED_NAMESPACE",
									Value: namespace,
								},
								{
									Name:  "DT_ID_SEED_K8S_CLUSTER_ID",
									Value: kubeSystemUUID,
								},
								{
									Name:  "DT_DEPLOYMENT_METADATA",
									Value: "orchestration_tech=Operator-active_gate;script_version=snapshot;orchestrator_id=" + kubeSystemUUID,
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "truststore-volume",
									ReadOnly:  true,
									MountPath: "/opt/dynatrace/gateway/jre/lib/security/cacerts",
									SubPath:   "k8s-local.jks",
								},
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/rest/health",
										Port: intstr.IntOrString{
											IntVal: 9999,
										},
										Scheme: "HTTPS",
									},
								},
								InitialDelaySeconds: 90,
								PeriodSeconds:       15,
								FailureThreshold:    3,
							},
							ImagePullPolicy: "Always",
						},
					},
					ServiceAccountName: "dynatrace-kubernetes-monitoring",
					ImagePullSecrets: []corev1.LocalObjectReference{
						{
							Name: name + "-pull-secret",
						},
					},
					Affinity: &corev1.Affinity{
						NodeAffinity: &corev1.NodeAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
								NodeSelectorTerms: []corev1.NodeSelectorTerm{
									{
										MatchExpressions: []corev1.NodeSelectorRequirement{
											{
												Key:      "kubernetes.io/arch",
												Operator: "In",
												Values: []string{
													"amd64",
												},
											},
											{
												Key:      "kubernetes.io/os",
												Operator: "In",
												Values: []string{
													"linux",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			PodManagementPolicy: "Parallel",
		},
	}
}

type errorClient struct {
	client.Client
}

func (clt errorClient) Get(_ context.Context, _ client.ObjectKey, _ client.Object) error {
	return errors.New("fake error")
}

func TestGetDynakube(t *testing.T) {
	t.Run("get dynakube", func(t *testing.T) {
		fakeClient := fake.NewClient(&dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			},
			Spec: dynatracev1beta1.DynaKubeSpec{
				OneAgent: dynatracev1beta1.OneAgentSpec{
					CloudNativeFullStack: &dynatracev1beta1.CloudNativeFullStackSpec{},
				},
			},
		})
		controller := &DynakubeController{
			client:    fakeClient,
			apiReader: fakeClient,
		}
		ctx := context.TODO()
		dynakube, err := controller.getDynakubeOrUnmap(ctx, testName, testNamespace)

		assert.NotNil(t, dynakube)
		assert.NoError(t, err)

		assert.Equal(t, testName, dynakube.Name)
		assert.Equal(t, testNamespace, dynakube.Namespace)
		assert.NotNil(t, dynakube.Spec.OneAgent.CloudNativeFullStack)
	})
	t.Run("unmap if not not found", func(t *testing.T) {
		namespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:   testNamespace,
				Labels: map[string]string{dtwebhook.InjectionInstanceLabel: testName},
			},
		}
		fakeClient := fake.NewClient(namespace)
		controller := &DynakubeController{
			client:    fakeClient,
			apiReader: fakeClient,
		}
		ctx := context.TODO()
		dynakube, err := controller.getDynakubeOrUnmap(ctx, testName, testNamespace)

		assert.Nil(t, dynakube)
		assert.NoError(t, err)

		err = fakeClient.Get(ctx, client.ObjectKey{Name: testNamespace}, namespace)
		require.NoError(t, err)
		assert.NotContains(t, namespace.Labels, dtwebhook.InjectionInstanceLabel)
	})
	t.Run("return unknown error", func(t *testing.T) {
		controller := &DynakubeController{
			client:    errorClient{},
			apiReader: errorClient{},
		}

		ctx := context.TODO()
		dynakube, err := controller.getDynakubeOrUnmap(ctx, testName, testNamespace)

		assert.Nil(t, dynakube)
		assert.EqualError(t, err, "fake error")
	})
}

func TestReconcileIstio(t *testing.T) {
	fakeClient := fake.NewClient()
	dynakube := &dynatracev1beta1.DynaKube{}
	controller := &DynakubeController{
		client:    fakeClient,
		apiReader: fakeClient,
	}
	updated := controller.reconcileIstio(dynakube)

	assert.False(t, updated)

	// Testing what happens if the flag is enabled is not testable without some bigger refactoring
}
