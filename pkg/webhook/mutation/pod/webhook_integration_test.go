// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package pod_test

import (
	"encoding/json"
	"fmt"
	"maps"
	"net/url"
	"strings"
	"testing"

	podattr "github.com/Dynatrace/dynatrace-bootstrapper/cmd/k8sinit/configure/attributes/pod"
	"github.com/Dynatrace/dynatrace-operator/cmd/bootstrapper"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/metadataenrichment"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	otlpspec "github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/otlp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/communication"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	agconsts "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/token"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8senv"
	maputils "github.com/Dynatrace/dynatrace-operator/pkg/util/map"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	podmutation "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/attributes"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/handler/otlp"
	podmutator "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	metadatamutator "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator/metadata"
	oneagentmutator "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator/otlp/exporter"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator/otlp/resourceattributes"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/volumes"
	"github.com/Dynatrace/dynatrace-operator/test/integrationtests"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
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
	metadataEnrichmentRules = metadataenrichment.Status{
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
	clt := integrationtests.SetupWebhookTestEnvironment(
		t,
		getWebhookInstallOptions(),

		func(mgr ctrl.Manager) error {
			namespace := getNamespace(testNamespace)
			namespace.Annotations = nsMetadataAnnotations
			maps.Copy(namespace.Labels, nsMetadataLabels)
			require.NoError(t, mgr.GetClient().Create(t.Context(), namespace))

			dummyWebhookPod := getDummyWebhookPod()
			require.NoError(t, mgr.GetClient().Create(t.Context(), dummyWebhookPod))
			t.Setenv(k8senv.PodName, dummyWebhookPod.Name)
			t.Setenv(k8senv.DTOperatorImageEnvName, dummyWebhookPod.Spec.Containers[0].Image)

			return podmutation.AddWebhookToManager(t.Context(), mgr, testNamespace, false)
		},
	)

	// shared between test cases
	bootstrapperSecret := getBoostrapperSecret(testNamespace)
	integrationtests.CreateKubernetesObject(t, clt, bootstrapperSecret)

	otlpExporterSecret := getOTLPExporterSecret(testNamespace)
	integrationtests.CreateKubernetesObject(t, clt, otlpExporterSecret)

	t.Run("success incl. enrichment rules, custom metadata and metadata annotation propagation", func(t *testing.T) {
		t.Run("with deprecated annotations", func(t *testing.T) {
			PropagationTest(t, clt, false)
		})
		t.Run("without deprecated annotations", func(t *testing.T) {
			PropagationTest(t, clt, true)
		})
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
					Enabled: new(true),
				},
			},
			Status: dynakube.DynaKubeStatus{
				KubernetesClusterMEID: testMEID,
				KubernetesClusterName: testClusterName,
				KubeSystemUUID:        testClusterUUID,
				OneAgent: oneagent.Status{
					ConnectionInfo: communication.ConnectionInfo{
						TenantUUID: uuid.NewString(),
					},
				},
				CodeModules: oneagent.CodeModulesStatus{
					VersionStatus: status.VersionStatus{
						Version: "1.2.3",
					},
				},
			},
		}
		integrationtests.CreateDynakube(t, clt, dk)

		overrideNamespace := getNamespace(overrideNamespaceName)
		overrideNamespace.Name = overrideNamespaceName
		overrideNamespace.Annotations = map[string]string{
			"metadata.dynatrace.com/dt.entity.kubernetes_cluster": "ns-meid",
			"metadata.dynatrace.com/k8s.cluster.name":             "override-cluster-name",
		}

		integrationtests.CreateKubernetesObject(t, clt, overrideNamespace)
		integrationtests.CreateKubernetesObject(t, clt, getBoostrapperSecret(overrideNamespaceName))
		integrationtests.CreateKubernetesObject(t, clt, getOTLPExporterSecret(overrideNamespaceName))

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
		integrationtests.CreateDynakube(t, clt, dk)

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
		integrationtests.CreateDynakube(t, clt, dk)

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
					Enabled: new(true),
				},
			},
		}
		integrationtests.CreateDynakube(t, clt, dk)

		pod := createPod(t, clt, func(pod *corev1.Pod) {
			pod.Annotations[metadatamutator.AnnotationInject] = "true"
			pod.OwnerReferences = []metav1.OwnerReference{
				{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
					Name:       "missing",
					UID:        types.UID(uuid.NewString()),
					Controller: new(true),
				},
			}
		})

		assert.Contains(t, pod.Annotations, metadatamutator.AnnotationReason)
	})

	t.Run("metadata JSON", func(t *testing.T) {
		// present in the "metadata.dynatrace.com" JSON annotation regardless of test case, since every
		// test case uses the same pod/owner shape; merged with each test case's expect below.
		commonExpected := map[string]string{
			"k8s.workload.kind": "deployment",
			"k8s.workload.name": "test-deployment",
		}

		tests := []struct {
			name                 string
			rules                []metadataenrichment.Rule
			namespaceLabels      map[string]string
			namespaceAnnotations map[string]string
			workloadLabels       map[string]string
			workloadAnnotations  map[string]string
			podLabels            map[string]string
			podAnnotations       map[string]string
			expect               map[string]string
		}{
			// general functionality: one case per rule type/source
			{
				name:            "legacy LABEL rule without target uses namespace-label key",
				rules:           []metadataenrichment.Rule{{Type: metadataenrichment.LabelRule, Source: "team"}},
				namespaceLabels: map[string]string{"team": "payments"},
				expect:          map[string]string{"k8s.namespace.label.team": "payments"},
			},
			{
				name:                 "legacy ANNOTATION rule without target uses namespace-annotation key",
				rules:                []metadataenrichment.Rule{{Type: metadataenrichment.AnnotationRule, Source: "cost-center"}},
				namespaceAnnotations: map[string]string{"cost-center": "42"},
				expect:               map[string]string{"k8s.namespace.annotation.cost-center": "42"},
			},
			{
				name:            "LABEL rule with explicit target reads namespace label",
				rules:           []metadataenrichment.Rule{{Type: metadataenrichment.LabelRule, Source: "env", Target: "custom.env"}},
				namespaceLabels: map[string]string{"env": "prod"},
				expect:          map[string]string{"custom.env": "prod"},
			},
			{
				name:                 "ANNOTATION rule with explicit target reads namespace annotation",
				rules:                []metadataenrichment.Rule{{Type: metadataenrichment.AnnotationRule, Source: "owner", Target: "custom.owner"}},
				namespaceAnnotations: map[string]string{"owner": "team-a"},
				expect:               map[string]string{"custom.owner": "team-a"},
			},
			{
				name:            "K8S_NAMESPACE_LABEL rule reads namespace label",
				rules:           []metadataenrichment.Rule{{Type: metadataenrichment.K8sNamespaceLabelRule, Source: "tier", Target: "custom.tier"}},
				namespaceLabels: map[string]string{"tier": "gold"},
				expect:          map[string]string{"custom.tier": "gold"},
			},
			{
				name:                 "K8S_NAMESPACE_ANNOTATION rule reads namespace annotation",
				rules:                []metadataenrichment.Rule{{Type: metadataenrichment.K8sNamespaceAnnotationRule, Source: "region", Target: "custom.region"}},
				namespaceAnnotations: map[string]string{"region": "eu-west-1"},
				expect:               map[string]string{"custom.region": "eu-west-1"},
			},
			{
				name:           "K8S_WORKLOAD_LABEL rule reads workload label",
				rules:          []metadataenrichment.Rule{{Type: metadataenrichment.K8sWorkloadLabelRule, Source: "team", Target: "custom.team"}},
				workloadLabels: map[string]string{"team": "checkout"},
				expect:         map[string]string{"custom.team": "checkout"},
			},
			{
				name:                "K8S_WORKLOAD_ANNOTATION rule reads workload annotation",
				rules:               []metadataenrichment.Rule{{Type: metadataenrichment.K8sWorkloadAnnotationRule, Source: "release", Target: "custom.release"}},
				workloadAnnotations: map[string]string{"release": "7"},
				expect:              map[string]string{"custom.release": "7"},
			},
			{
				name:      "K8S_POD_LABEL rule reads pod label",
				rules:     []metadataenrichment.Rule{{Type: metadataenrichment.K8sPodLabelRule, Source: "version", Target: "custom.version"}},
				podLabels: map[string]string{"version": "v2"},
				expect:    map[string]string{"custom.version": "v2"},
			},
			{
				name:           "K8S_POD_ANNOTATION rule reads pod annotation",
				rules:          []metadataenrichment.Rule{{Type: metadataenrichment.K8sPodAnnotationRule, Source: "build", Target: "custom.build"}},
				podAnnotations: map[string]string{"build": "123"},
				expect:         map[string]string{"custom.build": "123"},
			},
			{
				name:   "CUSTOM rule resolves to its literal source value",
				rules:  []metadataenrichment.Rule{{Type: metadataenrichment.CustomRule, Source: "static-value", Target: "custom.literal"}},
				expect: map[string]string{"custom.literal": "static-value"},
			},
			{
				name:  "rule with unresolved source does not produce an attribute",
				rules: []metadataenrichment.Rule{{Type: metadataenrichment.K8sWorkloadLabelRule, Source: "missing", Target: "custom.missing"}},
			},
			{
				name: "rules with different targets and source types do not interfere with each other",
				rules: []metadataenrichment.Rule{
					{Type: metadataenrichment.K8sNamespaceLabelRule, Source: "env", Target: "custom.env"},
					{Type: metadataenrichment.K8sWorkloadAnnotationRule, Source: "release", Target: "custom.release"},
					{Type: metadataenrichment.K8sPodLabelRule, Source: "version", Target: "custom.version"},
				},
				namespaceLabels:     map[string]string{"env": "prod"},
				workloadAnnotations: map[string]string{"release": "7"},
				podLabels:           map[string]string{"version": "v3"},
				expect:              map[string]string{"custom.env": "prod", "custom.release": "7", "custom.version": "v3"},
			},

			// inter-rule precedence: first rule (in definition order) that resolves to a value wins
			{
				name: "first rule targeting a key wins, later resolvable rules targeting the same key are ignored",
				rules: []metadataenrichment.Rule{
					{Type: metadataenrichment.CustomRule, Source: "first-value", Target: "shared.key"},
					{Type: metadataenrichment.CustomRule, Source: "second-value", Target: "shared.key"},
					{Type: metadataenrichment.CustomRule, Source: "third-value", Target: "shared.key"},
				},
				expect: map[string]string{"shared.key": "first-value"},
			},
			{
				name: "first-resolvable-rule-wins holds across different rule types, not by type priority",
				rules: []metadataenrichment.Rule{
					{Type: metadataenrichment.K8sWorkloadLabelRule, Source: "tier", Target: "shared.key"},
					{Type: metadataenrichment.K8sNamespaceAnnotationRule, Source: "tier", Target: "shared.key"},
				},
				namespaceAnnotations: map[string]string{"tier": "from-namespace"},
				workloadLabels:       map[string]string{"tier": "from-workload"},
				expect:               map[string]string{"shared.key": "from-workload"},
			},
			{
				name: "reversing rule order changes which value wins, proving precedence is positional, not type-based",
				rules: []metadataenrichment.Rule{
					{Type: metadataenrichment.K8sNamespaceAnnotationRule, Source: "tier", Target: "shared.key"},
					{Type: metadataenrichment.K8sWorkloadLabelRule, Source: "tier", Target: "shared.key"},
				},
				namespaceAnnotations: map[string]string{"tier": "from-namespace"},
				workloadLabels:       map[string]string{"tier": "from-workload"},
				expect:               map[string]string{"shared.key": "from-namespace"},
			},
			{
				name: "unresolved rule is skipped in favor of the next rule targeting the same key",
				rules: []metadataenrichment.Rule{
					{Type: metadataenrichment.K8sPodLabelRule, Source: "missing", Target: "shared.key"},
					{Type: metadataenrichment.CustomRule, Source: "fallback-value", Target: "shared.key"},
				},
				expect: map[string]string{"shared.key": "fallback-value"},
			},

			// hand-written metadata.dynatrace.com/ annotations: namespace < workload < pod, and the
			// annotation layers as a whole take precedence over config rules
			{
				name:                 "namespace annotation propagates to the pod's resolved metadata",
				namespaceAnnotations: map[string]string{"metadata.dynatrace.com/custom.namespace-key": "namespace-value"},
				expect:               map[string]string{"custom.namespace-key": "namespace-value"},
			},
			{
				name:                "workload annotation propagates to the pod's resolved metadata",
				workloadAnnotations: map[string]string{"metadata.dynatrace.com/custom.workload-key": "workload-value"},
				expect:              map[string]string{"custom.workload-key": "workload-value"},
			},
			{
				name:                 "workload annotation overrides namespace annotation for the same key",
				namespaceAnnotations: map[string]string{"metadata.dynatrace.com/tier": "from-namespace"},
				workloadAnnotations:  map[string]string{"metadata.dynatrace.com/tier": "from-workload"},
				expect:               map[string]string{"tier": "from-workload"},
			},
			{
				name:                "pod annotation overrides workload annotation for the same key",
				workloadAnnotations: map[string]string{"metadata.dynatrace.com/tier": "from-workload"},
				podAnnotations:      map[string]string{"metadata.dynatrace.com/tier": "from-pod"},
				expect:              map[string]string{"tier": "from-pod"},
			},
			{
				name:                 "pod annotation wins when namespace, workload and pod all set the same key",
				namespaceAnnotations: map[string]string{"metadata.dynatrace.com/tier": "from-namespace"},
				workloadAnnotations:  map[string]string{"metadata.dynatrace.com/tier": "from-workload"},
				podAnnotations:       map[string]string{"metadata.dynatrace.com/tier": "from-pod"},
				expect:               map[string]string{"tier": "from-pod"},
			},
			{
				name:                 "hand-written namespace annotation overrides a config rule targeting the same key",
				rules:                []metadataenrichment.Rule{{Type: metadataenrichment.CustomRule, Source: "from-rule", Target: "tier"}},
				namespaceAnnotations: map[string]string{"metadata.dynatrace.com/tier": "from-annotation"},
				expect:               map[string]string{"tier": "from-annotation"},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				dk := &dynakube.DynaKube{
					ObjectMeta: metav1.ObjectMeta{Name: "dynakube", Namespace: testNamespace},
					Spec:       dynakube.DynaKubeSpec{MetadataEnrichment: metadataenrichment.Spec{Enabled: new(true)}},
					Status: dynakube.DynaKubeStatus{
						KubernetesClusterMEID: testMEID,
						KubernetesClusterName: testClusterName,
						KubeSystemUUID:        testClusterUUID,
						MetadataEnrichment:    metadataenrichment.Status{Rules: tt.rules},
					},
				}
				integrationtests.CreateDynakube(t, clt, dk)

				ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{GenerateName: "ingest-rule-test-"}}
				ns.Labels = maputils.MergeMap(tt.namespaceLabels, map[string]string{podmutator.InjectionInstanceLabel: dk.Name})
				ns.Annotations = tt.namespaceAnnotations
				integrationtests.CreateKubernetesObject(t, clt, ns)
				integrationtests.CreateKubernetesObject(t, clt, getBoostrapperSecret(ns.Name))

				owner := getDummyOwnerDeployment()
				owner.Namespace = ns.Name
				owner.Labels = tt.workloadLabels
				owner.Annotations = tt.workloadAnnotations
				integrationtests.CreateKubernetesObject(t, clt, owner)

				pod := createPod(t, clt, func(pod *corev1.Pod) {
					pod.Namespace = ns.Name
					pod.Labels = tt.podLabels
					pod.Annotations = tt.podAnnotations
					require.NoError(t, controllerutil.SetControllerReference(owner, pod, scheme.Scheme))
				})

				var metadataJSON map[string]string
				require.NoError(t, json.Unmarshal([]byte(pod.Annotations["metadata.dynatrace.com"]), &metadataJSON))
				assert.Equal(t, maputils.MergeMap(commonExpected, tt.expect), metadataJSON)
			})
		}
	})
}

