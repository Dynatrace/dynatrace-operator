package pod_test

import (
	"context"
	"fmt"
	"maps"
	"net/url"
	"strings"
	"testing"

	podattr "github.com/Dynatrace/dynatrace-bootstrapper/cmd/configure/attributes/pod"
	"github.com/Dynatrace/dynatrace-operator/cmd/bootstrapper"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/metadataenrichment"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	otlpspec "github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/otlp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/communication"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	agconsts "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8senv"
	maputils "github.com/Dynatrace/dynatrace-operator/pkg/util/map"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	podmutation "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/handler/otlp"
	podmutator "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	metadatamutator "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator/metadata"
	oneagentmutator "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator/otlp/exporter"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator/otlp/resourceattributes"
	"github.com/Dynatrace/dynatrace-operator/test/integrationtests"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	admissionv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

const (
	testNamespace                = "dynatrace"
	testSecContextLabel          = "test-security-context-label"
	testCostCenterAnnotation     = "test-cost-center-annotation"
	testCustomMetadataLabel      = "test-custom-metadata-label"
	testCustomMetadataAnnotation = "test-custom-metadata-annotation"
	testMEID                     = "test-meid"
	testClusterName              = "test-cluster"
	testClusterUUID              = "123e4567-e89b-12d3-a456-426614174000"
	overrideNamespaceName        = "override-namespace"
)

var (
	podMetadataAnnotations = map[string]string{
		"metadata.dynatrace.com/service.name": "checkout service",
		"metadata.dynatrace.com/custom.key":   "value:with/special chars",
	}
	nsMetadataAnnotations = map[string]string{
		testCostCenterAnnotation:                "sales",
		testCustomMetadataAnnotation:            "custom-annotation",
		"metadata.dynatrace.com/custom.ns-meta": "custom-ns-meta-value",
	}
	nsMetadataLabels = map[string]string{
		testSecContextLabel:     "high",
		testCustomMetadataLabel: "custom-label",
	}
	metadataEnrichmentStatus = metadataenrichment.Status{
		Rules: []metadataenrichment.Rule{
			{
				Type:   "LABEL",
				Source: testSecContextLabel,
				Target: "dt.security_context",
			},
			{
				Type:   "LABEL",
				Source: testCustomMetadataLabel,
			},
			{
				Type:   "ANNOTATION",
				Source: testCostCenterAnnotation,
				Target: "dt.cost.costcenter",
			},
			{
				Type:   "ANNOTATION",
				Source: testCustomMetadataAnnotation,
			},
		},
	}
)

func buildArgument(attr string, value string) string {
	return fmt.Sprintf("--%s=%s=%s", podattr.Flag, attr, value)
}

func TestWebhook(t *testing.T) {
	clt := integrationtests.SetupWebhookTestEnvironment(t,
		getWebhookInstallOptions(),

		func(mgr ctrl.Manager) error {
			namespace := getNamespace(testNamespace)
			namespace.Annotations = nsMetadataAnnotations
			maps.Copy(namespace.Labels, nsMetadataLabels)
			require.NoError(t, mgr.GetClient().Create(t.Context(), namespace))

			dummyWebhookPod := getDummyWebhookPod()
			require.NoError(t, mgr.GetClient().Create(t.Context(), dummyWebhookPod))
			t.Setenv(k8senv.PodName, dummyWebhookPod.Name)

			return podmutation.AddWebhookToManager(t.Context(), mgr, testNamespace, false)
		},
	)

	// shared between test cases
	bootstrapperSecret := getBoostrapperSecret(testNamespace)
	createObject(t, clt, bootstrapperSecret)

	otlpExporterSecret := getOTLPExporterSecret(testNamespace)
	createObject(t, clt, otlpExporterSecret)

	t.Run("success incl. enrichment rules, custom metadata and metadata annotation propagation", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "dynakube",
				Namespace: testNamespace,
				Annotations: map[string]string{
					exp.InjectionAutomaticKey: "true",
				},
			},
			Spec: dynakube.DynaKubeSpec{
				OneAgent: oneagent.Spec{
					CloudNativeFullStack: &oneagent.CloudNativeFullStackSpec{},
				},
				MetadataEnrichment: metadataenrichment.Spec{
					Enabled: ptr.To(true),
				},
			},
			Status: dynakube.DynaKubeStatus{
				KubernetesClusterMEID: testMEID,
				KubernetesClusterName: testClusterName,
				MetadataEnrichment:    metadataEnrichmentStatus,
				KubeSystemUUID:        testClusterUUID,
				OneAgent: oneagent.Status{
					ConnectionInfoStatus: oneagent.ConnectionInfoStatus{
						ConnectionInfo: communication.ConnectionInfo{
							TenantUUID: uuid.NewString(),
						},
					},
				},
				CodeModules: oneagent.CodeModulesStatus{
					VersionStatus: status.VersionStatus{
						Version: "1.2.3",
					},
				},
			},
		}
		createDynaKube(t, clt, dk)

		dummyOwner, ownerReference := getDummyOwnerDeployment()
		createObject(t, clt, dummyOwner)
		pod := createPod(t, clt, func(pod *corev1.Pod) {
			pod.Annotations = podMetadataAnnotations
			pod.OwnerReferences = ownerReference
		})

		require.True(t, maputils.GetFieldBool(pod.Annotations, podmutator.AnnotationDynatraceInjected, false))
		require.True(t, maputils.GetFieldBool(pod.Annotations, metadatamutator.AnnotationInjected, false))
		require.True(t, maputils.GetFieldBool(pod.Annotations, oneagentmutator.AnnotationInjected, false))
		assert.Equal(t, "sales", maputils.GetField(pod.Annotations, "metadata.dynatrace.com/dt.cost.costcenter", ""))
		assert.Equal(t, "high", maputils.GetField(pod.Annotations, "metadata.dynatrace.com/dt.security_context", ""))
		assert.Equal(t, "custom-ns-meta-value", maputils.GetField(pod.Annotations, "metadata.dynatrace.com/custom.ns-meta", ""))
		require.Len(t, pod.Spec.InitContainers, 1)
		assert.Contains(t, pod.Spec.InitContainers[0].Args, buildArgument("k8s.workload.kind", strings.ToLower(pod.OwnerReferences[0].Kind)))
		assert.Contains(t, pod.Spec.InitContainers[0].Args, buildArgument("k8s.workload.name", strings.ToLower(pod.OwnerReferences[0].Name)))
		assert.Contains(t, pod.Spec.InitContainers[0].Args, buildArgument(metadatamutator.DeprecatedWorkloadKindKey, strings.ToLower(pod.OwnerReferences[0].Kind)))
		assert.Contains(t, pod.Spec.InitContainers[0].Args, buildArgument(metadatamutator.DeprecatedWorkloadNameKey, strings.ToLower(pod.OwnerReferences[0].Name)))
		assert.Contains(t, pod.Spec.InitContainers[0].Args, buildArgument("custom.ns-meta", "custom-ns-meta-value"))
		assert.Contains(t, pod.Spec.InitContainers[0].Args, buildArgument("dt.security_context", "high"))
		assert.Contains(t, pod.Spec.InitContainers[0].Args, buildArgument("dt.cost.costcenter", "sales"))
		assert.Contains(t, pod.Spec.InitContainers[0].Args, buildArgument("k8s.namespace.label."+testCustomMetadataLabel, "custom-label"))
		assert.Contains(t, pod.Spec.InitContainers[0].Args, buildArgument("k8s.namespace.annotation."+testCustomMetadataAnnotation, "custom-annotation"))
		assert.Contains(t, pod.Spec.InitContainers[0].Args, "--"+bootstrapper.MetadataEnrichmentFlag)
		assert.Contains(t, pod.Spec.InitContainers[0].Args, buildArgument("k8s.pod.uid", "$(K8S_PODUID)"))
		assert.Contains(t, pod.Spec.InitContainers[0].Args, buildArgument("k8s.pod.name", "$(K8S_PODNAME)"))
		assert.Contains(t, pod.Spec.InitContainers[0].Args, buildArgument("k8s.node.name", "$(K8S_NODE_NAME)"))
		assert.Contains(t, pod.Spec.InitContainers[0].Args, buildArgument("k8s.namespace.name", pod.Namespace))
		assert.Contains(t, pod.Spec.InitContainers[0].Args, buildArgument("k8s.cluster.uid", testClusterUUID))
		assert.Contains(t, pod.Spec.InitContainers[0].Args, buildArgument("k8s.cluster.name", testClusterName))
		assert.Contains(t, pod.Spec.InitContainers[0].Args, buildArgument("dt.entity.kubernetes_cluster", testMEID))
		assert.Contains(t, pod.Spec.InitContainers[0].Args, buildArgument("dt.kubernetes.cluster.id", testClusterUUID))
		assert.Contains(t, pod.Spec.InitContainers[0].Args, "--attribute-container={\"container_image.registry\":\"docker.io\",\"container_image.repository\":\"myapp\",\"container_image.tags\":\"1.2.3\",\"k8s.container.name\":\"app\"}")
	})

	t.Run("success with proper precedence", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "dynakube",
				Namespace: testNamespace,
				Annotations: map[string]string{
					exp.InjectionAutomaticKey: "true",
				},
			},
			Spec: dynakube.DynaKubeSpec{
				OneAgent: oneagent.Spec{
					CloudNativeFullStack: &oneagent.CloudNativeFullStackSpec{},
				},
				MetadataEnrichment: metadataenrichment.Spec{
					Enabled: ptr.To(true),
				},
			},
			Status: dynakube.DynaKubeStatus{
				KubernetesClusterMEID: testMEID,
				KubernetesClusterName: testClusterName,
				MetadataEnrichment:    metadataEnrichmentStatus,
				KubeSystemUUID:        testClusterUUID,
				OneAgent: oneagent.Status{
					ConnectionInfoStatus: oneagent.ConnectionInfoStatus{
						ConnectionInfo: communication.ConnectionInfo{
							TenantUUID: uuid.NewString(),
						},
					},
				},
				CodeModules: oneagent.CodeModulesStatus{
					VersionStatus: status.VersionStatus{
						Version: "1.2.3",
					},
				},
			},
		}
		createDynaKube(t, clt, dk)

		overrideNamespace := getNamespace(overrideNamespaceName)
		overrideNamespace.Name = overrideNamespaceName
		overrideNamespace.Annotations = map[string]string{
			"metadata.dynatrace.com/dt.entity.kubernetes_cluster": "ns-meid",
			"metadata.dynatrace.com/k8s.cluster.name":             "override-cluster-name",
		}

		createObject(t, clt, overrideNamespace)
		createObject(t, clt, getBoostrapperSecret(overrideNamespaceName))
		createObject(t, clt, getOTLPExporterSecret(overrideNamespaceName))

		pod := createPod(t, clt, func(pod *corev1.Pod) {
			pod.Namespace = overrideNamespaceName
			maps.Copy(pod.Annotations, map[string]string{
				"metadata.dynatrace.com/dt.entity.kubernetes_cluster": "pod-meid",
				"metadata.dynatrace.com/k8s.pod.name":                 "override-pod-name",
			})
		})

		require.True(t, maputils.GetFieldBool(pod.Annotations, podmutator.AnnotationDynatraceInjected, false))
		require.True(t, maputils.GetFieldBool(pod.Annotations, metadatamutator.AnnotationInjected, false))
		require.True(t, maputils.GetFieldBool(pod.Annotations, oneagentmutator.AnnotationInjected, false))

		// verify precedence
		require.Len(t, pod.Spec.InitContainers, 1)
		assert.Contains(t, pod.Spec.InitContainers[0].Args, buildArgument("k8s.pod.name", "override-pod-name"))
		assert.Contains(t, pod.Spec.InitContainers[0].Args, buildArgument("dt.entity.kubernetes_cluster", "pod-meid"))
		assert.Contains(t, pod.Spec.InitContainers[0].Args, buildArgument("k8s.cluster.name", "override-cluster-name"))
	})

	t.Run("oneagent mutator failure", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "dynakube",
				Namespace: testNamespace,
			},
			Spec: dynakube.DynaKubeSpec{
				OneAgent: oneagent.Spec{
					CloudNativeFullStack: &oneagent.CloudNativeFullStackSpec{},
				},
			},
			Status: dynakube.DynaKubeStatus{
				CodeModules: oneagent.CodeModulesStatus{
					VersionStatus: status.VersionStatus{
						Version: "1.2.3",
					},
				},
			},
		}
		createDynaKube(t, clt, dk)

		pod := createPod(t, clt, func(pod *corev1.Pod) {
			pod.Annotations[oneagentmutator.AnnotationInject] = "true"
		})

		assert.Contains(t, pod.Annotations, oneagentmutator.AnnotationReason)
	})

	t.Run("oneagent mutator failure -> status not ready", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "dynakube",
				Namespace: testNamespace,
			},
			Spec: dynakube.DynaKubeSpec{
				OneAgent: oneagent.Spec{
					CloudNativeFullStack: &oneagent.CloudNativeFullStackSpec{},
				},
			},
			Status: dynakube.DynaKubeStatus{
				CodeModules: oneagent.CodeModulesStatus{
					VersionStatus: status.VersionStatus{},
				},
			},
		}
		createDynaKube(t, clt, dk)

		pod := createPod(t, clt, func(pod *corev1.Pod) {
			pod.Annotations[oneagentmutator.AnnotationInject] = "true"
		})

		require.Contains(t, pod.Annotations, oneagentmutator.AnnotationReason)
		assert.Contains(t, pod.Annotations[oneagentmutator.AnnotationReason], oneagentmutator.DynaKubeStatusNotReadyReason)
	})

	t.Run("metadata mutator failure", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "dynakube",
				Namespace: testNamespace,
			},
			Spec: dynakube.DynaKubeSpec{
				MetadataEnrichment: metadataenrichment.Spec{
					Enabled: ptr.To(true),
				},
			},
		}
		createDynaKube(t, clt, dk)

		pod := createPod(t, clt, func(pod *corev1.Pod) {
			pod.Annotations[metadatamutator.AnnotationInject] = "true"
			pod.OwnerReferences = []metav1.OwnerReference{
				{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
					Name:       "missing",
					UID:        types.UID(uuid.NewString()),
					Controller: ptr.To(true),
				},
			}
		})

		assert.Contains(t, pod.Annotations, metadatamutator.AnnotationReason)
	})
}

