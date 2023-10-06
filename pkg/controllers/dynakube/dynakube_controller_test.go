package dynakube

import (
	"context"
	"fmt"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/oci/registry"
	"github.com/Dynatrace/dynatrace-operator/pkg/oci/registry/mocks"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/address"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubesystem"
	"net/http"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/dynatraceclient"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/istio"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/token"
	"github.com/Dynatrace/dynatrace-operator/pkg/dtclient"
	"github.com/Dynatrace/dynatrace-operator/pkg/version"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	containerv1 "github.com/google/go-containerregistry/pkg/v1"
	fakecontainer "github.com/google/go-containerregistry/pkg/v1/fake"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	fakeistio "istio.io/client-go/pkg/clientset/versioned/fake"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	fakediscovery "k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/rest"
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

	testName      = "test-name"
	testNamespace = "test-namespace"

	testApiUrl = "https://" + testHost + "/e/" + testUUID + "/api"
)

func TestMonitoringModesDynakube_Reconcile(t *testing.T) {
	deploymentModes := map[string]dynatracev1beta1.OneAgentSpec{
		"hostMonitoring":        {HostMonitoring: &dynatracev1beta1.HostInjectSpec{AutoUpdate: address.Of(false)}},
		"classicFullStack":      {ClassicFullStack: &dynatracev1beta1.HostInjectSpec{AutoUpdate: address.Of(false)}},
		"cloudNativeFullStack":  {CloudNativeFullStack: &dynatracev1beta1.CloudNativeFullStackSpec{HostInjectSpec: dynatracev1beta1.HostInjectSpec{AutoUpdate: address.Of(false)}}},
		"applicationMonitoring": {ApplicationMonitoring: &dynatracev1beta1.ApplicationMonitoringSpec{}},
	}

	for mode := range deploymentModes {
		t.Run(fmt.Sprintf(`Create dynakube with %s mode`, mode), func(t *testing.T) {
			mockClient := createDTMockClient(dtclient.TokenScopes{dtclient.TokenScopeInstallerDownload},
				dtclient.TokenScopes{dtclient.TokenScopeDataExport, dtclient.TokenScopeActiveGateTokenCreate},
			)

			mockClient.On("GetActiveGateAuthToken", testName).Return(&dtclient.ActiveGateAuthTokenInfo{}, nil)

			instance := &dynatracev1beta1.DynaKube{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testName,
					Namespace: testNamespace,
				},
				Spec: dynatracev1beta1.DynaKubeSpec{
					APIURL:   testApiUrl,
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
			assert.Equal(t, status.Running, instance.Status.Phase)
		})
	}
}

func TestReconcileActiveGate_Reconcile(t *testing.T) {
	t.Run(`Create works with minimal setup`, func(t *testing.T) {
		controller := &Controller{
			client:    fake.NewClient(),
			apiReader: fake.NewClient(),
		}
		result, err := controller.Reconcile(context.TODO(), reconcile.Request{})

		assert.NoError(t, err)
		assert.NotNil(t, result)
	})
	t.Run(`Create works with minimal setup and interface`, func(t *testing.T) {
		mockClient := createDTMockClient(dtclient.TokenScopes{dtclient.TokenScopeInstallerDownload},
			dtclient.TokenScopes{dtclient.TokenScopeDataExport, dtclient.TokenScopeActiveGateTokenCreate})

		mockClient.On("GetActiveGateAuthToken", testName).Return(&dtclient.ActiveGateAuthTokenInfo{}, nil)

		instance := &dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			},
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: testApiUrl,
			},
		}
		controller := createFakeClientAndReconciler(mockClient, instance, testPaasToken, testAPIToken)

		result, err := controller.Reconcile(context.TODO(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: testName},
		})

		assert.NoError(t, err)
		assert.NotNil(t, result)
	})
	t.Run(`Create reconciles Kubernetes Monitoring if enabled`, func(t *testing.T) {
		mockClient := createDTMockClient(dtclient.TokenScopes{dtclient.TokenScopeInstallerDownload},
			dtclient.TokenScopes{dtclient.TokenScopeDataExport, dtclient.TokenScopeActiveGateTokenCreate})

		mockClient.On("GetActiveGateAuthToken", testName).Return(&dtclient.ActiveGateAuthTokenInfo{}, nil)

		instance := &dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			},
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: testApiUrl,
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
	t.Run(`Create reconciles automatic kubernetes api monitoring`, func(t *testing.T) {
		mockClient := createDTMockClient(dtclient.TokenScopes{dtclient.TokenScopeInstallerDownload},
			dtclient.TokenScopes{dtclient.TokenScopeDataExport, dtclient.TokenScopeEntitiesRead, dtclient.TokenScopeSettingsRead, dtclient.TokenScopeSettingsWrite,
				dtclient.TokenScopeActiveGateTokenCreate})

		mockClient.On("GetActiveGateAuthToken", testName).Return(&dtclient.ActiveGateAuthTokenInfo{}, nil)

		instance := &dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
				Annotations: map[string]string{
					dynatracev1beta1.AnnotationFeatureAutomaticK8sApiMonitoring: "true",
				},
			},
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: testApiUrl,
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
	t.Run(`Create reconciles automatic kubernetes api monitoring with custom cluster name`, func(t *testing.T) {
		const clusterLabel = "..blabla..;.🙃"

		mockClient := createDTMockClient(dtclient.TokenScopes{dtclient.TokenScopeInstallerDownload},
			dtclient.TokenScopes{dtclient.TokenScopeDataExport, dtclient.TokenScopeEntitiesRead, dtclient.TokenScopeSettingsRead, dtclient.TokenScopeSettingsWrite,
				dtclient.TokenScopeActiveGateTokenCreate})
		mockClient.On("CreateOrUpdateKubernetesSetting",
			mock.AnythingOfType("string"),
			mock.AnythingOfType("string"),
			mock.AnythingOfType("string")).Return(testUID, nil)
		mockClient.On("GetActiveGateAuthToken", testName).Return(&dtclient.ActiveGateAuthTokenInfo{}, nil)

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
				APIURL: testApiUrl,
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
	t.Run(`Create reconciles proxy secret`, func(t *testing.T) {
		mockClient := createDTMockClient(dtclient.TokenScopes{dtclient.TokenScopeInstallerDownload},
			dtclient.TokenScopes{dtclient.TokenScopeDataExport, dtclient.TokenScopeActiveGateTokenCreate})

		mockClient.On("GetActiveGateAuthToken", testName).Return(&dtclient.ActiveGateAuthTokenInfo{}, nil)

		instance := &dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			},
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: testApiUrl,
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
		name := capability.BuildProxySecretName(testName)
		err = controller.client.Get(context.TODO(), client.ObjectKey{Name: name, Namespace: testNamespace}, &proxySecret)

		assert.NoError(t, err)
		assert.NotNil(t, proxySecret)
	})
	t.Run(`has proxy secret but feature flag disables proxy`, func(t *testing.T) {
		mockClient := createDTMockClient(dtclient.TokenScopes{dtclient.TokenScopeInstallerDownload},
			dtclient.TokenScopes{dtclient.TokenScopeDataExport, dtclient.TokenScopeActiveGateTokenCreate})

		mockClient.On("GetActiveGateAuthToken", testName).Return(&dtclient.ActiveGateAuthTokenInfo{}, nil)

		instance := &dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			},
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: testApiUrl,
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
		name := capability.BuildProxySecretName(testName)
		err = controller.client.Get(context.TODO(), client.ObjectKey{Name: name, Namespace: testNamespace}, &proxySecret)

		assert.Error(t, err)
		assert.True(t, k8serrors.IsNotFound(err))
	})
	t.Run(`reconciles phase change correctly`, func(t *testing.T) {
		mockClient := createDTMockClient(dtclient.TokenScopes{dtclient.TokenScopeInstallerDownload},
			dtclient.TokenScopes{dtclient.TokenScopeDataExport, dtclient.TokenScopeEntitiesRead, dtclient.TokenScopeSettingsRead, dtclient.TokenScopeSettingsWrite, dtclient.TokenScopeActiveGateTokenCreate})

		mockClient.On("GetActiveGateAuthToken", testName).Return(&dtclient.ActiveGateAuthTokenInfo{}, nil)

		instance := &dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
				Annotations: map[string]string{
					dynatracev1beta1.AnnotationFeatureAutomaticK8sApiMonitoring: "true",
				},
			},
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: testApiUrl,
				ActiveGate: dynatracev1beta1.ActiveGateSpec{
					Capabilities: []dynatracev1beta1.CapabilityDisplayName{
						dynatracev1beta1.KubeMonCapability.DisplayName,
					},
				},
			}}
		controller := createFakeClientAndReconciler(mockClient, instance, testPaasToken, testAPIToken)
		// Remove existing StatefulSet created by createFakeClientAndReconciler
		require.NoError(t, controller.client.Delete(context.TODO(), &appsv1.StatefulSet{ObjectMeta: metav1.ObjectMeta{Name: testName + "-activegate", Namespace: testNamespace}}))

		result, err := controller.Reconcile(context.TODO(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: testName},
		})

		mockClient.AssertCalled(t, "CreateOrUpdateKubernetesSetting",
			testName,
			testUID,
			mock.AnythingOfType("string"))

		assert.NoError(t, err)
		assert.Equal(t, false, result.Requeue)

		var activeGateStatefulSet appsv1.StatefulSet
		assert.NoError(t,
			controller.client.Get(context.TODO(), client.ObjectKey{Name: testName + "-activegate", Namespace: testNamespace}, &activeGateStatefulSet))
		assert.NoError(t, controller.client.Get(context.TODO(), client.ObjectKey{Name: testName, Namespace: testNamespace}, instance))
		assert.Equal(t, status.Running, instance.Status.Phase)
	})
}

