// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package pod_test

import (
	"encoding/json"
	"fmt"
	"maps"
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
	apiURL                       = "https://example.live.dynatrace.com"
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

func TestWebhook(t *testing.T) { //nolint:revive // Complexity too high
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

	t.Run("reject", func(t *testing.T) {
		// a volume with a source the mutators would never produce
		hostPathVolume := func(name string) corev1.Volume {
			return corev1.Volume{
				Name:         name,
				VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: "/"}},
			}
		}

		tests := []struct {
			name      string
			dk        *dynakube.DynaKube
			podMutate func(*corev1.Pod)
			expect    map[string]string
		}{
			{
				"success",
				getReadyCNFSDynaKube(),
				nil,
				map[string]string{podmutator.AnnotationDynatraceInjected: "true", metadatamutator.AnnotationInjected: "true", oneagentmutator.AnnotationInjected: "true"},
			},
			{
				"oneagent missing tenant UUID",
				func() *dynakube.DynaKube {
					dk := getReadyCNFSDynaKube()
					dk.Status.OneAgent.ConnectionInfo.TenantUUID = ""

					return dk
				}(),
				nil,
				map[string]string{oneagentmutator.AnnotationReason: oneagentmutator.MissingTenantUUIDReason},
			},
			{
				"oneagent version not ready",
				func() *dynakube.DynaKube {
					dk := getReadyCNFSDynaKube()
					dk.Status.CodeModules.Version = ""

					return dk
				}(),
				nil,
				map[string]string{oneagentmutator.AnnotationReason: oneagentmutator.DynaKubeStatusNotReadyReason},
			},
			{
				"metadata owner lookup",
				&dynakube.DynaKube{
					ObjectMeta: metav1.ObjectMeta{Name: "dynakube", Namespace: testNamespace},
					Spec:       dynakube.DynaKubeSpec{MetadataEnrichment: metadataenrichment.Spec{Enabled: new(true)}},
				},
				func(pod *corev1.Pod) {
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
				},
				map[string]string{metadatamutator.AnnotationReason: metadatamutator.OwnerLookupFailedReason},
			},
			{
				"data ingest token secret missing",
				getReadyOTLPDynaKube(),
				nil,
				map[string]string{podmutator.AnnotationOTLPInjected: "false", podmutator.AnnotationOTLPReason: otlp.NoOTLPExporterConfigSecretReason},
			},
			{
				"conflicting config volume",
				getReadyCNFSDynaKube(),
				func(pod *corev1.Pod) {
					pod.Spec.Volumes = append(pod.Spec.Volumes, hostPathVolume(volumes.ConfigVolumeName))
				},
				map[string]string{volumes.AnnotationReason: volumes.ConflictingVolumeTypeReason},
			},
			{
				"conflicting input volume",
				getReadyCNFSDynaKube(),
				func(pod *corev1.Pod) {
					pod.Spec.Volumes = append(pod.Spec.Volumes, hostPathVolume(volumes.ConfigVolumeName))
				},
				map[string]string{volumes.AnnotationReason: volumes.ConflictingVolumeTypeReason},
			},
			{
				"conflicting oneagent-bin emptyDir volume",
				func() *dynakube.DynaKube {
					dk := getReadyCNFSDynaKube()
					dk.Status.CodeModules.ImageID = "registry.example.com/codemodules@sha256:" + strings.Repeat("a", 64)

					return dk
				}(),
				func(pod *corev1.Pod) {
					pod.Annotations[oneagentmutator.AnnotationVolumeType] = oneagentmutator.EphemeralVolumeType
					pod.Spec.Volumes = append(pod.Spec.Volumes, hostPathVolume(oneagentmutator.BinVolumeName))
				},
				map[string]string{oneagentmutator.AnnotationReason: volumes.ConflictingVolumeTypeReason},
			},
			{
				"conflicting oneagent-bin CSI volume",
				func() *dynakube.DynaKube {
					dk := getReadyCNFSDynaKube()
					dk.Status.CodeModules.ImageID = "registry.example.com/codemodules@sha256:" + strings.Repeat("a", 64)

					return dk
				}(),
				func(pod *corev1.Pod) {
					pod.Annotations[oneagentmutator.AnnotationVolumeType] = oneagentmutator.CSIVolumeType
					pod.Spec.Volumes = append(pod.Spec.Volumes, hostPathVolume(oneagentmutator.BinVolumeName))
				},
				map[string]string{oneagentmutator.AnnotationReason: volumes.ConflictingVolumeTypeReason},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				integrationtests.CreateDynakube(t, clt, tt.dk)
				integrationtests.CreateKubernetesObject(t, clt, getBoostrapperSecret(testNamespace))

				pod := createPod(t, clt, tt.podMutate)

				assert.Subset(t, pod.Annotations, tt.expect)
			})
		}

		t.Run("conflicting otlp activegate cert volume", func(t *testing.T) {
			dk := getReadyOTLPDynaKube()
			dk.Spec.ActiveGate.Capabilities = []activegate.CapabilityDisplayName{activegate.RoutingCapability.DisplayName}
			dk.Status.OneAgent.ConnectionInfo.TenantUUID = uuid.NewString()

			integrationtests.CreateDynakube(t, clt, dk)
			integrationtests.CreateKubernetesObject(t, clt, getOTLPExporterSecret(testNamespace))
			integrationtests.CreateKubernetesObject(t, clt, getOTLPExporterCertsSecret(testNamespace))

			pod := createPod(t, clt, func(pod *corev1.Pod) {
				pod.Spec.Volumes = append(pod.Spec.Volumes, hostPathVolume(exporter.ActiveGateTrustedCertVolumeName))
			})

			assert.Equal(t, volumes.ConflictingVolumeTypeReason, pod.Annotations[podmutator.AnnotationOTLPReason])
		})

		t.Run("otlp exporter activegate certificate secret missing", func(t *testing.T) {
			dk := getReadyOTLPDynaKube()
			dk.Spec.ActiveGate.Capabilities = []activegate.CapabilityDisplayName{activegate.RoutingCapability.DisplayName}
			dk.Status.OneAgent.ConnectionInfo.TenantUUID = uuid.NewString()
			integrationtests.CreateDynakube(t, clt, dk)
			integrationtests.CreateKubernetesObject(t, clt, getOTLPExporterSecret(testNamespace))

			pod := createPod(t, clt, nil)

			assert.False(t, maputils.GetFieldBool(pod.Annotations, podmutator.AnnotationOTLPInjected, false))
			assert.Equal(t, otlp.NoOTLPExporterActiveGateCertSecretReason, pod.Annotations[podmutator.AnnotationOTLPReason])
		})
	})

	t.Run("metadata JSON", func(t *testing.T) {
		tests := []metadataJSONTestCase{
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
				name: "first rule wins",
				rules: []metadataenrichment.Rule{
					{Type: metadataenrichment.CustomRule, Source: "first", Target: "key"},
					{Type: metadataenrichment.CustomRule, Source: "second", Target: "key"},
					{Type: metadataenrichment.CustomRule, Source: "third", Target: "key"},
				},
				expect: map[string]string{"key": "first"},
			},
			{
				name: "first rule wins across types",
				rules: []metadataenrichment.Rule{
					{Type: metadataenrichment.K8sWorkloadLabelRule, Source: "tier", Target: "key"},
					{Type: metadataenrichment.K8sNamespaceAnnotationRule, Source: "tier", Target: "key"},
					{Type: metadataenrichment.K8sPodLabelRule, Source: "tier", Target: "key"},
				},
				namespaceAnnotations: map[string]string{"tier": "namespace"},
				workloadLabels:       map[string]string{"tier": "workload"},
				podLabels:            map[string]string{"tier": "pod"},
				expect:               map[string]string{"key": "workload"},
			},
			{
				name: "unresolved rule is skipped",
				rules: []metadataenrichment.Rule{
					{Type: metadataenrichment.K8sPodLabelRule, Source: "missing", Target: "key"},
					{Type: metadataenrichment.CustomRule, Source: "fallback", Target: "key"},
				},
				expect: map[string]string{"key": "fallback"},
			},

			// hand-written metadata.dynatrace.com/ annotations: namespace < workload < pod, and the
			// annotation layers as a whole take precedence over config rules
			{
				name:                 "namespace annotation",
				namespaceAnnotations: map[string]string{"metadata.dynatrace.com/tier": "namespace"},
				expect:               map[string]string{"tier": "namespace"},
			},
			{
				name:                "workload annotation",
				workloadAnnotations: map[string]string{"metadata.dynatrace.com/tier": "workload"},
				expect:              map[string]string{"tier": "workload"},
			},
			{
				name:                 "workload annotation overrides namespace",
				namespaceAnnotations: map[string]string{"metadata.dynatrace.com/tier": "namespace"},
				workloadAnnotations:  map[string]string{"metadata.dynatrace.com/tier": "workload"},
				expect:               map[string]string{"tier": "workload"},
			},
			{
				name:                "pod annotation overrides workload",
				workloadAnnotations: map[string]string{"metadata.dynatrace.com/tier": "workload"},
				podAnnotations:      map[string]string{"metadata.dynatrace.com/tier": "pod"},
				expect:              map[string]string{"tier": "pod"},
			},
			{
				name:                 "pod annotation overrides namespace and workload",
				namespaceAnnotations: map[string]string{"metadata.dynatrace.com/tier": "namespace"},
				workloadAnnotations:  map[string]string{"metadata.dynatrace.com/tier": "workload"},
				podAnnotations:       map[string]string{"metadata.dynatrace.com/tier": "pod"},
				expect:               map[string]string{"tier": "pod"},
			},
			{
				name:                 "annotation overrides rule",
				rules:                []metadataenrichment.Rule{{Type: metadataenrichment.CustomRule, Source: "rule", Target: "tier"}},
				namespaceAnnotations: map[string]string{"metadata.dynatrace.com/tier": "annotation"},
				expect:               map[string]string{"tier": "annotation"},
			},
			{
				name:                 "annotation overrides attribute",
				namespaceAnnotations: map[string]string{"metadata.dynatrace.com/foo": "annotation"},
				resourceAttributes:   map[string]string{"foo": "global"},
				expect:               map[string]string{"foo": "annotation"},
			},

			{
				name:               "resource attributes",
				resourceAttributes: map[string]string{"foo": "global"},
				expect:             map[string]string{"foo": "global"},
			},
			{
				name:               "oneagent attributes override",
				resourceAttributes: map[string]string{"foo": "global"},
				oaAttributes:       map[string]string{"foo": "oneagent"},
				expect:             map[string]string{"foo": "oneagent"},
			},
			{
				name:               "otlp attributes override",
				resourceAttributes: map[string]string{"foo": "global"},
				otlpAttributes:     map[string]string{"foo": "otlp"},
				expect:             map[string]string{"foo": "otlp"},
			},

			{
				name: "resource attributes override rule",
				rules: []metadataenrichment.Rule{
					{Type: metadataenrichment.K8sNamespaceLabelRule, Source: "tier", Target: "foo"},
				},
				namespaceAnnotations: map[string]string{"tier": "namespace"},
				resourceAttributes:   map[string]string{"foo": "global"},
				expect:               map[string]string{"foo": "global"},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				testMetadataJSON(t, clt, tt)
			})
		}
	})

	t.Run("bootstrapper args", func(t *testing.T) {
		t.Run("with deprecated fields", func(t *testing.T) {
			testBoostrapperArgs(t, clt, false)
		})
		t.Run("without deprecated fields", func(t *testing.T) {
			testBoostrapperArgs(t, clt, true)
		})
	})

	t.Run("OTLP", func(t *testing.T) {
		tests := []otlpTestCase{
			{
				name:                  "without deprecated keys",
				withoutDeprecatedKeys: true,
			},
			{
				name:                 "user-provided annotations",
				namespaceAnnotations: map[string]string{"metadata.dynatrace.com/ns-key": "ns-value", "ignore-namespace": "foo"},
				workloadAnnotations:  map[string]string{"metadata.dynatrace.com/workload-key": "workload-value", "ignore-workload": "bar"},
				podAnnotations:       map[string]string{"metadata.dynatrace.com/pod-key": "pod-value", "ignore-pod": "baz"},
				expectAttributes:     map[string]string{"ns-key": "ns-value", "workload-key": "workload-value", "pod-key": "pod-value"},
			},
			{
				name:                 "values are escaped",
				namespaceAnnotations: map[string]string{"metadata.dynatrace.com/ns-key": "ns/value"},
				workloadAnnotations:  map[string]string{"metadata.dynatrace.com/workload-key": "workload:value"},
				podAnnotations:       map[string]string{"metadata.dynatrace.com/pod-key": "pod value"},
				expectAttributes:     map[string]string{"ns-key": "ns%2Fvalue", "workload-key": "workload%3Avalue", "pod-key": "pod+value"},
			},
			{
				name: "first rule wins",
				rules: []metadataenrichment.Rule{
					{Type: metadataenrichment.K8sNamespaceLabelRule, Source: "lookup", Target: "key"},
					{Type: metadataenrichment.K8sWorkloadLabelRule, Source: "lookup", Target: "key"},
					{Type: metadataenrichment.K8sPodLabelRule, Source: "lookup", Target: "key"},
				},
				namespaceLabels:  map[string]string{"lookup": "namespace"},
				workloadLabels:   map[string]string{"lookup": "workload"},
				podLabels:        map[string]string{"lookup": "pod"},
				expectAttributes: map[string]string{"key": "namespace"},
			},
			{
				name:               "resource attributes",
				resourceAttributes: map[string]string{"global": "global", "shared": "global"},
				otlpAttributes:     map[string]string{"shared": "otlp", "foo": "bar"},
				expectAttributes:   map[string]string{"global": "global", "shared": "otlp", "foo": "bar"},
			},
			{
				name:             "existing env",
				podEnvs:          []corev1.EnvVar{{Name: resourceattributes.OTelResourceAttributesEnv, Value: "foo=bar"}},
				expectAttributes: map[string]string{"foo": "bar"},
			},

			{
				name: "rules override cluster info",
				rules: []metadataenrichment.Rule{
					{Type: metadataenrichment.K8sNamespaceLabelRule, Source: "lookup", Target: "k8s.namespace.name"},
					{Type: metadataenrichment.K8sWorkloadLabelRule, Source: "lookup", Target: "k8s.workload.name"},
					{Type: metadataenrichment.K8sPodLabelRule, Source: "lookup", Target: "k8s.pod.name"},
				},
				namespaceLabels:  map[string]string{"lookup": "namespace-value"},
				workloadLabels:   map[string]string{"lookup": "workload-value"},
				podLabels:        map[string]string{"lookup": "pod-value"},
				expectAttributes: map[string]string{"k8s.namespace.name": "namespace-value", "k8s.workload.name": "workload-value", "k8s.pod.name": "pod-value"},
			},
			{
				name: "resource attributes override rules",
				rules: []metadataenrichment.Rule{
					{Type: metadataenrichment.K8sNamespaceLabelRule, Source: "lookup", Target: "namespace-key"},
					{Type: metadataenrichment.K8sWorkloadLabelRule, Source: "lookup", Target: "workload-key"},
					{Type: metadataenrichment.K8sPodLabelRule, Source: "lookup", Target: "pod-key"},
				},
				namespaceLabels:    map[string]string{"lookup": "base"},
				workloadLabels:     map[string]string{"lookup": "base"},
				podLabels:          map[string]string{"lookup": "base"},
				resourceAttributes: map[string]string{"namespace-key": "override", "workload-key": "override"},
				otlpAttributes:     map[string]string{"pod-key": "override"},
				expectAttributes:   map[string]string{"namespace-key": "override", "workload-key": "override", "pod-key": "override"},
			},
			{
				name: "annotations override rules",
				rules: []metadataenrichment.Rule{
					{Type: metadataenrichment.K8sNamespaceLabelRule, Source: "lookup", Target: "namespace-key"},
					{Type: metadataenrichment.K8sWorkloadLabelRule, Source: "lookup", Target: "workload-key"},
					{Type: metadataenrichment.K8sPodLabelRule, Source: "lookup", Target: "pod-key"},
				},
				namespaceLabels:      map[string]string{"lookup": "namespace"},
				workloadLabels:       map[string]string{"lookup": "workload"},
				podLabels:            map[string]string{"lookup": "pod"},
				namespaceAnnotations: map[string]string{"metadata.dynatrace.com/namespace-key": "override"},
				workloadAnnotations:  map[string]string{"metadata.dynatrace.com/workload-key": "override"},
				podAnnotations:       map[string]string{"metadata.dynatrace.com/pod-key": "override"},
				expectAttributes:     map[string]string{"namespace-key": "override", "workload-key": "override", "pod-key": "override"},
			},
			{
				name:                 "annotations override resource attributes",
				namespaceAnnotations: map[string]string{"metadata.dynatrace.com/namespace-key": "override"},
				workloadAnnotations:  map[string]string{"metadata.dynatrace.com/workload-key": "override"},
				podAnnotations:       map[string]string{"metadata.dynatrace.com/pod-key": "override"},
				resourceAttributes:   map[string]string{"namespace-key": "base", "workload-key": "base", "pod-key": "base"},
				expectAttributes:     map[string]string{"namespace-key": "override", "workload-key": "override", "pod-key": "override"},
			},
			{
				name:                 "existing env overrides annotations",
				podEnvs:              []corev1.EnvVar{{Name: resourceattributes.OTelResourceAttributesEnv, Value: "namespace-key=override,workload-key=override,pod-key=override"}},
				namespaceAnnotations: map[string]string{"metadata.dynatrace.com/namespace-key": "base"},
				workloadAnnotations:  map[string]string{"metadata.dynatrace.com/workload-key": "base"},
				podAnnotations:       map[string]string{"metadata.dynatrace.com/pod-key": "base"},
				expectAttributes:     map[string]string{"namespace-key": "override", "workload-key": "override", "pod-key": "override"},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				testOTLP(t, clt, tt)
			})
		}

		t.Run("otlp exporter activegate", func(t *testing.T) {
			tenantUUID := uuid.NewString()

			dk := getReadyOTLPDynaKube()
			dk.Spec.ActiveGate.Capabilities = []activegate.CapabilityDisplayName{activegate.RoutingCapability.DisplayName}
			dk.Status.OneAgent.ConnectionInfo.TenantUUID = tenantUUID
			integrationtests.CreateDynakube(t, clt, dk)
			integrationtests.CreateKubernetesObject(t, clt, getOTLPExporterSecret(testNamespace))
			integrationtests.CreateKubernetesObject(t, clt, getOTLPExporterCertsSecret(testNamespace))

			pod := createPod(t, clt, nil)

			expectedService := fmt.Sprintf("%s-%s.%s", dk.Name, agconsts.MultiActiveGateName, testNamespace)
			expectedBase := fmt.Sprintf("https://%s/e/%s/api/v2/otlp", expectedService, tenantUUID)

			assert.Subset(
				t,
				pod.Spec.Containers[0].Env,
				[]corev1.EnvVar{
					{Name: exporter.OTLPMetricsEndpointEnv, Value: expectedBase + "/v1/metrics"},
					{Name: exporter.OTLPLogsEndpointEnv, Value: expectedBase + "/v1/logs"},
					{Name: exporter.OTLPTraceEndpointEnv, Value: expectedBase + "/v1/traces"},
				},
			)
		})

		t.Run("ignore unknown environment variables", func(t *testing.T) {
			dk := getReadyOTLPDynaKube()
			integrationtests.CreateDynakube(t, clt, dk)
			integrationtests.CreateKubernetesObject(t, clt, getOTLPExporterSecret(testNamespace))

			pod := createPod(t, clt, func(p *corev1.Pod) {
				p.Spec.Containers[0].Env = append(
					p.Spec.Containers[0].Env,
					corev1.EnvVar{Name: "OTLP_EXPORTER_OTLP_ENDPOINT", Value: "https://my-collector.example.com/otlp"},
					corev1.EnvVar{Name: "OTLP_EXPORTER_OTLP_PROTOCOL", Value: "http/protobuf"},
				)
			})

			assertContainsEnvs(
				t,
				pod.Spec.Containers[0].Env,
				exporter.DynatraceAPITokenEnv,
				exporter.OTLPTraceEndpointEnv,
				exporter.OTLPLogsEndpointEnv,
				exporter.OTLPMetricsEndpointEnv,
				exporter.OTLPTraceHeadersEnv,
				exporter.OTLPLogsHeadersEnv,
				exporter.OTLPMetricsHeadersEnv,
				"OTLP_EXPORTER_OTLP_ENDPOINT",
				"OTLP_EXPORTER_OTLP_PROTOCOL",
			)
		})

		t.Run("skip when known environment variables present", func(t *testing.T) {
			dk := getReadyOTLPDynaKube()
			integrationtests.CreateDynakube(t, clt, dk)
			integrationtests.CreateKubernetesObject(t, clt, getOTLPExporterSecret(testNamespace))

			pod := createPod(t, clt, func(p *corev1.Pod) {
				p.Spec.Containers[0].Env = append(
					p.Spec.Containers[0].Env,
					corev1.EnvVar{Name: exporter.OTLPExporterEndpointEnv, Value: "https://my-collector.example.com/otlp"},
					corev1.EnvVar{Name: exporter.OTLPExporterProtocolEnv, Value: "http/protobuf"},
				)
			})

			assert.Contains(t, pod.Spec.Containers[0].Env, corev1.EnvVar{Name: exporter.OTLPExporterEndpointEnv, Value: "https://my-collector.example.com/otlp"})
			assert.Contains(t, pod.Spec.Containers[0].Env, corev1.EnvVar{Name: exporter.OTLPExporterProtocolEnv, Value: "http/protobuf"})
			assertNotContainsEnvs(
				t,
				pod.Spec.Containers[0].Env,
				exporter.DynatraceAPITokenEnv,
				exporter.OTLPTraceEndpointEnv,
				exporter.OTLPLogsEndpointEnv,
				exporter.OTLPMetricsEndpointEnv,
				exporter.OTLPTraceHeadersEnv,
				exporter.OTLPLogsHeadersEnv,
				exporter.OTLPMetricsHeadersEnv,
			)
		})

		t.Run("override known environment variables", func(t *testing.T) {
			dk := getReadyOTLPDynaKube()
			dk.Spec.OTLPExporterConfiguration.OverrideEnvVars = new(true)
			integrationtests.CreateDynakube(t, clt, dk)
			integrationtests.CreateKubernetesObject(t, clt, getOTLPExporterSecret(testNamespace))

			pod := createPod(t, clt, func(p *corev1.Pod) {
				p.Spec.Containers[0].Env = append(
					p.Spec.Containers[0].Env,
					corev1.EnvVar{Name: exporter.OTLPExporterEndpointEnv, Value: "https://my-collector.example.com/otlp"},
					corev1.EnvVar{Name: exporter.OTLPExporterProtocolEnv, Value: "http/protobuf"},
				)
			})

			assert.Contains(t, pod.Spec.Containers[0].Env, corev1.EnvVar{Name: exporter.OTLPExporterEndpointEnv, Value: "https://my-collector.example.com/otlp"})
			assert.Contains(t, pod.Spec.Containers[0].Env, corev1.EnvVar{Name: exporter.OTLPExporterProtocolEnv, Value: "http/protobuf"})
			assertContainsEnvs(
				t,
				pod.Spec.Containers[0].Env,
				exporter.DynatraceAPITokenEnv,
				exporter.OTLPTraceEndpointEnv,
				exporter.OTLPLogsEndpointEnv,
				exporter.OTLPMetricsEndpointEnv,
				exporter.OTLPTraceHeadersEnv,
				exporter.OTLPLogsHeadersEnv,
				exporter.OTLPMetricsHeadersEnv,
			)
			assertNotContainsEnvs(
				t,
				pod.Spec.Containers[0].Env,
				"OTLP_EXPORTER_OTLP_ENDPOINT",
				"OTLP_EXPORTER_OTLP_PROTOCOL",
			)
		})
	})
}