func TestOTLPWebhook(t *testing.T) {
	clt := integrationtests.SetupWebhookTestEnvironment(t,
		getWebhookInstallOptions(),

		func(mgr ctrl.Manager) error {
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: testNamespace,
					Labels: map[string]string{
						podmutator.InjectionInstanceLabel: "dynakube",
					},
					Annotations: nsMetadataAnnotations,
				},
			}
			maps.Copy(ns.Labels, nsMetadataLabels)
			require.NoError(t, mgr.GetClient().Create(t.Context(), ns))

			dummyWebhookPod := getDummyWebhookPod()
			require.NoError(t, mgr.GetClient().Create(t.Context(), dummyWebhookPod))
			t.Setenv(k8senv.PodName, dummyWebhookPod.Name)

			return podmutation.AddWebhookToManager(t.Context(), mgr, testNamespace, false)
		},
	)

	t.Run("otlp exporter with ns metadata propagation and custom enrichment rules", func(t *testing.T) {
		apiURL := "https://example.live.dynatrace.com"
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "dynakube",
				Namespace: testNamespace,
				Annotations: map[string]string{
					exp.InjectionAutomaticKey: "true",
				},
			},
			Spec: dynakube.DynaKubeSpec{
				APIURL: apiURL,
				OTLPExporterConfiguration: &otlpspec.ExporterConfigurationSpec{
					NamespaceSelector: metav1.LabelSelector{ // match test namespace label applied earlier
						MatchExpressions: []metav1.LabelSelectorRequirement{
							{Key: podmutator.InjectionInstanceLabel, Operator: metav1.LabelSelectorOpExists},
						},
					},
					Signals: otlpspec.SignalConfiguration{
						Metrics: &otlpspec.MetricsSignal{},
						Logs:    &otlpspec.LogsSignal{},
						Traces:  &otlpspec.TracesSignal{},
					},
				},
			},
			Status: dynakube.DynaKubeStatus{
				KubernetesClusterMEID: testMEID,
				KubernetesClusterName: testClusterName,
				MetadataEnrichment:    metadataEnrichmentStatus,
			},
		}

		apiTokenSecret := getOTLPExporterSecret(testNamespace)
		createObject(t, clt, apiTokenSecret)

		createDynaKube(t, clt, dk)

		dummyOwner, ownerReference := getDummyOwnerDeployment()
		createObject(t, clt, dummyOwner)
		pod := createPod(t, clt, func(pod *corev1.Pod) {
			pod.Annotations = podMetadataAnnotations
			pod.OwnerReferences = ownerReference
		})

		// verify mutation occurred by presence of OTLP env vars (annotation may not be set when no OneAgent injection)

		appContainer := pod.Spec.Containers[0]
		// Expect DT_API_TOKEN env var via secret ref
		var dtTokenEnv *corev1.EnvVar
		for i := range appContainer.Env {
			if appContainer.Env[i].Name == exporter.DynatraceAPITokenEnv {
				dtTokenEnv = &appContainer.Env[i]

				break
			}
		}

		require.NotNil(t, dtTokenEnv, "expected DT_API_TOKEN env var to be injected")
		require.NotNil(t, dtTokenEnv.ValueFrom)
		require.NotNil(t, dtTokenEnv.ValueFrom.SecretKeyRef)
		assert.Equal(t, consts.OTLPExporterSecretName, dtTokenEnv.ValueFrom.SecretKeyRef.Name)
		assert.Equal(t, dynatrace.DataIngestToken, dtTokenEnv.ValueFrom.SecretKeyRef.Key)

		// Headers env vars should reference DT_API_TOKEN via authorization header literal
		assert.Contains(t, appContainer.Env, corev1.EnvVar{Name: exporter.OTLPMetricsHeadersEnv, Value: exporter.OTLPAuthorizationHeader})
		assert.Contains(t, appContainer.Env, corev1.EnvVar{Name: exporter.OTLPLogsHeadersEnv, Value: exporter.OTLPAuthorizationHeader})
		assert.Contains(t, appContainer.Env, corev1.EnvVar{Name: exporter.OTLPTraceHeadersEnv, Value: exporter.OTLPAuthorizationHeader})

		// Endpoint base constructed by BuildOTLPEndpoint(apiURL) => apiURL + /v2/otlp plus per-signal suffix
		baseEndpoint := apiURL + "/v2/otlp"
		assert.Contains(t, appContainer.Env, corev1.EnvVar{Name: exporter.OTLPMetricsEndpointEnv, Value: baseEndpoint + "/v1/metrics"})
		assert.Contains(t, appContainer.Env, corev1.EnvVar{Name: exporter.OTLPLogsEndpointEnv, Value: baseEndpoint + "/v1/logs"})
		assert.Contains(t, appContainer.Env, corev1.EnvVar{Name: exporter.OTLPTraceEndpointEnv, Value: baseEndpoint + "/v1/traces"})

		// metrics temporality preference should be set to delta
		assert.Contains(t, appContainer.Env, corev1.EnvVar{Name: exporter.OTLPMetricsExporterTemporalityPreference, Value: exporter.OTLPMetricsExporterAggregationTemporalityDelta})

		raEnv := k8senv.Find(appContainer.Env, resourceattributes.OTELResourceAttributesEnv)

		require.NotNil(t, raEnv, "OTEL_RESOURCE_ATTRIBUTES missing")

		gotResourceAttributes, envVarFound := resourceattributes.NewAttributesFromEnv(appContainer.Env, resourceattributes.OTELResourceAttributesEnv)
		require.True(t, envVarFound, "OTEL_RESOURCE_ATTRIBUTES missing")

		assert.Equal(t, testNamespace, gotResourceAttributes["k8s.namespace.name"])
		assert.Equal(t, "$(K8S_PODUID)", gotResourceAttributes["k8s.pod.uid"])
		assert.Equal(t, "$(K8S_PODNAME)", gotResourceAttributes["k8s.pod.name"])
		assert.Equal(t, "$(K8S_NODE_NAME)", gotResourceAttributes["k8s.node.name"])
		assert.Contains(t, appContainer.Env, corev1.EnvVar{
			Name: "K8S_PODUID",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					APIVersion: "v1",
					FieldPath:  "metadata.uid",
				},
			},
		})
		assert.Contains(t, appContainer.Env, corev1.EnvVar{
			Name: "K8S_PODNAME",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					APIVersion: "v1",
					FieldPath:  "metadata.name",
				},
			},
		})
		assert.Contains(t, appContainer.Env, corev1.EnvVar{
			Name: "K8S_NODE_NAME",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					APIVersion: "v1",
					FieldPath:  "spec.nodeName",
				},
			},
		})
		assert.Contains(t, appContainer.Env, corev1.EnvVar{Name: exporter.OTLPLogsEndpointEnv, Value: baseEndpoint + "/v1/logs"})
		assert.Contains(t, appContainer.Env, corev1.EnvVar{Name: exporter.OTLPTraceEndpointEnv, Value: baseEndpoint + "/v1/traces"})
		assert.Equal(t, dk.Status.KubernetesClusterName, gotResourceAttributes["k8s.cluster.name"])
		assert.Equal(t, pod.Spec.Containers[0].Name, gotResourceAttributes["k8s.container.name"])
		assert.Equal(t, dk.Status.KubeSystemUUID, gotResourceAttributes["dt.kubernetes.cluster.id"])
		assert.Equal(t, dk.Status.KubernetesClusterMEID, gotResourceAttributes["dt.entity.kubernetes_cluster"])
		assert.Equal(t, pod.OwnerReferences[0].Name, gotResourceAttributes["k8s.workload.name"])
		assert.Equal(t, pod.OwnerReferences[0].Name, gotResourceAttributes[metadatamutator.DeprecatedWorkloadNameKey])
		assert.Equal(t, strings.ToLower(pod.OwnerReferences[0].Kind), gotResourceAttributes["k8s.workload.kind"])
		assert.Equal(t, strings.ToLower(pod.OwnerReferences[0].Kind), gotResourceAttributes[metadatamutator.DeprecatedWorkloadKindKey])
		assert.Equal(t, url.QueryEscape(nsMetadataAnnotations["metadata.dynatrace.com/custom.ns-meta"]), gotResourceAttributes["custom.ns-meta"])
		assert.Equal(t, url.QueryEscape(podMetadataAnnotations["metadata.dynatrace.com/service.name"]), gotResourceAttributes["service.name"])
		assert.Equal(t, url.QueryEscape(podMetadataAnnotations["metadata.dynatrace.com/custom.key"]), gotResourceAttributes["custom.key"])
		assert.Equal(t, url.QueryEscape(nsMetadataAnnotations[testCustomMetadataAnnotation]), gotResourceAttributes["k8s.namespace.annotation."+testCustomMetadataAnnotation])
		assert.Equal(t, url.QueryEscape(nsMetadataAnnotations[testCostCenterAnnotation]), gotResourceAttributes["dt.cost.costcenter"])
		assert.Equal(t, url.QueryEscape(nsMetadataLabels[testSecContextLabel]), gotResourceAttributes["dt.security_context"])
		assert.Equal(t, url.QueryEscape(nsMetadataLabels[testCustomMetadataLabel]), gotResourceAttributes["k8s.namespace.label."+testCustomMetadataLabel])
	})

	t.Run("otlp exporter attribute precedence", func(t *testing.T) {
		apiURL := "https://example.live.dynatrace.com"
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "dynakube",
				Namespace: testNamespace,
				Annotations: map[string]string{
					exp.InjectionAutomaticKey: "true",
				},
			},
			Spec: dynakube.DynaKubeSpec{
				APIURL: apiURL,
				OTLPExporterConfiguration: &otlpspec.ExporterConfigurationSpec{
					NamespaceSelector: metav1.LabelSelector{ // match test namespace label applied earlier
						MatchExpressions: []metav1.LabelSelectorRequirement{
							{Key: podmutator.InjectionInstanceLabel, Operator: metav1.LabelSelectorOpExists},
						},
					},
					Signals: otlpspec.SignalConfiguration{
						Metrics: &otlpspec.MetricsSignal{},
						Logs:    &otlpspec.LogsSignal{},
						Traces:  &otlpspec.TracesSignal{},
					},
				},
			},
			Status: dynakube.DynaKubeStatus{
				KubernetesClusterMEID: testMEID,
				KubernetesClusterName: testClusterName,
				MetadataEnrichment:    metadataEnrichmentStatus,
			},
		}

		createDynaKube(t, clt, dk)

		overrideNamespace := getNamespace(overrideNamespaceName)
		overrideNamespace.Annotations = map[string]string{
			"metadata.dynatrace.com/dt.entity.kubernetes_cluster": "ns-meid",
			"metadata.dynatrace.com/k8s.cluster.name":             "override-cluster-name",
		}
		createObject(t, clt, overrideNamespace)

		apiTokenSecret := getOTLPExporterSecret(overrideNamespaceName)
		createObject(t, clt, apiTokenSecret)

		pod := createPod(t, clt, func(pod *corev1.Pod) {
			pod.Namespace = overrideNamespaceName
			pod.Annotations = map[string]string{
				"metadata.dynatrace.com/dt.entity.kubernetes_cluster": "pod-meid",
				"metadata.dynatrace.com/k8s.pod.name":                 "override-pod-name",
			}
		})

		// verify mutation occurred by presence of OTLP env vars (annotation may not be set when no OneAgent injection)

		appContainer := pod.Spec.Containers[0]
		// Expect DT_API_TOKEN env var via secret ref
		dtTokenEnv := k8senv.Find(appContainer.Env, exporter.DynatraceAPITokenEnv)

		require.NotNil(t, dtTokenEnv, "expected DT_API_TOKEN env var to be injected")
		require.NotNil(t, dtTokenEnv.ValueFrom)
		require.NotNil(t, dtTokenEnv.ValueFrom.SecretKeyRef)
		assert.Equal(t, consts.OTLPExporterSecretName, dtTokenEnv.ValueFrom.SecretKeyRef.Name)
		assert.Equal(t, dynatrace.DataIngestToken, dtTokenEnv.ValueFrom.SecretKeyRef.Key)

		raEnv := k8senv.Find(appContainer.Env, resourceattributes.OTELResourceAttributesEnv)
		require.NotNil(t, raEnv, "OTEL_RESOURCE_ATTRIBUTES missing")

		gotResourceAttributes, envVarFound := resourceattributes.NewAttributesFromEnv(appContainer.Env, resourceattributes.OTELResourceAttributesEnv)
		require.True(t, envVarFound, "OTEL_RESOURCE_ATTRIBUTES missing")

		assert.Equal(t, overrideNamespaceName, gotResourceAttributes["k8s.namespace.name"])
		assert.Equal(t, "override-pod-name", gotResourceAttributes["k8s.pod.name"])
		assert.Equal(t, "override-cluster-name", gotResourceAttributes["k8s.cluster.name"])
		assert.Equal(t, "pod-meid", gotResourceAttributes["dt.entity.kubernetes_cluster"])
	})

	t.Run("data ingest token secret missing", func(t *testing.T) {
		apiURL := "https://example.live.dynatrace.com"
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "dynakube",
				Namespace: testNamespace,
				Annotations: map[string]string{
					exp.InjectionAutomaticKey: "true",
				},
			},
			Spec: dynakube.DynaKubeSpec{
				APIURL: apiURL,
				OTLPExporterConfiguration: &otlpspec.ExporterConfigurationSpec{
					NamespaceSelector: metav1.LabelSelector{ // match test namespace label applied earlier
						MatchExpressions: []metav1.LabelSelectorRequirement{
							{Key: podmutator.InjectionInstanceLabel, Operator: metav1.LabelSelectorOpExists},
						},
					},
					Signals: otlpspec.SignalConfiguration{
						Metrics: &otlpspec.MetricsSignal{},
						Logs:    &otlpspec.LogsSignal{},
						Traces:  &otlpspec.TracesSignal{},
					},
				},
			},
		}

		createDynaKube(t, clt, dk)

		pod := createPod(t, clt, nil)

		assert.False(t, maputils.GetFieldBool(pod.Annotations, podmutator.AnnotationOTLPInjected, false))
		assert.Equal(t, otlp.NoOTLPExporterConfigSecretReason, pod.Annotations[podmutator.AnnotationOTLPReason])
	})

	t.Run("otlp exporter activegate", func(t *testing.T) {
		const dataIngestToken = "test-token"
		const agCertData = "ag-cert-data"

		apiURL := "https://example.live.dynatrace.com"
		tenantUUID := uuid.NewString()

		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "dynakube",
				Namespace: testNamespace,
				Annotations: map[string]string{
					exp.InjectionAutomaticKey: "true",
				},
			},
			Spec: dynakube.DynaKubeSpec{
				APIURL: apiURL,
				ActiveGate: activegate.Spec{
					Capabilities: []activegate.CapabilityDisplayName{
						activegate.RoutingCapability.DisplayName,
					},
				},
				OTLPExporterConfiguration: &otlpspec.ExporterConfigurationSpec{
					NamespaceSelector: metav1.LabelSelector{
						MatchExpressions: []metav1.LabelSelectorRequirement{
							{Key: podmutator.InjectionInstanceLabel, Operator: metav1.LabelSelectorOpExists},
						},
					},
					Signals: otlpspec.SignalConfiguration{
						Metrics: &otlpspec.MetricsSignal{},
						Logs:    &otlpspec.LogsSignal{},
						Traces:  &otlpspec.TracesSignal{},
					},
				},
			},
			Status: dynakube.DynaKubeStatus{
				OneAgent: oneagent.Status{
					ConnectionInfoStatus: oneagent.ConnectionInfoStatus{
						ConnectionInfo: communication.ConnectionInfo{
							TenantUUID: tenantUUID,
						},
					},
				},
			},
		}

		apiTokenSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      consts.OTLPExporterSecretName,
				Namespace: testNamespace,
			},
			Data: map[string][]byte{
				dynatrace.APIToken:        []byte(dataIngestToken),
				dynatrace.DataIngestToken: []byte(dataIngestToken),
			},
		}
		createObject(t, clt, apiTokenSecret)

		agCertSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      consts.OTLPExporterCertsSecretName,
				Namespace: testNamespace,
			},
			Data: map[string][]byte{
				dynakube.TLSCertKey: []byte(agCertData),
			},
		}
		createObject(t, clt, agCertSecret)

		createDynaKube(t, clt, dk)

		pod := createPod(t, clt, nil)
		appContainer := pod.Spec.Containers[0]

		envMap := map[string]corev1.EnvVar{}
		for _, e := range appContainer.Env {
			envMap[e.Name] = e
		}

		dtTokenEnv, ok := envMap[exporter.DynatraceAPITokenEnv]
		require.True(t, ok, "DT_API_TOKEN missing")
		require.NotNil(t, dtTokenEnv.ValueFrom)
		require.NotNil(t, dtTokenEnv.ValueFrom.SecretKeyRef)
		assert.Equal(t, consts.OTLPExporterSecretName, dtTokenEnv.ValueFrom.SecretKeyRef.Name)
		assert.Equal(t, dynatrace.DataIngestToken, dtTokenEnv.ValueFrom.SecretKeyRef.Key)

		expectedService := fmt.Sprintf("%s-%s.%s", dk.Name, agconsts.MultiActiveGateName, testNamespace)
		expectedBase := fmt.Sprintf("https://%s/e/%s/api/v2/otlp", expectedService, tenantUUID)

		assert.Equal(t, expectedBase+"/v1/metrics", envMap[exporter.OTLPMetricsEndpointEnv].Value)
		assert.Equal(t, expectedBase+"/v1/logs", envMap[exporter.OTLPLogsEndpointEnv].Value)
		assert.Equal(t, expectedBase+"/v1/traces", envMap[exporter.OTLPTraceEndpointEnv].Value)
	})

	t.Run("otlp exporter activegate - certificate secret missing", func(t *testing.T) {
		const dataIngestToken = "test-token"

		apiURL := "https://example.live.dynatrace.com"
		tenantUUID := uuid.NewString()

		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "dynakube",
				Namespace: testNamespace,
				Annotations: map[string]string{
					exp.InjectionAutomaticKey: "true",
				},
			},
			Spec: dynakube.DynaKubeSpec{
				APIURL: apiURL,
				ActiveGate: activegate.Spec{
					Capabilities: []activegate.CapabilityDisplayName{
						activegate.RoutingCapability.DisplayName,
					},
				},
				OTLPExporterConfiguration: &otlpspec.ExporterConfigurationSpec{
					NamespaceSelector: metav1.LabelSelector{
						MatchExpressions: []metav1.LabelSelectorRequirement{
							{Key: podmutator.InjectionInstanceLabel, Operator: metav1.LabelSelectorOpExists},
						},
					},
					Signals: otlpspec.SignalConfiguration{
						Metrics: &otlpspec.MetricsSignal{},
						Logs:    &otlpspec.LogsSignal{},
						Traces:  &otlpspec.TracesSignal{},
					},
				},
			},
			Status: dynakube.DynaKubeStatus{
				OneAgent: oneagent.Status{
					ConnectionInfoStatus: oneagent.ConnectionInfoStatus{
						ConnectionInfo: communication.ConnectionInfo{
							TenantUUID: tenantUUID,
						},
					},
				},
			},
		}

		apiTokenSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      consts.OTLPExporterSecretName,
				Namespace: testNamespace,
			},
			Data: map[string][]byte{
				dynatrace.APIToken:        []byte(dataIngestToken),
				dynatrace.DataIngestToken: []byte(dataIngestToken),
			},
		}
		createObject(t, clt, apiTokenSecret)

		createDynaKube(t, clt, dk)

		pod := createPod(t, clt, nil)

		assert.False(t, maputils.GetFieldBool(pod.Annotations, podmutator.AnnotationOTLPInjected, false))
		assert.Equal(t, otlp.NoOTLPExporterActiveGateCertSecretReason, pod.Annotations[podmutator.AnnotationOTLPReason])
	})
}