func TestReconcileOnlyOneTokenProvided_Reconcile(t *testing.T) {
	t.Run(`Create validates apiToken correctly if apiToken with "InstallerDownload"-scope is provided`, func(t *testing.T) {
		mockClient := createDTMockClient(dtclient.TokenScopes{},
			dtclient.TokenScopes{dtclient.TokenScopeDataExport, dtclient.TokenScopeInstallerDownload, dtclient.TokenScopeActiveGateTokenCreate})

		mockClient.On("GetActiveGateAuthToken", testName).Return(&dtclient.ActiveGateAuthTokenInfo{}, nil)

		instance := &dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			},
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: testApiUrl,
			}}
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
	t.Run(`Create validates apiToken correctly if apiToken with "InstallerDownload"-scope is provided`, func(t *testing.T) {
		mockClient := createDTMockClient(dtclient.TokenScopes{},
			dtclient.TokenScopes{
				dtclient.TokenScopeDataExport,
				dtclient.TokenScopeInstallerDownload,
				dtclient.TokenScopeActiveGateTokenCreate})

		mockClient.On("GetActiveGateAuthToken", testName).Return(&dtclient.ActiveGateAuthTokenInfo{}, nil)

		instance := &dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			},
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: testApiUrl,
			}}
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
		mockDtcBuilder := &dynatraceclient.StubBuilder{
			DynatraceClient: mockClient,
		}

		controller := &Controller{
			client:                 fakeClient,
			apiReader:              fakeClient,
			scheme:                 scheme.Scheme,
			dynatraceClientBuilder: mockDtcBuilder,
			registryClientBuilder:  createFakeRegistryClientBuilder(),
		}

		result, err := controller.Reconcile(context.TODO(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: testName},
		})

		assert.NoError(t, err)
		assert.NotNil(t, result)

		var daemonSet appsv1.DaemonSet

		err = controller.client.Get(context.TODO(), client.ObjectKey{Name: instance.OneAgentDaemonsetName(), Namespace: testNamespace}, &daemonSet)

		assert.Error(t, err)
	})
}

