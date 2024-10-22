package dynakube

import (
	"context"
	"net/http"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/value"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube/activegate"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	ag "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/apimonitoring"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/deploymentmetadata"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/extension"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/injection"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/kspm"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/logmonitoring"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/proxy"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/labels"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubesystem"
	semVersion "github.com/Dynatrace/dynatrace-operator/pkg/version"
	dtclientmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace"
	dtbuildermock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/controllers/dynakube/dynatraceclient"
	"github.com/spf13/afero"
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

func TestReconcileActiveGate_Reconcile(t *testing.T) {
	t.Run(`Create works with minimal setup`, func(t *testing.T) {
		controller := &Controller{
			client:    fake.NewClient(),
			apiReader: fake.NewClient(),
		}
		result, err := controller.Reconcile(context.Background(), reconcile.Request{})

		require.NoError(t, err)
		assert.NotNil(t, result)
	})
	t.Run(`Create works with minimal setup and interface`, func(t *testing.T) { // keep as integration test?
		mockClient := createDTMockClient(t, dtclient.TokenScopes{dtclient.TokenScopeInstallerDownload}, dtclient.TokenScopes{dtclient.TokenScopeDataExport, dtclient.TokenScopeActiveGateTokenCreate})

		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			},
			Spec: dynakube.DynaKubeSpec{
				APIURL: testApiUrl,
			},
		}
		controller := createFakeClientAndReconciler(t, mockClient, dk, testPaasToken, testAPIToken)

		result, err := controller.Reconcile(context.Background(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: testName},
		})

		require.NoError(t, err)
		assert.NotNil(t, result)
	})
	t.Run(`Create reconciles proxy secret`, func(t *testing.T) {
		mockClient := createDTMockClient(t, dtclient.TokenScopes{dtclient.TokenScopeInstallerDownload}, dtclient.TokenScopes{dtclient.TokenScopeDataExport, dtclient.TokenScopeActiveGateTokenCreate})
		mockClient.On("GetActiveGateAuthToken", mock.AnythingOfType("context.backgroundCtx"), testName).Return(&dtclient.ActiveGateAuthTokenInfo{TokenId: "test", Token: "dt.some.valuegoeshere"}, nil)
		mockClient.On("GetLatestActiveGateVersion", mock.AnythingOfType("context.backgroundCtx"), mock.Anything).Return(testVersion, nil)

		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			},
			Spec: dynakube.DynaKubeSpec{
				APIURL: testApiUrl,
				Proxy: &value.Source{
					Value:     "https://proxy:1234",
					ValueFrom: "",
				},
				ActiveGate: activegate.Spec{Capabilities: []activegate.CapabilityDisplayName{activegate.KubeMonCapability.DisplayName}},
			},

			Status: *getTestDynkubeStatus(),
		}
		controller := createFakeClientAndReconciler(t, mockClient, dk, testPaasToken, testAPIToken)

		result, err := controller.Reconcile(context.Background(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: testName},
		})

		require.NoError(t, err)
		assert.NotNil(t, result)

		var proxySecret corev1.Secret

		name := proxy.BuildSecretName(testName)
		err = controller.client.Get(context.Background(), client.ObjectKey{Name: name, Namespace: testNamespace}, &proxySecret)

		require.NoError(t, err)
		assert.NotNil(t, proxySecret)
	})
	t.Run("reconciles phase change correctly", func(t *testing.T) {
		mockClient := createDTMockClient(t, dtclient.TokenScopes{dtclient.TokenScopeInstallerDownload}, dtclient.TokenScopes{dtclient.TokenScopeDataExport, dtclient.TokenScopeEntitiesRead, dtclient.TokenScopeSettingsRead, dtclient.TokenScopeSettingsWrite, dtclient.TokenScopeActiveGateTokenCreate})

		mockClient.On("GetActiveGateAuthToken", mock.AnythingOfType("context.backgroundCtx"), testName).Return(&dtclient.ActiveGateAuthTokenInfo{TokenId: "test", Token: "dt.some.valuegoeshere"}, nil)
		mockClient.On("GetLatestActiveGateVersion", mock.AnythingOfType("context.backgroundCtx"), mock.Anything).Return(testVersion, nil)

		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
				Annotations: map[string]string{
					dynakube.AnnotationFeatureAutomaticK8sApiMonitoring: "true",
				},
			},
			Spec: dynakube.DynaKubeSpec{
				APIURL: testApiUrl,
				ActiveGate: activegate.Spec{
					Capabilities: []activegate.CapabilityDisplayName{
						activegate.KubeMonCapability.DisplayName,
					},
				},
			},
			Status: *getTestDynkubeStatus(),
		}
		controller := createFakeClientAndReconciler(t, mockClient, dk, testPaasToken, testAPIToken)
		// Remove existing StatefulSet created by createFakeClientAndReconciler
		require.NoError(t, controller.client.Delete(context.Background(), &appsv1.StatefulSet{ObjectMeta: metav1.ObjectMeta{Name: testName + "-activegate", Namespace: testNamespace}}))

		result, err := controller.Reconcile(context.Background(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: testName},
		})

		mockClient.AssertCalled(t, "CreateOrUpdateKubernetesSetting",
			mock.AnythingOfType("context.backgroundCtx"),
			testName,
			testUID,
			mock.AnythingOfType("string"))

		require.NoError(t, err)
		assert.False(t, result.Requeue)

		var activeGateStatefulSet appsv1.StatefulSet

		require.NoError(t,
			controller.client.Get(context.Background(), client.ObjectKey{Name: testName + "-activegate", Namespace: testNamespace}, &activeGateStatefulSet))
		require.NoError(t, controller.client.Get(context.Background(), client.ObjectKey{Name: testName, Namespace: testNamespace}, dk))
		assert.Equal(t, status.Running, dk.Status.Phase)
	})
}