func getWebhookInstallOptions() envtest.WebhookInstallOptions {
	return envtest.WebhookInstallOptions{
		MutatingWebhooks: []*admissionv1.MutatingWebhookConfiguration{
			// TODO(avorima): Load this from a file using Paths
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "dynatrace-webhook",
				},
				Webhooks: []admissionv1.MutatingWebhook{
					{
						Name:               "webhook.pod.dynatrace.com",
						ReinvocationPolicy: ptr.To(admissionv1.IfNeededReinvocationPolicy),
						FailurePolicy:      ptr.To(admissionv1.Ignore),
						TimeoutSeconds:     ptr.To[int32](30),
						Rules: []admissionv1.RuleWithOperations{
							{
								Rule: admissionv1.Rule{
									APIGroups:   []string{""},
									APIVersions: []string{"v1"},
									Resources:   []string{"pods"},
									Scope:       ptr.To(admissionv1.NamespacedScope),
								},
								Operations: []admissionv1.OperationType{
									admissionv1.Create,
								},
							},
						},
						NamespaceSelector: &metav1.LabelSelector{
							MatchExpressions: []metav1.LabelSelectorRequirement{
								{
									Key:      podmutator.InjectionInstanceLabel,
									Operator: metav1.LabelSelectorOpExists,
								},
							},
						},
						ClientConfig: admissionv1.WebhookClientConfig{
							Service: &admissionv1.ServiceReference{
								Name: "dynatrace-webhook",
								Path: ptr.To("/inject"),
							},
						},
						AdmissionReviewVersions: []string{"v1beta1", "v1"},
						SideEffects:             ptr.To(admissionv1.SideEffectClassNone),
					},
				},
			},
		},
	}
}