func PropagationTest(t *testing.T, clt client.Client, withoutDeprecatedAnnotations bool) {
	t.Helper()
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
				Enabled: new(true),
			},
		},
		Status: dynakube.DynaKubeStatus{
			KubernetesClusterMEID: testMEID,
			KubernetesClusterName: testClusterName,
			MetadataEnrichment:    metadataEnrichmentRules,
			KubeSystemUUID:        testClusterUUID,
			OneAgent: oneagent.Status{
				ConnectionInfo: communication.ConnectionInfo{
					TenantUUID: uuid.NewString(),
				},
			},
			CodeModules: oneagent.CodeModulesStatus{
				VersionStatus: status.VersionStatus{
					Version: "1.2.3",
				},
			},
		},
	}

	if withoutDeprecatedAnnotations {
		dk.Annotations[exp.EnrichmentEnableAttributesDTKubernetes] = "false"
	}

	integrationtests.CreateDynakube(t, clt, dk)

	dummyOwner := getDummyOwnerDeployment()
	integrationtests.CreateKubernetesObject(t, clt, dummyOwner)
	pod := createPod(t, clt, func(pod *corev1.Pod) {
		pod.Annotations = podMetadataAnnotations
		require.NoError(t, controllerutil.SetControllerReference(dummyOwner, pod, scheme.Scheme))
	})

	require.True(t, maputils.GetFieldBool(pod.Annotations, podmutator.AnnotationDynatraceInjected, false))
	require.True(t, maputils.GetFieldBool(pod.Annotations, metadatamutator.AnnotationInjected, false))
	require.True(t, maputils.GetFieldBool(pod.Annotations, oneagentmutator.AnnotationInjected, false))
	require.Len(t, pod.Spec.InitContainers, 1)
	assert.Contains(t, pod.Spec.InitContainers[0].Args, buildArgument("k8s.workload.kind", strings.ToLower(pod.OwnerReferences[0].Kind)))
	assert.Contains(t, pod.Spec.InitContainers[0].Args, buildArgument("k8s.workload.name", strings.ToLower(pod.OwnerReferences[0].Name)))
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

	if withoutDeprecatedAnnotations {
		assert.NotContains(t, pod.Spec.InitContainers[0].Args, buildArgument(attributes.DeprecatedWorkloadKindKey, strings.ToLower(pod.OwnerReferences[0].Kind)))
		assert.NotContains(t, pod.Spec.InitContainers[0].Args, buildArgument(attributes.DeprecatedWorkloadNameKey, strings.ToLower(pod.OwnerReferences[0].Name)))
		assert.NotContains(t, pod.Spec.InitContainers[0].Args, buildArgument(attributes.DeprecatedClusterIDKey, testClusterUUID))
	} else {
		assert.Contains(t, pod.Spec.InitContainers[0].Args, buildArgument(attributes.DeprecatedWorkloadKindKey, strings.ToLower(pod.OwnerReferences[0].Kind)))
		assert.Contains(t, pod.Spec.InitContainers[0].Args, buildArgument(attributes.DeprecatedWorkloadNameKey, strings.ToLower(pod.OwnerReferences[0].Name)))
		assert.Contains(t, pod.Spec.InitContainers[0].Args, buildArgument(attributes.DeprecatedClusterIDKey, testClusterUUID))
	}

	assert.Contains(t, pod.Spec.InitContainers[0].Args, "--attribute-container={\"container_image.registry\":\"docker.io\",\"container_image.repository\":\"myapp\",\"container_image.tags\":\"1.2.3\",\"k8s.container.name\":\"app\"}")
}