func TestReconcileOnlyOneTokenProvided_Reconcile(t *testing.T) {
	t.Run(`Create validates apiToken correctly if apiToken with "InstallerDownload"-scope is provided`, func(t *testing.T) {
		mockClient := createDTMockClient(t, dtclient.TokenScopes{}, dtclient.TokenScopes{dtclient.TokenScopeDataExport, dtclient.TokenScopeInstallerDownload, dtclient.TokenScopeActiveGateTokenCreate})

		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			},
			Spec: dynakube.DynaKubeSpec{
				APIURL: testApiUrl,
			}}
		controller := createFakeClientAndReconciler(t, mockClient, dk, "", testAPIToken)

		result, err := controller.Reconcile(context.Background(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: testName},
		})

		require.NoError(t, err)
		assert.NotNil(t, result)

		var secret corev1.Secret

		err = controller.client.Get(context.Background(), client.ObjectKey{Name: testName, Namespace: testNamespace}, &secret)

		require.NoError(t, err)
		assert.NotNil(t, secret)
		assert.Equal(t, testAPIToken, string(secret.Data[dtclient.ApiToken]))
	})
}
func TestReconcile_ActiveGateMultiCapability(t *testing.T) {
	mockClient := createDTMockClient(t, dtclient.TokenScopes{dtclient.TokenScopeInstallerDownload}, dtclient.TokenScopes{
		dtclient.TokenScopeDataExport,
		dtclient.TokenScopeMetricsIngest,
		dtclient.TokenScopeActiveGateTokenCreate,
	})

	mockClient.On("GetActiveGateAuthToken", mock.AnythingOfType("context.backgroundCtx"), testName).Return(&dtclient.ActiveGateAuthTokenInfo{TokenId: "test", Token: "dt.some.valuegoeshere"}, nil)
	mockClient.On("GetLatestActiveGateVersion", mock.AnythingOfType("context.backgroundCtx"), mock.Anything).Return(testVersion, nil)

	dk := &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testName,
			Namespace: testNamespace,
		},
		Spec: dynakube.DynaKubeSpec{
			APIURL: testApiUrl,
			ActiveGate: activegate.Spec{
				Capabilities: []activegate.CapabilityDisplayName{
					activegate.MetricsIngestCapability.DisplayName,
					activegate.KubeMonCapability.DisplayName,
					activegate.RoutingCapability.DisplayName,
				},
			},
		},
	}

	r := createFakeClientAndReconciler(t, mockClient, dk, testPaasToken, testAPIToken)
	request := reconcile.Request{
		NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: testName},
	}

	_, err := r.Reconcile(context.Background(), request)
	require.NoError(t, err)

	// Reconcile twice since routing service is created before the stateful set
	_, err = r.Reconcile(context.Background(), request)
	require.NoError(t, err)

	multiCapability := capability.NewMultiCapability(dk)
	stsName := capability.CalculateStatefulSetName(multiCapability, testName)

	routingSts := &appsv1.StatefulSet{}
	err = r.client.Get(context.Background(), client.ObjectKey{
		Namespace: testNamespace,
		Name:      stsName,
	}, routingSts)
	require.NoError(t, err)
	assert.NotNil(t, routingSts)

	routingSvc := &corev1.Service{}
	err = r.client.Get(context.Background(), client.ObjectKey{
		Namespace: testNamespace,
		Name:      testName + "-" + multiCapability.ShortName(),
	}, routingSvc)
	require.NoError(t, err)
	assert.NotNil(t, routingSvc)

	err = r.client.Get(context.Background(), client.ObjectKey{Name: dk.Name, Namespace: dk.Namespace}, dk)
	require.NoError(t, err)

	dk.Spec.ActiveGate.Capabilities = []activegate.CapabilityDisplayName{}
	err = r.client.Update(context.Background(), dk)
	require.NoError(t, err)

	_, err = r.Reconcile(context.Background(), request)
	require.NoError(t, err)

	err = r.client.Get(context.Background(), client.ObjectKey{
		Namespace: testNamespace,
		Name:      stsName,
	}, routingSts)
	require.Error(t, err)
	assert.True(t, k8serrors.IsNotFound(err))

	err = r.client.Get(context.Background(), client.ObjectKey{
		Namespace: testNamespace,
		Name:      testName + "-" + multiCapability.ShortName(),
	}, routingSvc)
	require.Error(t, err)
	assert.True(t, k8serrors.IsNotFound(err))
}