func setupOTLPWebhookEnv(t *testing.T) client.Client {
	t.Helper()

	return integrationtests.SetupWebhookTestEnvironment(t,
		getWebhookInstallOptions(),

		func(mgr ctrl.Manager) error {
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: testNamespace,
					Labels: map[string]string{
						podmutator.InjectionInstanceLabel: "dynakube",
					},
					Annotations: nsMetadataAnnotations,
				},
			}
			maps.Copy(ns.Labels, nsMetadataLabels)
			require.NoError(t, mgr.GetClient().Create(t.Context(), ns))

			dummyWebhookPod := getDummyWebhookPod()
			require.NoError(t, mgr.GetClient().Create(t.Context(), dummyWebhookPod))
			t.Setenv(k8senv.PodName, dummyWebhookPod.Name)

			return podmutation.AddWebhookToManager(t.Context(), mgr, testNamespace, false)
		},
	)
}

func TestOTLPExporterSkipWhenGeneralOTELPreset(t *testing.T) {
	clt := setupOTLPWebhookEnv(t)

	apiURL := "https://example.live.dynatrace.com"
	dk := &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dynakube",
			Namespace: testNamespace,
			Annotations: map[string]string{
				exp.InjectionAutomaticKey: "true",
			},
		},
		Spec: dynakube.DynaKubeSpec{
			APIURL: apiURL,
			OTLPExporterConfiguration: &otlpspec.ExporterConfigurationSpec{
				NamespaceSelector: metav1.LabelSelector{
					MatchExpressions: []metav1.LabelSelectorRequirement{
						{Key: podmutator.InjectionInstanceLabel, Operator: metav1.LabelSelectorOpExists},
					},
				},
				Signals: otlpspec.SignalConfiguration{
					Metrics: &otlpspec.MetricsSignal{},
					Logs:    &otlpspec.LogsSignal{},
					Traces:  &otlpspec.TracesSignal{},
				},
			},
		},
	}

	createDynaKube(t, clt, dk)

	pod := createPod(t, clt, func(p *corev1.Pod) {
		p.Spec.Containers[0].Env = append(p.Spec.Containers[0].Env,
			corev1.EnvVar{Name: exporter.OTLPExporterEndpointEnv, Value: "https://my-collector.example.com/otlp"},
			corev1.EnvVar{Name: exporter.OTLPExporterProtocolEnv, Value: "http/protobuf"},
		)
	})

	app := pod.Spec.Containers[0]
	assert.Contains(t, app.Env, corev1.EnvVar{Name: exporter.OTLPExporterEndpointEnv, Value: "https://my-collector.example.com/otlp"})
	assert.Contains(t, app.Env, corev1.EnvVar{Name: exporter.OTLPExporterProtocolEnv, Value: "http/protobuf"})

	assert.False(t, k8senv.Contains(app.Env, exporter.DynatraceAPITokenEnv))
	assert.False(t, k8senv.Contains(app.Env, exporter.OTLPTraceEndpointEnv))
	assert.False(t, k8senv.Contains(app.Env, exporter.OTLPLogsEndpointEnv))
	assert.False(t, k8senv.Contains(app.Env, exporter.OTLPMetricsEndpointEnv))
	assert.False(t, k8senv.Contains(app.Env, exporter.OTLPTraceHeadersEnv))
	assert.False(t, k8senv.Contains(app.Env, exporter.OTLPLogsHeadersEnv))
	assert.False(t, k8senv.Contains(app.Env, exporter.OTLPMetricsHeadersEnv))
}