// TestConflictingVolumeType verifies that the webhook refuses to inject when a pod pre-defines one of the
// operator-managed volumes with a source that conflicts with the one the mutators would add, and that the
// corresponding not-injected reason is exposed on the pod. See ICP-6182.
func TestConflictingVolumeType(t *testing.T) {
	clt := integrationtests.SetupWebhookTestEnvironment(
		t,
		getWebhookInstallOptions(),

		func(mgr ctrl.Manager) error {
			require.NoError(t, mgr.GetClient().Create(t.Context(), getNamespace(testNamespace)))

			dummyWebhookPod := getDummyWebhookPod()
			require.NoError(t, mgr.GetClient().Create(t.Context(), dummyWebhookPod))
			t.Setenv(k8senv.PodName, dummyWebhookPod.Name)
			t.Setenv(k8senv.DTOperatorImageEnvName, dummyWebhookPod.Spec.Containers[0].Image)

			return podmutation.AddWebhookToManager(t.Context(), mgr, testNamespace, false)
		},
	)

	// shared between test cases
	integrationtests.CreateKubernetesObject(t, clt, getBoostrapperSecret(testNamespace))
	integrationtests.CreateKubernetesObject(t, clt, getOTLPExporterSecret(testNamespace))
	integrationtests.CreateKubernetesObject(t, clt, getOTLPExporterCertsSecret(testNamespace))

	// a volume with a source the mutators would never produce
	hostPathVolume := func(name string) corev1.Volume {
		return corev1.Volume{
			Name:         name,
			VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: "/"}},
		}
	}

	// a fully ready cloud-native DynaKube so injection proceeds up to the conflicting volume
	oneAgentDynaKube := func() *dynakube.DynaKube {
		return &dynakube.DynaKube{
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
			},
			Status: dynakube.DynaKubeStatus{
				OneAgent: oneagent.Status{
					ConnectionInfo: communication.ConnectionInfo{
						TenantUUID: uuid.NewString(),
					},
				},
				CodeModules: oneagent.CodeModulesStatus{
					VersionStatus: status.VersionStatus{
						Version: "1.2.3",
					},
				},
			},
		}
	}

	t.Run("config volume", func(t *testing.T) {
		integrationtests.CreateDynakube(t, clt, oneAgentDynaKube())

		pod := createPod(t, clt, func(pod *corev1.Pod) {
			pod.Spec.Volumes = append(pod.Spec.Volumes, hostPathVolume(volumes.ConfigVolumeName))
		})

		assert.Equal(t, volumes.ConflictingVolumeTypeReason, pod.Annotations[volumes.AnnotationReason])
	})

	t.Run("input volume", func(t *testing.T) {
		integrationtests.CreateDynakube(t, clt, oneAgentDynaKube())

		pod := createPod(t, clt, func(pod *corev1.Pod) {
			pod.Spec.Volumes = append(pod.Spec.Volumes, hostPathVolume(volumes.InputVolumeName))
		})

		assert.Equal(t, volumes.ConflictingVolumeTypeReason, pod.Annotations[volumes.AnnotationReason])
	})

	t.Run("oneagent-bin emptyDir volume", func(t *testing.T) {
		dk := oneAgentDynaKube()
		// a code modules image lets the pod-level volume-type annotation force the emptyDir bin volume path
		dk.Status.CodeModules.ImageID = "registry.example.com/codemodules@sha256:" + strings.Repeat("a", 64)
		integrationtests.CreateDynakube(t, clt, dk)

		pod := createPod(t, clt, func(pod *corev1.Pod) {
			pod.Annotations[oneagentmutator.AnnotationVolumeType] = oneagentmutator.EphemeralVolumeType
			pod.Spec.Volumes = append(pod.Spec.Volumes, hostPathVolume(oneagentmutator.BinVolumeName))
		})

		assert.Equal(t, volumes.ConflictingVolumeTypeReason, pod.Annotations[oneagentmutator.AnnotationReason])
	})

	t.Run("oneagent-bin CSI volume", func(t *testing.T) {
		dk := oneAgentDynaKube()
		// a code modules image lets the pod-level volume-type annotation force the CSI bin volume path
		dk.Status.CodeModules.ImageID = "registry.example.com/codemodules@sha256:" + strings.Repeat("a", 64)
		integrationtests.CreateDynakube(t, clt, dk)

		pod := createPod(t, clt, func(pod *corev1.Pod) {
			pod.Annotations[oneagentmutator.AnnotationVolumeType] = oneagentmutator.CSIVolumeType
			pod.Spec.Volumes = append(pod.Spec.Volumes, hostPathVolume(oneagentmutator.BinVolumeName))
		})

		assert.Equal(t, volumes.ConflictingVolumeTypeReason, pod.Annotations[oneagentmutator.AnnotationReason])
	})

	t.Run("otlp activegate cert volume", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "dynakube",
				Namespace: testNamespace,
				Annotations: map[string]string{
					exp.InjectionAutomaticKey: "true",
				},
			},
			Spec: dynakube.DynaKubeSpec{
				APIURL: "https://example.live.dynatrace.com",
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
					ConnectionInfo: communication.ConnectionInfo{
						TenantUUID: uuid.NewString(),
					},
				},
			},
		}
		integrationtests.CreateDynakube(t, clt, dk)

		pod := createPod(t, clt, func(pod *corev1.Pod) {
			pod.Spec.Volumes = append(pod.Spec.Volumes, hostPathVolume(exporter.ActiveGateTrustedCertVolumeName))
		})

		assert.Equal(t, volumes.ConflictingVolumeTypeReason, pod.Annotations[podmutator.AnnotationOTLPReason])
	})
}