func TestAPIError(t *testing.T) {
	mockClient := createDTMockClient(t, dtclient.TokenScopes{dtclient.TokenScopeInstallerDownload}, dtclient.TokenScopes{dtclient.TokenScopeDataExport, dtclient.TokenScopeActiveGateTokenCreate})
	mockClient.On("GetLatestActiveGateVersion", mock.AnythingOfType("context.backgroundCtx"), mock.Anything).Return(testVersion, nil)

	dk := &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testName,
			Namespace: testNamespace,
		},
		Spec: dynakube.DynaKubeSpec{
			APIURL:   testApiUrl,
			OneAgent: dynakube.OneAgentSpec{CloudNativeFullStack: &dynakube.CloudNativeFullStackSpec{HostInjectSpec: dynakube.HostInjectSpec{}}},
			ActiveGate: activegate.Spec{
				Capabilities: []activegate.CapabilityDisplayName{
					activegate.KubeMonCapability.DisplayName,
				},
			},
		},
		Status: *getTestDynkubeStatus(),
	}

	t.Run("should return error result on 503", func(t *testing.T) {
		mockClient.On("GetActiveGateAuthToken", mock.AnythingOfType("context.backgroundCtx"), testName).Return(&dtclient.ActiveGateAuthTokenInfo{}, dtclient.ServerError{Code: http.StatusServiceUnavailable, Message: "Service unavailable"})
		controller := createFakeClientAndReconciler(t, mockClient, dk, testPaasToken, testAPIToken)

		result, err := controller.Reconcile(context.Background(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: testName},
		})

		require.NoError(t, err)
		assert.Equal(t, fastUpdateInterval, result.RequeueAfter)
	})
	t.Run("should return error result on 429", func(t *testing.T) {
		mockClient.On("GetActiveGateAuthToken", mock.AnythingOfType("context.backgroundCtx"), testName).Return(&dtclient.ActiveGateAuthTokenInfo{}, dtclient.ServerError{Code: http.StatusTooManyRequests, Message: "Too many requests"})
		controller := createFakeClientAndReconciler(t, mockClient, dk, testPaasToken, testAPIToken)

		result, err := controller.Reconcile(context.Background(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: testName},
		})

		require.NoError(t, err)
		assert.Equal(t, fastUpdateInterval, result.RequeueAfter)
	})
}