func TestOTLPExporterInjectWhenInvalidGeneralEnvPreset(t *testing.T) {
	clt := setupOTLPWebhookEnv(t)

	apiURL := "https://example.live.dynatrace.com"
	dk := &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dynakube",
			Namespace: testNamespace,
			Annotations: map[string]string{
				exp.InjectionAutomaticKey: "true",
			},
		},
		Spec: dynakube.DynaKubeSpec{
			APIURL: apiURL,
			OTLPExporterConfiguration: &otlpspec.ExporterConfigurationSpec{
				NamespaceSelector: metav1.LabelSelector{
					MatchExpressions: []metav1.LabelSelectorRequirement{
						{Key: podmutator.InjectionInstanceLabel, Operator: metav1.LabelSelectorOpExists},
					},
				},
				Signals: otlpspec.SignalConfiguration{
					Metrics: &otlpspec.MetricsSignal{},
					Logs:    &otlpspec.LogsSignal{},
					Traces:  &otlpspec.TracesSignal{},
				},
			},
		},
	}

	apiTokenSecret := getOTLPExporterSecret(testNamespace)
	createObject(t, clt, apiTokenSecret)

	createDynaKube(t, clt, dk)

	pod := createPod(t, clt, func(p *corev1.Pod) {
		p.Spec.Containers[0].Env = append(p.Spec.Containers[0].Env,
			corev1.EnvVar{Name: "OTLP_EXPORTER_OTLP_ENDPOINT", Value: "https://my-collector.example.com/otlp"},
			corev1.EnvVar{Name: "OTLP_EXPORTER_OTLP_PROTOCOL", Value: "http/protobuf"},
		)
	})

	app := pod.Spec.Containers[0]

	dtTokenEnv := k8senv.Find(app.Env, exporter.DynatraceAPITokenEnv)
	require.NotNil(t, dtTokenEnv, "expected DT_API_TOKEN env var to be injected")

	assert.True(t, k8senv.Contains(app.Env, exporter.OTLPTraceEndpointEnv))
	assert.True(t, k8senv.Contains(app.Env, exporter.OTLPLogsEndpointEnv))
	assert.True(t, k8senv.Contains(app.Env, exporter.OTLPMetricsEndpointEnv))
	assert.True(t, k8senv.Contains(app.Env, exporter.OTLPTraceHeadersEnv))
	assert.True(t, k8senv.Contains(app.Env, exporter.OTLPLogsHeadersEnv))
	assert.True(t, k8senv.Contains(app.Env, exporter.OTLPMetricsHeadersEnv))

	assert.True(t, k8senv.Contains(app.Env, "OTLP_EXPORTER_OTLP_ENDPOINT"))
	assert.True(t, k8senv.Contains(app.Env, "OTLP_EXPORTER_OTLP_PROTOCOL"))
}