func TestOTLPWebhook(t *testing.T) { //nolint:revive
	clt := integrationtests.SetupWebhookTestEnvironment(
		t,
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
			t.Setenv(k8senv.DTOperatorImageEnvName, dummyWebhookPod.Spec.Containers[0].Image)

			return podmutation.AddWebhookToManager(t.Context(), mgr, testNamespace, false)
		},
	)

	t.Run("otlp exporter with ns metadata propagation and custom enrichment rules", func(t *testing.T) {
		apiURL := "https://example.live.dynatrace.com"
		type testCase struct {
			name                     string
			annotations              map[string]string
			withDeprecatedAttributes bool
		}

		testCases := []testCase{
			{
				name:                     "without deprecated annotations",
				annotations:              map[string]string{exp.EnrichmentEnableAttributesDTKubernetes: "false"},
				withDeprecatedAttributes: false,
			},
			{
				name:                     "with deprecated annotations",
				annotations:              map[string]string{},
				withDeprecatedAttributes: true,
			},
		}

		for _, tc := range testCases {
			annotations := map[string]string{exp.InjectionAutomaticKey: "true"}
			t.Run(tc.name, func(t *testing.T) {
				maps.Copy(annotations, tc.annotations)
				dk := &dynakube.DynaKube{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "dynakube",
						Namespace:   testNamespace,
						Annotations: annotations,
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
						MetadataEnrichment:    metadataEnrichmentRules,
					},
				}

				apiTokenSecret := getOTLPExporterSecret(testNamespace)
				integrationtests.CreateKubernetesObject(t, clt, apiTokenSecret)

				integrationtests.CreateDynakube(t, clt, dk)

				dummyOwner := getDummyOwnerDeployment()
				integrationtests.CreateKubernetesObject(t, clt, dummyOwner)
				pod := createPod(t, clt, func(pod *corev1.Pod) {
					pod.Annotations = podMetadataAnnotations
					require.NoError(t, controllerutil.SetControllerReference(dummyOwner, pod, scheme.Scheme))
				})

				// verify mutation occurred by presence of OTLP env vars (annotation may not be set when no OneAgent injection)

				appContainer := pod.Spec.Containers[0]
				dtTokenEnv := k8senv.Find(appContainer.Env, exporter.DynatraceAPITokenEnv)
				require.NotNil(t, dtTokenEnv, "expected DT_API_TOKEN env var to be injected")
				require.NotNil(t, dtTokenEnv.ValueFrom)
				require.NotNil(t, dtTokenEnv.ValueFrom.SecretKeyRef)
				assert.Equal(t, consts.OTLPExporterSecretName, dtTokenEnv.ValueFrom.SecretKeyRef.Name)
				assert.Equal(t, token.DataIngestKey, dtTokenEnv.ValueFrom.SecretKeyRef.Key)

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

				raEnv := k8senv.Find(appContainer.Env, resourceattributes.OTelResourceAttributesEnv)

				require.NotNil(t, raEnv, "OTEL_RESOURCE_ATTRIBUTES missing")

				gotResourceAttributes, envVarFound := resourceattributes.NewAttributesFromEnv(appContainer.Env, resourceattributes.OTelResourceAttributesEnv)
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
				assert.Equal(t, pod.OwnerReferences[0].Name, gotResourceAttributes["k8s.workload.name"])
				assert.Equal(t, strings.ToLower(pod.OwnerReferences[0].Kind), gotResourceAttributes["k8s.workload.kind"])
				assert.Equal(t, dk.Status.KubernetesClusterMEID, gotResourceAttributes["dt.entity.kubernetes_cluster"])

				if tc.withDeprecatedAttributes {
					assert.Equal(t, dk.Status.KubeSystemUUID, gotResourceAttributes[attributes.DeprecatedClusterIDKey])
					assert.Equal(t, pod.OwnerReferences[0].Name, gotResourceAttributes[attributes.DeprecatedWorkloadNameKey])
					assert.Equal(t, strings.ToLower(pod.OwnerReferences[0].Kind), gotResourceAttributes[attributes.DeprecatedWorkloadKindKey])
				}

				assert.Equal(t, url.QueryEscape(nsMetadataAnnotations["metadata.dynatrace.com/custom.ns-meta"]), gotResourceAttributes["custom.ns-meta"])
				assert.Equal(t, url.QueryEscape(podMetadataAnnotations["metadata.dynatrace.com/service.name"]), gotResourceAttributes["service.name"])
				assert.Equal(t, url.QueryEscape(podMetadataAnnotations["metadata.dynatrace.com/custom.key"]), gotResourceAttributes["custom.key"])
				assert.Equal(t, url.QueryEscape(nsMetadataAnnotations[testCustomMetadataAnnotation]), gotResourceAttributes["k8s.namespace.annotation."+testCustomMetadataAnnotation])
				assert.Equal(t, url.QueryEscape(nsMetadataAnnotations[testCostCenterAnnotation]), gotResourceAttributes["dt.cost.costcenter"])
				assert.Equal(t, url.QueryEscape(nsMetadataLabels[testSecContextLabel]), gotResourceAttributes["dt.security_context"])
				assert.Equal(t, url.QueryEscape(nsMetadataLabels[testCustomMetadataLabel]), gotResourceAttributes["k8s.namespace.label."+testCustomMetadataLabel])
			})
		}
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
			},
		}

		integrationtests.CreateDynakube(t, clt, dk)

		overrideNamespace := getNamespace(overrideNamespaceName)
		overrideNamespace.Annotations = map[string]string{
			"metadata.dynatrace.com/dt.entity.kubernetes_cluster": "ns-meid",
			"metadata.dynatrace.com/k8s.cluster.name":             "override-cluster-name",
		}
		integrationtests.CreateKubernetesObject(t, clt, overrideNamespace)

		apiTokenSecret := getOTLPExporterSecret(overrideNamespaceName)
		integrationtests.CreateKubernetesObject(t, clt, apiTokenSecret)

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
		assert.Equal(t, token.DataIngestKey, dtTokenEnv.ValueFrom.SecretKeyRef.Key)

		raEnv := k8senv.Find(appContainer.Env, resourceattributes.OTelResourceAttributesEnv)
		require.NotNil(t, raEnv, "OTEL_RESOURCE_ATTRIBUTES missing")

		gotResourceAttributes, envVarFound := resourceattributes.NewAttributesFromEnv(appContainer.Env, resourceattributes.OTelResourceAttributesEnv)
		require.True(t, envVarFound, "OTEL_RESOURCE_ATTRIBUTES missing")

		assert.Equal(t, overrideNamespaceName, gotResourceAttributes["k8s.namespace.name"])
		assert.Equal(t, "override-pod-name", gotResourceAttributes["k8s.pod.name"])
		assert.Equal(t, "override-cluster-name", gotResourceAttributes["k8s.cluster.name"])
		assert.Equal(t, "pod-meid", gotResourceAttributes["dt.entity.kubernetes_cluster"])
	})

	t.Run("resource attribute full precedence chain", func(t *testing.T) {
		// Precedence order low→high for OTEL_RESOURCE_ATTRIBUTES:
		//   enrichment rules
		//   < dynakube.resourceAttributes
		//   < dynakube.otlpExporterConfiguration.additionalResourceAttributes
		//   < namespace metadata.dynatrace.com/ annotations
		//   < pod metadata.dynatrace.com/ annotations
		//   < existing OTEL_RESOURCE_ATTRIBUTES
		//
		// For the JSON annotation at "metadata.dynatrace.com" (caseJSONAnnotation):
		//   enrichment rules < dynakube < namespaceAnnotations < podAnnotations (existing OTEL_RA not included)
		const raPrecedenceNs = "ra-precedence-ns"

		raNS := getNamespace(raPrecedenceNs)
		raNS.Annotations = map[string]string{
			// wins over dynakube in all combine cases
			"metadata.dynatrace.com/conflict.ns.vs.dynakube": "from-ns",
			// will be overridden by pod annotation in OTEL_RA and JSON, but preserved via SetAnnotationIfNotExists
			"metadata.dynatrace.com/conflict.pod.vs.ns": "from-ns",
		}
		// label picked up by the enrichment rule below; dynakube layer must win over the rule result
		raNS.Labels["conflict-rule-label"] = "from-rule"
		integrationtests.CreateKubernetesObject(t, clt, raNS)
		integrationtests.CreateKubernetesObject(t, clt, getOTLPExporterSecret(raPrecedenceNs))

		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "dynakube",
				Namespace: testNamespace,
				Annotations: map[string]string{
					exp.InjectionAutomaticKey: "true",
				},
			},
			Spec: dynakube.DynaKubeSpec{
				APIURL: "https://example.live.dynatrace.com",
				ResourceAttributes: map[string]string{
					// overridden by additionalResourceAttributes within the dynakube layer
					"conflict.additional.vs.global": "from-global",
					// overridden by namespace annotation in every combine case
					"conflict.ns.vs.dynakube": "from-dynakube",
					// wins over enrichment-rule result for the same key (dynakube layer 7 > rules layer 6)
					"conflict.dynakube.vs.rules": "from-dynakube",
				},
				OTLPExporterConfiguration: &otlpspec.ExporterConfigurationSpec{
					NamespaceSelector: metav1.LabelSelector{
						MatchExpressions: []metav1.LabelSelectorRequirement{
							{Key: podmutator.InjectionInstanceLabel, Operator: metav1.LabelSelectorOpExists},
						},
					},
					Signals: otlpspec.SignalConfiguration{
						Metrics: &otlpspec.MetricsSignal{},
					},
					AdditionalResourceAttributes: map[string]string{
						// wins over the same key in resourceAttributes
						"conflict.additional.vs.global": "from-additional",
					},
				},
			},
			Status: dynakube.DynaKubeStatus{
				KubernetesClusterMEID: testMEID,
				KubernetesClusterName: testClusterName,
				MetadataEnrichment: metadataenrichment.Status{
					Rules: []metadataenrichment.Rule{
						{
							Type:   "LABEL",
							Source: "conflict-rule-label",
							Target: "conflict.dynakube.vs.rules",
						},
					},
				},
			},
		}
		integrationtests.CreateDynakube(t, clt, dk)

		pod := createPod(t, clt, func(pod *corev1.Pod) {
			pod.Namespace = raPrecedenceNs
			pod.Annotations = map[string]string{
				// wins over ns annotation in OTEL_RA and JSON annotation
				"metadata.dynatrace.com/conflict.pod.vs.ns": "from-pod",
				// wins over this same-keyed existing OTEL_RA value in JSON and metadata annotations,
				// but the existing OTEL_RA wins in the OTEL_RESOURCE_ATTRIBUTES env var
				"metadata.dynatrace.com/conflict.existing.vs.pod": "from-pod",
			}
			pod.Spec.Containers[0].Env = []corev1.EnvVar{
				// existing OTEL_RESOURCE_ATTRIBUTES has highest precedence in the env var
				{Name: resourceattributes.OTelResourceAttributesEnv, Value: "conflict.existing.vs.pod=from-existing"},
			}
		})

		appContainer := pod.Spec.Containers[0]
		gotRA, found := resourceattributes.NewAttributesFromEnv(appContainer.Env, resourceattributes.OTelResourceAttributesEnv)
		require.True(t, found, "OTEL_RESOURCE_ATTRIBUTES missing")

		// additionalResourceAttributes wins over resourceAttributes (both merged into dynakube layer)
		assert.Equal(t, "from-additional", gotRA["conflict.additional.vs.global"], "additionalResourceAttributes must win over resourceAttributes")
		// dynakube.resourceAttributes wins over enrichment-rule result (dynakube layer 7 > rules layer 6)
		assert.Equal(t, "from-dynakube", gotRA["conflict.dynakube.vs.rules"], "dynakube must win over enrichment rules")
		// namespace metadata.dynatrace.com/ annotation wins over dynakube layer
		assert.Equal(t, "from-ns", gotRA["conflict.ns.vs.dynakube"], "namespace annotation must win over dynakube layer")
		// pod metadata.dynatrace.com/ annotation wins over namespace annotation
		assert.Equal(t, "from-pod", gotRA["conflict.pod.vs.ns"], "pod annotation must win over namespace annotation")
		// existing OTEL_RESOURCE_ATTRIBUTES wins over pod annotation
		assert.Equal(t, "from-existing", gotRA["conflict.existing.vs.pod"], "existing OTEL_RESOURCE_ATTRIBUTES must win over pod annotation")

		// pre-seeded pod annotations are preserved
		assert.Equal(t, "from-pod", pod.Annotations["metadata.dynatrace.com/conflict.pod.vs.ns"],
			"pre-existing pod annotation must be preserved")
		assert.Equal(t, "from-pod", pod.Annotations["metadata.dynatrace.com/conflict.existing.vs.pod"],
			"pre-existing pod annotation must be preserved")

		// JSON annotation at "metadata.dynatrace.com" (caseJSONAnnotation: dynakube + namespaceAnnotations + podAnnotations, no custom/existing OTEL_RA)
		jsonAnnotation := pod.Annotations["metadata.dynatrace.com"]
		require.NotEmpty(t, jsonAnnotation, "JSON metadata annotation missing")

		var jsonAttrs map[string]string
		require.NoError(t, json.Unmarshal([]byte(jsonAnnotation), &jsonAttrs))

		assert.Equal(t, "from-additional", jsonAttrs["conflict.additional.vs.global"],
			"dynakube value must appear in JSON annotation")
		// dynakube wins over enrichment rules in the JSON annotation too (caseJSONAnnotation includes both layers)
		assert.Equal(t, "from-dynakube", jsonAttrs["conflict.dynakube.vs.rules"],
			"dynakube must win over enrichment rules in JSON annotation")
		assert.Equal(t, "from-ns", jsonAttrs["conflict.ns.vs.dynakube"],
			"namespace annotation must win over dynakube in JSON annotation")
		// existing OTEL_RESOURCE_ATTRIBUTES (custom layer) is NOT included in the JSON annotation
		assert.Equal(t, "from-pod", jsonAttrs["conflict.existing.vs.pod"],
			"JSON annotation must reflect pod annotation, not the existing OTEL_RESOURCE_ATTRIBUTES value")
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

		integrationtests.CreateDynakube(t, clt, dk)

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
					ConnectionInfo: communication.ConnectionInfo{
						TenantUUID: tenantUUID,
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
				token.APIKey:        []byte(dataIngestToken),
				token.DataIngestKey: []byte(dataIngestToken),
			},
		}
		integrationtests.CreateKubernetesObject(t, clt, apiTokenSecret)

		agCertSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      consts.OTLPExporterCertsSecretName,
				Namespace: testNamespace,
			},
			Data: map[string][]byte{
				consts.TLSCrtDataName: []byte(agCertData),
			},
		}
		integrationtests.CreateKubernetesObject(t, clt, agCertSecret)

		integrationtests.CreateDynakube(t, clt, dk)

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
		assert.Equal(t, token.DataIngestKey, dtTokenEnv.ValueFrom.SecretKeyRef.Key)

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
					ConnectionInfo: communication.ConnectionInfo{
						TenantUUID: tenantUUID,
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
				token.APIKey:        []byte(dataIngestToken),
				token.DataIngestKey: []byte(dataIngestToken),
			},
		}
		integrationtests.CreateKubernetesObject(t, clt, apiTokenSecret)

		integrationtests.CreateDynakube(t, clt, dk)

		pod := createPod(t, clt, nil)

		assert.False(t, maputils.GetFieldBool(pod.Annotations, podmutator.AnnotationOTLPInjected, false))
		assert.Equal(t, otlp.NoOTLPExporterActiveGateCertSecretReason, pod.Annotations[podmutator.AnnotationOTLPReason])
	})
}