func TestReconcile_RemoveRoutingIfDisabled(t *testing.T) {
	mockClient := createDTMockClient(dtclient.TokenScopes{dtclient.TokenScopeInstallerDownload},
		dtclient.TokenScopes{dtclient.TokenScopeDataExport, dtclient.TokenScopeActiveGateTokenCreate})

	mockClient.On("GetActiveGateAuthToken", testName).Return(&dtclient.ActiveGateAuthTokenInfo{}, nil)

	instance := &dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testName,
			Namespace: testNamespace,
		},
		Spec: dynatracev1beta1.DynaKubeSpec{
			APIURL: testApiUrl,
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
			dtclient.TokenScopeActiveGateTokenCreate,
		})

	mockClient.On("GetActiveGateAuthToken", testName).Return(&dtclient.ActiveGateAuthTokenInfo{}, nil)

	instance := &dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testName,
			Namespace: testNamespace,
		},
		Spec: dynatracev1beta1.DynaKubeSpec{
			APIURL: testApiUrl,
			ActiveGate: dynatracev1beta1.ActiveGateSpec{
				Capabilities: []dynatracev1beta1.CapabilityDisplayName{
					dynatracev1beta1.MetricsIngestCapability.DisplayName,
					dynatracev1beta1.KubeMonCapability.DisplayName,
					dynatracev1beta1.RoutingCapability.DisplayName,
				},
			}},
	}

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
	mockClient.On("GetOneAgentConnectionInfo").Return(dtclient.OneAgentConnectionInfo{
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
		ConnectionInfo: dtclient.ConnectionInfo{
			TenantUUID: testUUID,
		},
	}, nil)
	mockClient.On("GetTokenScopes", testPaasToken).Return(paasTokenScopes, nil)
	mockClient.On("GetTokenScopes", testAPIToken).Return(apiTokenScopes, nil)
	mockClient.On("GetOneAgentConnectionInfo").Return(
		dtclient.OneAgentConnectionInfo{
			ConnectionInfo: dtclient.ConnectionInfo{
				TenantUUID: "abc123456",
			},
		}, nil)
	mockClient.On("GetLatestAgentVersion", mock.Anything, mock.Anything).Return(testVersion, nil)
	mockClient.On("GetMonitoredEntitiesForKubeSystemUUID", mock.AnythingOfType("string")).
		Return([]dtclient.MonitoredEntity{}, nil)
	mockClient.On("GetSettingsForMonitoredEntities", []dtclient.MonitoredEntity{}).
		Return(dtclient.GetSettingsResponse{}, nil)
	mockClient.On("CreateOrUpdateKubernetesSetting", testName, testUID, mock.AnythingOfType("string")).
		Return(testObjectID, nil)
	mockClient.On("GetActiveGateConnectionInfo").Return(&dtclient.ActiveGateConnectionInfo{}, nil)

	return mockClient
}