func createPod(t *testing.T, clt client.Client, mutateFn func(*corev1.Pod)) *corev1.Pod {
	t.Helper()

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "pod-inject-test",
			Namespace:   testNamespace,
			Annotations: map[string]string{},
		},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyAlways,
			Containers: []corev1.Container{
				{
					Name:            "app",
					Image:           "docker.io/myapp:1.2.3",
					ImagePullPolicy: corev1.PullAlways,
				},
			},
		},
	}

	if mutateFn != nil {
		mutateFn(pod)
	}

	createObject(t, clt, pod)

	return pod
}

func createObject(t *testing.T, clt client.Client, obj client.Object) {
	t.Helper()
	require.NoError(t, clt.Create(t.Context(), obj))
	t.Cleanup(func() {
		// t.Context is no longer valid during cleanup
		assert.NoError(t, clt.Delete(context.Background(), obj))
	})
}

func createDynaKube(t *testing.T, clt client.Client, dk *dynakube.DynaKube) {
	status := dk.Status
	createObject(t, clt, dk)
	dk.Status = status
	dk.UpdateStatus(t.Context(), clt)
}
func getDummyWebhookPod() *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dynatrace-webhook",
			Namespace: testNamespace,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  dtwebhook.WebhookContainerName,
					Image: "dummy-webhook-img:1.0.0",
				},
			},
		},
	}
}
func getDummyOwnerDeployment() (*appsv1.Deployment, []metav1.OwnerReference) {
	deploy := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-deployment",
			Namespace: testNamespace,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "test-app"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": "test-app"},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "test",
							Image: "dummy-app-img:1.0.0",
						},
					},
				},
			},
		},
		Status: appsv1.DeploymentStatus{},
	}
	ownerReference := []metav1.OwnerReference{
		{
			Name:       deploy.Name,
			APIVersion: deploy.APIVersion,
			Kind:       deploy.Kind,
			Controller: ptr.To(true),
			UID:        types.UID(uuid.NewString()),
		},
	}

	return deploy, ownerReference
}

func getNamespace(name string) *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				podmutator.InjectionInstanceLabel: "dynakube",
			},
		},
	}
}

func getBoostrapperSecret(namespace string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      consts.BootstrapperInitSecretName,
			Namespace: namespace,
		},
	}
}

func getOTLPExporterSecret(namespace string) *corev1.Secret {
	const dataIngestToken = "test-token"

	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      consts.OTLPExporterSecretName,
			Namespace: namespace,
		},
		Data: map[string][]byte{
			dynatrace.APIToken:        []byte(dataIngestToken),
			dynatrace.DataIngestToken: []byte(dataIngestToken),
		},
	}
}