func createDTMockClient(t *testing.T, paasTokenScopes, apiTokenScopes dtclient.TokenScopes) *dtclientmock.Client {
	mockClient := dtclientmock.NewClient(t)

	mockClient.On("GetCommunicationHostForClient").Return(dtclient.CommunicationHost{
		Protocol: testProtocol,
		Host:     testHost,
		Port:     testPort,
	}, nil).Maybe()
	mockClient.On("GetOneAgentConnectionInfo", mock.AnythingOfType("context.backgroundCtx")).Return(dtclient.OneAgentConnectionInfo{
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
	}, nil).Maybe()
	mockClient.On("GetTokenScopes", mock.AnythingOfType("context.backgroundCtx"), testPaasToken).
		Return(paasTokenScopes, nil).Maybe()
	mockClient.On("GetTokenScopes", mock.AnythingOfType("context.backgroundCtx"), testAPIToken).
		Return(apiTokenScopes, nil).Maybe()
	mockClient.On("GetOneAgentConnectionInfo").
		Return(
			mock.AnythingOfType("context.backgroundCtx"),
			dtclient.OneAgentConnectionInfo{
				ConnectionInfo: dtclient.ConnectionInfo{
					TenantUUID: testUUID,
				},
			}, nil).Maybe()
	mockClient.On("GetLatestAgentVersion", mock.AnythingOfType("context.backgroundCtx"), mock.Anything, mock.Anything).
		Return(testVersion, nil).Maybe()
	mockClient.On("GetMonitoredEntitiesForKubeSystemUUID", mock.AnythingOfType("context.backgroundCtx"), mock.AnythingOfType("string")).
		Return([]dtclient.MonitoredEntity{{EntityId: "KUBERNETES_CLUSTER-0E30FE4BF2007587", DisplayName: "operator test entity 1", LastSeenTms: 1639483869085}}, nil).Maybe()
	mockClient.On("GetSettingsForMonitoredEntities", mock.AnythingOfType("context.backgroundCtx"), []dtclient.MonitoredEntity{{EntityId: "KUBERNETES_CLUSTER-0E30FE4BF2007587", DisplayName: "operator test entity 1", LastSeenTms: 1639483869085}}, mock.AnythingOfType("string")).
		Return(dtclient.GetSettingsResponse{}, nil).Maybe()
	mockClient.On("GetSettingsForMonitoredEntities", mock.AnythingOfType("context.backgroundCtx"), []dtclient.MonitoredEntity{{EntityId: "KUBERNETES_CLUSTER-0E30FE4BF2007587", DisplayName: "", LastSeenTms: 0}}, mock.AnythingOfType("string")).
		Return(dtclient.GetSettingsResponse{}, nil).Maybe()
	mockClient.On("CreateOrUpdateKubernetesSetting", mock.AnythingOfType("context.backgroundCtx"), testName, testUID, mock.AnythingOfType("string")).
		Return(testObjectID, nil).Maybe()
	mockClient.On("GetActiveGateConnectionInfo", mock.AnythingOfType("context.backgroundCtx")).
		Return(dtclient.ActiveGateConnectionInfo{
			ConnectionInfo: dtclient.ConnectionInfo{
				TenantUUID: testUUID,
			},
		}, nil).Maybe()
	mockClient.On("GetProcessModuleConfig", mock.AnythingOfType("context.backgroundCtx"), mock.AnythingOfType("uint")).
		Return(&dtclient.ProcessModuleConfig{}, nil).Maybe()

	return mockClient
}

func createFakeClientAndReconciler(t *testing.T, mockClient dtclient.Client, dk *dynakube.DynaKube, paasToken, apiToken string) *Controller {
	data := map[string][]byte{
		dtclient.ApiToken: []byte(apiToken),
	}
	if paasToken != "" {
		data[dtclient.PaasToken] = []byte(paasToken)
	}

	objects := []client.Object{
		dk,
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
	}

	objects = append(objects, generateStatefulSetForTesting(testName, testNamespace, "activegate", testUID))
	objects = append(objects, createTenantSecrets(dk)...)

	fakeClient := fake.NewClientWithIndex(objects...)

	mockDtcBuilder := dtbuildermock.NewBuilder(t)
	mockDtcBuilder.On("SetContext", mock.Anything).Return(mockDtcBuilder)
	mockDtcBuilder.On("SetDynakube", mock.Anything).Return(mockDtcBuilder)
	mockDtcBuilder.On("SetTokens", mock.Anything).Return(mockDtcBuilder)
	mockDtcBuilder.On("BuildWithTokenVerification", mock.Anything).Return(mockClient, nil)

	controller := &Controller{
		client:                              fakeClient,
		apiReader:                           fakeClient,
		dynatraceClientBuilder:              mockDtcBuilder,
		fs:                                  afero.Afero{Fs: afero.NewMemMapFs()},
		deploymentMetadataReconcilerBuilder: deploymentmetadata.NewReconciler,
		activeGateReconcilerBuilder:         ag.NewReconciler,
		apiMonitoringReconcilerBuilder:      apimonitoring.NewReconciler,
		injectionReconcilerBuilder:          injection.NewReconciler,
		oneAgentReconcilerBuilder:           oneagent.NewReconciler,
		logMonitoringReconcilerBuilder:      logmonitoring.NewReconciler,
		proxyReconcilerBuilder:              proxy.NewReconciler,
		extensionReconcilerBuilder:          extension.NewReconciler,
		kspmReconcilerBuilder:               kspm.NewReconciler,
		clusterID:                           testUID,
	}

	return controller
}

// generateStatefulSetForTesting prepares an ActiveGate StatefulSet after a Reconciliation of the Dynakube with a specific feature enabled
func generateStatefulSetForTesting(name, namespace, feature, kubeSystemUUID string) *appsv1.StatefulSet {
	expectedLabels := map[string]string{
		labels.AppNameLabel:      labels.ActiveGateComponentLabel,
		labels.AppVersionLabel:   testComponentVersion,
		labels.AppComponentLabel: feature,
		labels.AppCreatedByLabel: name,
		labels.AppManagedByLabel: semVersion.AppName,
	}
	expectedMatchLabels := map[string]string{
		labels.AppNameLabel:      labels.ActiveGateComponentLabel,
		labels.AppManagedByLabel: semVersion.AppName,
		labels.AppCreatedByLabel: name,
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