func createFakeRegistryClientBuilder() func(options ...func(*registry.Client)) (registry.ImageGetter, error) {
	fakeRegistryClient := &mocks.MockImageGetter{}
	fakeImage := &fakecontainer.FakeImage{}
	fakeImage.ConfigFileStub = func() (*containerv1.ConfigFile, error) {
		return &containerv1.ConfigFile{}, nil
	}
	fakeImage.ConfigFile()
	image := containerv1.Image(fakeImage)
	fakeRegistryClient.On("GetImageVersion", mock.Anything, mock.Anything).Return(registry.ImageVersion{Version: "1.2.3.4-5"}, nil)
	fakeRegistryClient.On("PullImageInfo", mock.Anything, mock.Anything).Return(&image, nil)

	return func(options ...func(*registry.Client)) (registry.ImageGetter, error) {
		return fakeRegistryClient, nil
	}
}

func createFakeClientAndReconciler(mockClient dtclient.Client, instance *dynatracev1beta1.DynaKube, paasToken, apiToken string) *Controller {
	data := map[string][]byte{
		dtclient.DynatraceApiToken: []byte(apiToken),
	}
	if paasToken != "" {
		data[dtclient.DynatracePaasToken] = []byte(paasToken)
	}

	fakeClient := fake.NewClientWithIndex(instance,
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
	mockDtcBuilder := &dynatraceclient.StubBuilder{
		DynatraceClient: mockClient,
	}

	controller := &Controller{
		client:                 fakeClient,
		apiReader:              fakeClient,
		registryClientBuilder:  createFakeRegistryClientBuilder(),
		scheme:                 scheme.Scheme,
		dynatraceClientBuilder: mockDtcBuilder,
		fs:                     afero.Afero{Fs: afero.NewMemMapFs()},
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

func (clt errorClient) Get(_ context.Context, _ client.ObjectKey, _ client.Object, _ ...client.GetOption) error {
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
		controller := &Controller{
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
		controller := &Controller{
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
		controller := &Controller{
			client:    errorClient{},
			apiReader: errorClient{},
		}

		ctx := context.TODO()
		dynakube, err := controller.getDynakubeOrUnmap(ctx, testName, testNamespace)

		assert.Nil(t, dynakube)
		assert.EqualError(t, err, "fake error")
	})
}

func TestTokenConditions(t *testing.T) {
	t.Run("token condition error is set if token are invalid", func(t *testing.T) {
		fakeClient := fake.NewClient()
		dynakube := &dynatracev1beta1.DynaKube{}
		controller := &Controller{
			client:    fakeClient,
			apiReader: fakeClient,
		}

		err := controller.reconcileDynaKube(context.TODO(), dynakube)

		assert.Error(t, err)
		assertCondition(t, dynakube, dynatracev1beta1.TokenConditionType, metav1.ConditionFalse, dynatracev1beta1.ReasonTokenError, "secrets \"\" not found")
	})
	t.Run("token condition is set if token are valid", func(t *testing.T) {
		dynakube := &dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			},
		}
		fakeClient := fake.NewClient(&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			},
			Data: map[string][]byte{
				dtclient.DynatraceApiToken: []byte(testAPIToken),
			},
		})
		mockClient := &dtclient.MockDynatraceClient{}
		mockDtcBuilder := &dynatraceclient.StubBuilder{
			DynatraceClient: mockClient,
		}

		controller := &Controller{
			client:                 fakeClient,
			apiReader:              fakeClient,
			dynatraceClientBuilder: mockDtcBuilder,
			registryClientBuilder:  createFakeRegistryClientBuilder(),
		}
		requiredScopes := token.Tokens{
			dtclient.DynatraceApiToken: {Value: testAPIToken},
		}.SetScopesForDynakube(*dynakube).ApiToken().RequiredScopes

		mockClient.On("GetTokenScopes", testAPIToken).Return(dtclient.TokenScopes(requiredScopes), nil)

		_ = controller.reconcileDynaKube(context.TODO(), dynakube)

		assertCondition(t, dynakube, dynatracev1beta1.TokenConditionType, metav1.ConditionTrue, dynatracev1beta1.ReasonTokenReady, "")
	})
}