type metadataJSONTestCase struct {
	name                 string
	rules                []metadataenrichment.Rule
	resourceAttributes   map[string]string
	oaAttributes         map[string]string
	otlpAttributes       map[string]string
	namespaceLabels      map[string]string
	namespaceAnnotations map[string]string
	workloadLabels       map[string]string
	workloadAnnotations  map[string]string
	podLabels            map[string]string
	podAnnotations       map[string]string
	expect               map[string]string
}

func testMetadataJSON(t *testing.T, clt client.Client, tt metadataJSONTestCase) {
	t.Helper()

	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "test"}}
	ns.Labels = maputils.MergeMap(tt.namespaceLabels, map[string]string{podmutator.InjectionInstanceLabel: "dynakube"})
	ns.Annotations = tt.namespaceAnnotations
	integrationtests.CreateKubernetesObject(t, clt, ns)

	dk := &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{Name: "dynakube", Namespace: testNamespace},
		Spec: dynakube.DynaKubeSpec{
			ResourceAttributes: tt.resourceAttributes,
			MetadataEnrichment: metadataenrichment.Spec{Enabled: new(true)},
		},
		Status: dynakube.DynaKubeStatus{
			KubernetesClusterMEID: testMEID,
			KubernetesClusterName: testClusterName,
			KubeSystemUUID:        testClusterUUID,
			MetadataEnrichment:    metadataenrichment.Status{Rules: tt.rules},
		},
	}
	if len(tt.oaAttributes) > 0 {
		dk.Spec.OneAgent.ApplicationMonitoring = &oneagent.ApplicationMonitoringSpec{AdditionalResourceAttributes: tt.oaAttributes}
		dk.Status.OneAgent.ConnectionInfo.TenantUUID = uuid.NewString()
		dk.Status.CodeModules.Version = "1.2.3"
	}
	if len(tt.otlpAttributes) > 0 {
		dk.Spec.OTLPExporterConfiguration = &otlpspec.ExporterConfigurationSpec{
			AdditionalResourceAttributes: tt.otlpAttributes,
			Signals:                      otlpspec.SignalConfiguration{Metrics: &otlpspec.MetricsSignal{}},
		}
		integrationtests.CreateKubernetesObject(t, clt, getOTLPExporterSecret("test"))
	}
	integrationtests.CreateDynakube(t, clt, dk)
	integrationtests.CreateKubernetesObject(t, clt, getBoostrapperSecret("test"))

	owner := getOwnerDeployment()
	owner.Namespace = ns.Name
	owner.Labels = tt.workloadLabels
	owner.Annotations = tt.workloadAnnotations
	integrationtests.CreateKubernetesObject(t, clt, owner)

	pod := createPod(t, clt, func(pod *corev1.Pod) {
		pod.Namespace = ns.Name
		pod.Labels = tt.podLabels
		pod.Annotations = tt.podAnnotations
		// this should never show up in the JSON annotation
		pod.Spec.Containers[0].Env = append(pod.Spec.Containers[0].Env, corev1.EnvVar{Name: resourceattributes.OTelResourceAttributesEnv, Value: "ignore=me"})
		require.NoError(t, controllerutil.SetControllerReference(owner, pod, scheme.Scheme))
	})

	var metadataJSON map[string]string
	require.NoError(t, json.Unmarshal([]byte(pod.Annotations["metadata.dynatrace.com"]), &metadataJSON))

	// present in the "metadata.dynatrace.com" JSON annotation regardless of test case, since every
	// test case uses the same pod/owner shape; merged with each test case's expect below.
	commonExpected := map[string]string{
		"k8s.workload.kind": "deployment",
		"k8s.workload.name": "test-deployment",
	}
	assert.Equal(t, maputils.MergeMap(commonExpected, tt.expect), metadataJSON)
}