func getWebhookInstallOptions() envtest.WebhookInstallOptions {
	return envtest.WebhookInstallOptions{
		MutatingWebhooks: []*admissionregistrationv1.MutatingWebhookConfiguration{
			// TODO(avorima): Load this from a file using Paths
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "dynatrace-webhook",
				},
				Webhooks: []admissionregistrationv1.MutatingWebhook{
					{
						Name:               "webhook.pod.dynatrace.com",
						ReinvocationPolicy: new(admissionregistrationv1.IfNeededReinvocationPolicy),
						FailurePolicy:      new(admissionregistrationv1.Ignore),
						TimeoutSeconds:     new(int32(30)),
						Rules: []admissionregistrationv1.RuleWithOperations{
							{
								Rule: admissionregistrationv1.Rule{
									APIGroups:   []string{""},
									APIVersions: []string{"v1"},
									Resources:   []string{"pods"},
									Scope:       new(admissionregistrationv1.NamespacedScope),
								},
								Operations: []admissionregistrationv1.OperationType{
									admissionregistrationv1.Create,
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
						ClientConfig: admissionregistrationv1.WebhookClientConfig{
							Service: &admissionregistrationv1.ServiceReference{
								Name: "dynatrace-webhook",
								Path: new("/inject"),
							},
						},
						AdmissionReviewVersions: []string{"v1beta1", "v1"},
						SideEffects:             new(admissionregistrationv1.SideEffectClassNone),
					},
				},
			},
		},
	}
}

func setupOTLPWebhookEnv(t *testing.T) client.Client {
	t.Helper()

	return integrationtests.SetupWebhookTestEnvironment(
		t,
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
			t.Setenv(k8senv.DTOperatorImageEnvName, dummyWebhookPod.Spec.Containers[0].Image)

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

	integrationtests.CreateDynakube(t, clt, dk)

	pod := createPod(t, clt, func(p *corev1.Pod) {
		p.Spec.Containers[0].Env = append(
			p.Spec.Containers[0].Env,
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
	integrationtests.CreateKubernetesObject(t, clt, apiTokenSecret)

	integrationtests.CreateDynakube(t, clt, dk)

	pod := createPod(t, clt, func(p *corev1.Pod) {
		p.Spec.Containers[0].Env = append(
			p.Spec.Containers[0].Env,
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
					Name:  "app",
					Image: "docker.io/myapp:1.2.3",
				},
			},
		},
	}

	if mutateFn != nil {
		mutateFn(pod)
	}

	integrationtests.CreateKubernetesObject(t, clt, pod)

	return pod
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

func getDummyOwnerDeployment() *appsv1.Deployment {
	return &appsv1.Deployment{
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
	}
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
			token.APIKey:        []byte(dataIngestToken),
			token.DataIngestKey: []byte(dataIngestToken),
		},
	}
}

func getOTLPExporterCertsSecret(namespace string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      consts.OTLPExporterCertsSecretName,
			Namespace: namespace,
		},
		Data: map[string][]byte{
			consts.TLSCrtDataName: []byte("ag-cert-data"),
		},
	}
}