func TestAPIError(t *testing.T) {
	mockClient := createDTMockClient(dtclient.TokenScopes{dtclient.TokenScopeInstallerDownload},
		dtclient.TokenScopes{dtclient.TokenScopeDataExport, dtclient.TokenScopeActiveGateTokenCreate},
	)
	instance := &dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testName,
			Namespace: testNamespace,
		},
		Spec: dynatracev1beta1.DynaKubeSpec{
			APIURL:   testApiUrl,
			OneAgent: dynatracev1beta1.OneAgentSpec{CloudNativeFullStack: &dynatracev1beta1.CloudNativeFullStackSpec{HostInjectSpec: dynatracev1beta1.HostInjectSpec{}}},
			ActiveGate: dynatracev1beta1.ActiveGateSpec{
				Capabilities: []dynatracev1beta1.CapabilityDisplayName{
					dynatracev1beta1.KubeMonCapability.DisplayName,
				},
			},
		},
	}
	t.Run("should return error result on 503", func(t *testing.T) {
		mockClient.On("GetActiveGateAuthToken", testName).Return(&dtclient.ActiveGateAuthTokenInfo{}, dtclient.ServerError{Code: http.StatusServiceUnavailable, Message: "Service unavailable"})
		controller := createFakeClientAndReconciler(mockClient, instance, testPaasToken, testAPIToken)

		result, err := controller.Reconcile(context.TODO(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: testName},
		})

		assert.NoError(t, err)
		assert.Equal(t, fastUpdateInterval, result.RequeueAfter)
	})
	t.Run("should return error result on 429", func(t *testing.T) {
		mockClient.On("GetActiveGateAuthToken", testName).Return(&dtclient.ActiveGateAuthTokenInfo{}, dtclient.ServerError{Code: http.StatusTooManyRequests, Message: "Too many requests"})
		controller := createFakeClientAndReconciler(mockClient, instance, testPaasToken, testAPIToken)

		result, err := controller.Reconcile(context.TODO(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: testName},
		})

		assert.NoError(t, err)
		assert.Equal(t, fastUpdateInterval, result.RequeueAfter)
	})
}