func testBoostrapperArgs(t *testing.T, clt client.Client, withoutDeprecatedAnnotations bool) {
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
	integrationtests.CreateKubernetesObject(t, clt, getBoostrapperSecret(testNamespace))

	dummyOwner := getOwnerDeployment()
	integrationtests.CreateKubernetesObject(t, clt, dummyOwner)
	pod := createPod(t, clt, func(pod *corev1.Pod) {
		pod.Annotations = podMetadataAnnotations
		require.NoError(t, controllerutil.SetControllerReference(dummyOwner, pod, scheme.Scheme))
	})

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

type otlpTestCase struct {
	name                  string
	rules                 []metadataenrichment.Rule
	resourceAttributes    map[string]string
	otlpAttributes        map[string]string
	namespaceLabels       map[string]string
	namespaceAnnotations  map[string]string
	workloadLabels        map[string]string
	workloadAnnotations   map[string]string
	podLabels             map[string]string
	podAnnotations        map[string]string
	podEnvs               []corev1.EnvVar
	withoutDeprecatedKeys bool
	expectAttributes      map[string]string
}

func testOTLP(t *testing.T, clt client.Client, tt otlpTestCase) {
	t.Helper()

	integrationtests.CreateKubernetesObject(t, clt, getOTLPExporterSecret(testNamespace))

	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "otlp-test"}}
	ns.Labels = maputils.MergeMap(tt.namespaceLabels, map[string]string{podmutator.InjectionInstanceLabel: "dynakube"})
	ns.Annotations = tt.namespaceAnnotations
	integrationtests.CreateKubernetesObject(t, clt, ns)
	integrationtests.CreateKubernetesObject(t, clt, getOTLPExporterSecret(ns.Name))

	dk := getReadyOTLPDynaKube()
	if tt.withoutDeprecatedKeys {
		dk.Annotations[exp.EnrichmentEnableAttributesDTKubernetes] = "false"
	}
	dk.Spec.ResourceAttributes = tt.resourceAttributes
	dk.Spec.OTLPExporterConfiguration.AdditionalResourceAttributes = tt.otlpAttributes
	dk.Status.MetadataEnrichment.Rules = tt.rules
	// this should never show up the OTEL_RESOURCE_ATTRIBUTES environment variable
	dk.Spec.OneAgent.ApplicationMonitoring = &oneagent.ApplicationMonitoringSpec{AdditionalResourceAttributes: map[string]string{"ignore": "me"}}
	integrationtests.CreateDynakube(t, clt, dk)

	owner := getOwnerDeployment()
	owner.Namespace = ns.Name
	owner.Labels = tt.workloadLabels
	owner.Annotations = tt.workloadAnnotations
	integrationtests.CreateKubernetesObject(t, clt, owner)

	pod := createPod(t, clt, func(pod *corev1.Pod) {
		pod.Namespace = ns.Name
		pod.Annotations = tt.podAnnotations
		pod.Labels = tt.podLabels
		pod.Spec.Containers[0].Env = append(pod.Spec.Containers[0].Env, tt.podEnvs...)
		require.NoError(t, controllerutil.SetControllerReference(owner, pod, scheme.Scheme))
	})

	dtTokenEnv := k8senv.Find(pod.Spec.Containers[0].Env, exporter.DynatraceAPITokenEnv)
	require.NotNil(t, dtTokenEnv, "DT_API_TOKEN missing")
	require.NotNil(t, dtTokenEnv.ValueFrom)
	require.NotNil(t, dtTokenEnv.ValueFrom.SecretKeyRef)
	assert.Equal(t, consts.OTLPExporterSecretName, dtTokenEnv.ValueFrom.SecretKeyRef.Name)
	assert.Equal(t, token.DataIngestKey, dtTokenEnv.ValueFrom.SecretKeyRef.Key)

	baseEndpoint := apiURL + "/v2/otlp"
	assert.Subset(
		t,
		pod.Spec.Containers[0].Env,
		[]corev1.EnvVar{
			// Headers env vars should reference DT_API_TOKEN via authorization header literal
			{Name: exporter.OTLPMetricsHeadersEnv, Value: exporter.OTLPAuthorizationHeader},
			{Name: exporter.OTLPLogsHeadersEnv, Value: exporter.OTLPAuthorizationHeader},
			{Name: exporter.OTLPTraceHeadersEnv, Value: exporter.OTLPAuthorizationHeader},
			// Endpoint base constructed by BuildOTLPEndpoint(apiURL) => apiURL + /v2/otlp plus per-signal suffix
			{Name: exporter.OTLPMetricsEndpointEnv, Value: baseEndpoint + "/v1/metrics"},
			{Name: exporter.OTLPLogsEndpointEnv, Value: baseEndpoint + "/v1/logs"},
			{Name: exporter.OTLPTraceEndpointEnv, Value: baseEndpoint + "/v1/traces"},
			// metrics temporality preference should be set to delta
			{Name: exporter.OTLPMetricsExporterTemporalityPreference, Value: exporter.OTLPMetricsExporterAggregationTemporalityDelta},
			{
				Name: "K8S_PODUID",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						APIVersion: "v1",
						FieldPath:  "metadata.uid",
					},
				},
			},
			{
				Name: "K8S_PODNAME",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						APIVersion: "v1",
						FieldPath:  "metadata.name",
					},
				},
			},
			{
				Name: "K8S_NODE_NAME",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						APIVersion: "v1",
						FieldPath:  "spec.nodeName",
					},
				},
			},
		},
	)

	expectAttributes := map[string]string{
		// workload info
		"k8s.namespace.name":           ns.Name,
		"k8s.pod.uid":                  "$(K8S_PODUID)",
		"k8s.pod.name":                 "$(K8S_PODNAME)",
		"k8s.node.name":                "$(K8S_NODE_NAME)",
		"k8s.cluster.name":             testClusterName,
		"k8s.cluster.uid":              testClusterUUID,
		"k8s.container.name":           "app",
		"k8s.workload.name":            owner.Name,
		"k8s.workload.kind":            "deployment",
		"dt.entity.kubernetes_cluster": testMEID,
	}
	if !tt.withoutDeprecatedKeys {
		expectAttributes[attributes.DeprecatedClusterIDKey] = expectAttributes["k8s.cluster.uid"]
		expectAttributes[attributes.DeprecatedWorkloadNameKey] = expectAttributes["k8s.workload.name"]
		expectAttributes[attributes.DeprecatedWorkloadKindKey] = expectAttributes["k8s.workload.kind"]
	}
	maps.Copy(expectAttributes, tt.expectAttributes)

	gotAttributes, envVarFound := resourceattributes.NewAttributesFromEnv(pod.Spec.Containers[0].Env, resourceattributes.OTelResourceAttributesEnv)
	require.True(t, envVarFound, "OTEL_RESOURCE_ATTRIBUTES missing")
	assert.Equal(t, expectAttributes, gotAttributes)
}

func assertContainsEnvs(t *testing.T, envs []corev1.EnvVar, names ...string) {
	t.Helper()

	for _, name := range names {
		assert.Truef(t, k8senv.Contains(envs, name), "should contain %s", name)
	}
}

func assertNotContainsEnvs(t *testing.T, envs []corev1.EnvVar, names ...string) {
	t.Helper()

	for _, name := range names {
		assert.Falsef(t, k8senv.Contains(envs, name), "should not contain %s", name)
	}
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

func getOwnerDeployment() *appsv1.Deployment {
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

func getReadyCNFSDynaKube() *dynakube.DynaKube {
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
}

func getReadyOTLPDynaKube() *dynakube.DynaKube {
	return &dynakube.DynaKube{
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
		Status: dynakube.DynaKubeStatus{
			KubernetesClusterMEID: testMEID,
			KubernetesClusterName: testClusterName,
			KubeSystemUUID:        testClusterUUID,
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
			dynakube.TLSCertKey: []byte("ag-cert-data"),
		},
	}
}
