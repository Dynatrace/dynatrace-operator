package dynakube

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/logmonitoring"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	ag "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/apimonitoring"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/deploymentmetadata"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/extension"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/injection"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/kspm"
	logmon "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/logmonitoring"
	oneagentcontroller "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/otelc"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/proxy"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/labels"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubesystem"
	semVersion "github.com/Dynatrace/dynatrace-operator/pkg/version"
	dtclientmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace"
	dtbuildermock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/controllers/dynakube/dynatraceclient"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/mock"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

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
	mockClient.On("GetK8sClusterME", mock.AnythingOfType("context.backgroundCtx"), mock.AnythingOfType("string")).
		Return(dtclient.K8sClusterME{ID: "KUBERNETES_CLUSTER-0E30FE4BF2007587", Name: "operator test entity 1"}, nil).Maybe()
	mockClient.On("GetSettingsForMonitoredEntity", mock.AnythingOfType("context.backgroundCtx"), dtclient.K8sClusterME{ID: "KUBERNETES_CLUSTER-0E30FE4BF2007587", Name: "operator test entity 1"}, mock.AnythingOfType("string")).
		Return(dtclient.GetSettingsResponse{}, nil).Maybe()
	mockClient.On("GetSettingsForMonitoredEntity", mock.AnythingOfType("context.backgroundCtx"), dtclient.K8sClusterME{ID: "KUBERNETES_CLUSTER-0E30FE4BF2007587", Name: ""}, mock.AnythingOfType("string")).
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
	mockClient.On("GetSettingsForLogModule", mock.AnythingOfType("context.backgroundCtx"), "KUBERNETES_CLUSTER-0E30FE4BF2007587").
		Return(dtclient.GetLogMonSettingsResponse{}, nil).Maybe()
	mockClient.On("CreateLogMonitoringSetting", mock.AnythingOfType("context.backgroundCtx"), "KUBERNETES_CLUSTER-0E30FE4BF2007587", "operator test entity 1", []logmonitoring.IngestRuleMatchers{}).
		Return(testObjectID, nil).Maybe()

	return mockClient
}

func createFakeClientAndReconciler(t *testing.T, mockClient dtclient.Client, dk *dynakube.DynaKube, paasToken, apiToken string) *Controller {
	data := map[string][]byte{
		dtclient.APIToken: []byte(apiToken),
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
	mockDtcBuilder.On("SetDynakube", mock.Anything).Return(mockDtcBuilder)
	mockDtcBuilder.On("SetTokens", mock.Anything).Return(mockDtcBuilder)
	mockDtcBuilder.On("Build", mock.Anything).Return(mockClient, nil)

	controller := &Controller{
		client:                              fakeClient,
		apiReader:                           fakeClient,
		dynatraceClientBuilder:              mockDtcBuilder,
		fs:                                  afero.Afero{Fs: afero.NewMemMapFs()},
		deploymentMetadataReconcilerBuilder: deploymentmetadata.NewReconciler,
		activeGateReconcilerBuilder:         ag.NewReconciler,
		apiMonitoringReconcilerBuilder:      apimonitoring.NewReconciler,
		injectionReconcilerBuilder:          injection.NewReconciler,
		oneAgentReconcilerBuilder:           oneagentcontroller.NewReconciler,
		logMonitoringReconcilerBuilder:      logmon.NewReconciler,
		proxyReconcilerBuilder:              proxy.NewReconciler,
		extensionReconcilerBuilder:          extension.NewReconciler,
		otelcReconcilerBuilder:              otelc.NewReconciler,
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