func TestSetupIstio(t *testing.T) {
	ctx := context.Background()
	dynakubeBase := &dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testName,
			Namespace: testNamespace,
		},
		Spec: dynatracev1beta1.DynaKubeSpec{
			APIURL:      testApiUrl,
			EnableIstio: true,
		},
	}
	t.Run("EnableIstio: false => do nothing + nil", func(t *testing.T) {
		dynakube := dynakubeBase.DeepCopy()
		dynakube.Spec.EnableIstio = false
		controller := &Controller{}
		istioReconciler, err := controller.setupIstio(ctx, dynakube)
		require.NoError(t, err)
		assert.Nil(t, istioReconciler)
	})
	t.Run("no istio installed + EnableIstio: true => error", func(t *testing.T) {
		dynakube := dynakubeBase.DeepCopy()
		fakeIstio := fakeistio.NewSimpleClientset()
		isIstioInstalled := false
		controller := &Controller{
			istioClientBuilder: fakeIstioClientBuilder(t, fakeIstio, isIstioInstalled),
			scheme:             scheme.Scheme,
		}
		istioReconciler, err := controller.setupIstio(ctx, dynakube)
		require.Error(t, err)
		assert.Nil(t, istioReconciler)
	})
	t.Run("success", func(t *testing.T) {
		dynakube := dynakubeBase.DeepCopy()
		fakeIstio := fakeistio.NewSimpleClientset()
		isIstioInstalled := true
		controller := &Controller{
			istioClientBuilder: fakeIstioClientBuilder(t, fakeIstio, isIstioInstalled),
			scheme:             scheme.Scheme,
		}
		istioReconciler, err := controller.setupIstio(ctx, dynakube)
		require.NoError(t, err)
		assert.NotNil(t, istioReconciler)

		expectedName := istio.BuildNameForFQDNServiceEntry(dynakube.GetName(), istio.OperatorComponent)
		serviceEntry, err := fakeIstio.NetworkingV1alpha3().ServiceEntries(dynakube.GetNamespace()).Get(ctx, expectedName, metav1.GetOptions{})
		require.NoError(t, err)
		assert.NotNil(t, serviceEntry)
		virtualService, err := fakeIstio.NetworkingV1alpha3().VirtualServices(dynakube.GetNamespace()).Get(ctx, expectedName, metav1.GetOptions{})
		require.NoError(t, err)
		assert.NotNil(t, virtualService)
	})
}

func fakeIstioClientBuilder(t *testing.T, fakeIstio *fakeistio.Clientset, isIstioInstalled bool) istio.ClientBuilder {
	return func(_ *rest.Config, scheme *runtime.Scheme, namespace string) (*istio.Client, error) {
		if isIstioInstalled == true {
			fakeDiscovery, ok := fakeIstio.Discovery().(*fakediscovery.FakeDiscovery)
			fakeDiscovery.Resources = []*metav1.APIResourceList{{GroupVersion: istio.IstioGVR}}
			if !ok {
				t.Fatalf("couldn't convert Discovery() to *FakeDiscovery")
			}
		}
		return &istio.Client{
			IstioClientset: fakeIstio,
			Scheme:         scheme,
			Namespace:      namespace,
		}, nil
	}
}

func assertCondition(t *testing.T, dk *dynatracev1beta1.DynaKube, expectedConditionType string, expectedConditionStatus metav1.ConditionStatus, expectedReason string, expectedMessage string) { //nolint:revive // argument-limit
	t.Helper()

	actualCondition := meta.FindStatusCondition(dk.Status.Conditions, expectedConditionType)
	require.NotNil(t, actualCondition)
	assert.Equal(t, expectedConditionStatus, actualCondition.Status)
	assert.Equal(t, expectedReason, actualCondition.Reason)
	assert.Equal(t, expectedMessage, actualCondition.Message)
}
