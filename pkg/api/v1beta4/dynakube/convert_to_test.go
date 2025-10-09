package dynakube

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	dynakubelatest "github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/communication"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/image"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/value"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube/kspm"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube/logmonitoring"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube/telemetryingest"
	registryv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
)

var testTime = metav1.Now()

func TestConvertTo(t *testing.T) {
	t.Run("migrate from v1beta4 to latest", func(t *testing.T) {
		from := getOldDynakubeBase()
		to := dynakubelatest.DynaKube{}

		err := from.ConvertTo(&to)
		require.NoError(t, err)

		compareBase(t, from, to)
	})

	t.Run("migrate metadata-enrichment from v1beta4 to latest", func(t *testing.T) {
		from := getOldDynakubeBase()
		to := dynakubelatest.DynaKube{}

		err := from.ConvertTo(&to)
		require.NoError(t, err)

		assert.False(t, to.MetadataEnrichment().IsEnabled())
		compareBase(t, from, to)
	})

	t.Run("migrate host-monitoring from v1beta4 to latest", func(t *testing.T) {
		from := getOldDynakubeBase()
		hostSpec := getOldHostInjectSpec()
		from.Spec.OneAgent.HostMonitoring = &hostSpec
		to := dynakubelatest.DynaKube{}

		err := from.ConvertTo(&to)
		require.NoError(t, err)

		compareHostInjectSpec(t, *from.Spec.OneAgent.HostMonitoring, *to.Spec.OneAgent.HostMonitoring, to.RemovedFields())
		compareBase(t, from, to)
		assert.False(t, to.MetadataEnrichment().IsEnabled())
	})

	t.Run("migrate classic-fullstack from v1beta4 to latest", func(t *testing.T) {
		from := getOldDynakubeBase()
		hostSpec := getOldHostInjectSpec()
		from.Spec.OneAgent.ClassicFullStack = &hostSpec
		to := dynakubelatest.DynaKube{}

		err := from.ConvertTo(&to)
		require.NoError(t, err)

		assert.Nil(t, to.Spec.OneAgent.CloudNativeFullStack)
		assert.Nil(t, to.Spec.OneAgent.ApplicationMonitoring)
		assert.Nil(t, to.Spec.OneAgent.HostMonitoring)
		compareHostInjectSpec(t, *from.Spec.OneAgent.ClassicFullStack, *to.Spec.OneAgent.ClassicFullStack, to.RemovedFields())
		compareBase(t, from, to)
		assert.False(t, to.MetadataEnrichment().IsEnabled())
	})

	t.Run("migrate cloud-native from v1beta4 to latest", func(t *testing.T) {
		from := getOldDynakubeBase()
		spec := getOldCloudNativeSpec()
		from.Spec.OneAgent.CloudNativeFullStack = &spec
		to := dynakubelatest.DynaKube{}

		err := from.ConvertTo(&to)
		require.NoError(t, err)

		assert.Nil(t, to.Spec.OneAgent.ClassicFullStack)
		assert.Nil(t, to.Spec.OneAgent.ApplicationMonitoring)
		assert.Nil(t, to.Spec.OneAgent.HostMonitoring)
		compareCloudNativeSpec(t, *from.Spec.OneAgent.CloudNativeFullStack, *to.Spec.OneAgent.CloudNativeFullStack, to.RemovedFields())
		compareBase(t, from, to)
	})

	t.Run("migrate application-monitoring from v1beta4 to latest", func(t *testing.T) {
		from := getOldDynakubeBase()
		appSpec := getOldApplicationMonitoringSpec()
		from.Spec.OneAgent.ApplicationMonitoring = &appSpec
		to := dynakubelatest.DynaKube{}

		err := from.ConvertTo(&to)
		require.NoError(t, err)

		assert.Nil(t, to.Spec.OneAgent.ClassicFullStack)
		assert.Nil(t, to.Spec.OneAgent.CloudNativeFullStack)
		assert.Nil(t, to.Spec.OneAgent.HostMonitoring)
		compareApplicationMonitoringSpec(t, *from.Spec.OneAgent.ApplicationMonitoring, *to.Spec.OneAgent.ApplicationMonitoring)
		compareBase(t, from, to)
	})

	t.Run("migrate activegate from v1beta4 to latest", func(t *testing.T) {
		from := getOldDynakubeBase()
		agSpec := getOldActiveGateSpec()
		from.Spec.ActiveGate = agSpec
		to := dynakubelatest.DynaKube{}

		err := from.ConvertTo(&to)
		require.NoError(t, err)

		compareActiveGateSpec(t, from.Spec.ActiveGate, to.Spec.ActiveGate)
		compareBase(t, from, to)
		assert.False(t, to.MetadataEnrichment().IsEnabled())
	})

	t.Run("migrate extensions from v1beta4 to latest", func(t *testing.T) {
		from := getOldDynakubeBase()
		from.Spec.Extensions = &ExtensionsSpec{}
		to := dynakubelatest.DynaKube{}

		err := from.ConvertTo(&to)
		require.NoError(t, err)

		assert.NotNil(t, to.Spec.Extensions)
		assert.NotNil(t, to.Spec.Extensions.PrometheusSpec)
		compareBase(t, from, to)
	})

	t.Run("migrate log-monitoring from v1beta4 to latest", func(t *testing.T) {
		from := getOldDynakubeBase()
		from.Spec.LogMonitoring = getOldLogMonitoringSpec()
		to := dynakubelatest.DynaKube{}

		err := from.ConvertTo(&to)
		require.NoError(t, err)

		compareLogMonitoringSpec(t, from.Spec.LogMonitoring, to.Spec.LogMonitoring)
		compareBase(t, from, to)
	})

	t.Run("migrate kspm from v1beta4 to latest", func(t *testing.T) {
		from := getOldDynakubeBase()
		from.Spec.Kspm = &kspm.Spec{}
		to := dynakubelatest.DynaKube{}

		err := from.ConvertTo(&to)
		require.NoError(t, err)

		assert.NotNil(t, to.Spec.Kspm)
		assert.Equal(t, []string{"/"}, to.Spec.Kspm.MappedHostPaths)
		compareBase(t, from, to)
	})

	t.Run("migrate extensions templates from v1beta4 to latest", func(t *testing.T) {
		from := getOldDynakubeBase()
		from.Spec.Templates.OpenTelemetryCollector = getOldOpenTelemetryTemplateSpec()
		from.Spec.Templates.ExtensionExecutionController = getOldExtensionExecutionControllerSpec()

		to := dynakubelatest.DynaKube{}

		err := from.ConvertTo(&to)
		require.NoError(t, err)

		compareOpenTelemetryTemplateSpec(t, from.Spec.Templates.OpenTelemetryCollector, to.Spec.Templates.OpenTelemetryCollector)
		compareExtensionsExecutionControllerTemplateSpec(t, from.Spec.Templates.ExtensionExecutionController, to.Spec.Templates.ExtensionExecutionController)

		compareBase(t, from, to)
	})

	t.Run("migrate log-monitoring templates from v1beta4 to latest", func(t *testing.T) {
		from := getOldDynakubeBase()
		from.Spec.Templates.LogMonitoring = getOldLogMonitoringTemplateSpec()

		to := dynakubelatest.DynaKube{}

		err := from.ConvertTo(&to)
		require.NoError(t, err)

		compareLogMonitoringTemplateSpec(t, from.Spec.Templates.LogMonitoring, to.Spec.Templates.LogMonitoring)
		compareBase(t, from, to)
	})

	t.Run("migrate kspm templates from v1beta4 to latest", func(t *testing.T) {
		from := getOldDynakubeBase()
		from.Spec.Templates.KspmNodeConfigurationCollector = getOldNodeConfigurationCollectorTemplateSpec()

		to := dynakubelatest.DynaKube{}

		err := from.ConvertTo(&to)
		require.NoError(t, err)

		compareNodeConfigurationCollectorTemplateSpec(t, from.Spec.Templates.KspmNodeConfigurationCollector, to.Spec.Templates.KspmNodeConfigurationCollector)
		compareBase(t, from, to)
	})

	t.Run("migrate status from v1beta4 to latest", func(t *testing.T) {
		from := getOldDynakubeBase()
		from.Status = getOldStatus()
		to := dynakubelatest.DynaKube{}

		err := from.ConvertTo(&to)
		require.NoError(t, err)

		compareStatus(t, from.Status, to.Status)
	})

	t.Run("migrate hostGroup", func(t *testing.T) {
		from := getOldDynakubeBase()
		from.Status = getOldStatus()
		to := dynakubelatest.DynaKube{}

		err := from.ConvertTo(&to)
		require.NoError(t, err)

		assert.Equal(t, from.Spec.OneAgent.HostGroup, to.Spec.OneAgent.HostGroup)
	})

	t.Run("migrate telemetryIngest", func(t *testing.T) {
		from := getOldDynakubeBase()
		from.Status = getOldStatus()
		to := dynakubelatest.DynaKube{}

		err := from.ConvertTo(&to)
		require.NoError(t, err)

		require.NotNil(t, to.Spec.TelemetryIngest)
		assert.Equal(t, from.Spec.TelemetryIngest.Protocols, to.Spec.TelemetryIngest.Protocols)
		assert.Equal(t, from.Spec.TelemetryIngest.ServiceName, to.Spec.TelemetryIngest.ServiceName)
		assert.Equal(t, from.Spec.TelemetryIngest.TLSRefName, to.Spec.TelemetryIngest.TLSRefName)
	})
}

func getTestNamespaceSelector() metav1.LabelSelector {
	return metav1.LabelSelector{
		MatchLabels: map[string]string{
			"match-label-key": "match-label-value",
		},
		MatchExpressions: []metav1.LabelSelectorRequirement{
			{
				Key:      "match-expression-key",
				Operator: "In",
				Values:   []string{"match-expression-value-test-1", "match-expression-value-test-2"},
			},
		},
	}
}

func getOldDynakubeBase() DynaKube {
	return DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "name",
			Namespace: "namespace",
			Annotations: map[string]string{
				exp.AGIgnoreProxyKey:               "true", //nolint:staticcheck
				exp.AGAutomaticK8sAPIMonitoringKey: "true",
			},
			Labels: map[string]string{
				"label": "label-value",
			},
		},
		Spec: DynaKubeSpec{
			OneAgent:         oneagent.Spec{HostGroup: "hostgroup-value"},
			APIURL:           "api-url",
			Tokens:           "token",
			CustomPullSecret: "pull-secret",
			EnableIstio:      true,
			SkipCertCheck:    true,
			Proxy: &value.Source{
				Value:     "proxy-value",
				ValueFrom: "proxy-from",
			},
			TrustedCAs:                   "trusted-ca",
			NetworkZone:                  "network-zone",
			DynatraceAPIRequestThreshold: ptr.To(uint16(42)),
			MetadataEnrichment: MetadataEnrichment{
				Enabled: ptr.To(false),
			},
			TelemetryIngest: &telemetryingest.Spec{
				ServiceName: "telemetry-ingest-service-name",
				TLSRefName:  "telemetry-ingest-tls-secret-name",
				Protocols:   []string{"protocol1", "protocol2"},
			},
		},
	}
}

func getOldHostInjectSpec() oneagent.HostInjectSpec {
	return oneagent.HostInjectSpec{
		Version: "host-inject-version",
		Image:   "host-inject-image",
		Tolerations: []corev1.Toleration{
			{Key: "host-inject-toleration-key", Operator: "In", Value: "host-inject-toleration-value"},
		},
		AutoUpdate: ptr.To(false),
		DNSPolicy:  corev1.DNSClusterFirstWithHostNet,
		Annotations: map[string]string{
			"host-inject-annotation-key": "host-inject-annotation-value",
		},
		Labels: map[string]string{
			"host-inject-label-key": "host-inject-label-value",
		},
		Env: []corev1.EnvVar{
			{Name: "host-inject-env-1", Value: "host-inject-env-value-1", ValueFrom: &corev1.EnvVarSource{
				ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
					Key: "host-inject-env-from-1",
				},
			}},
			{Name: "host-inject-env-2", Value: "host-inject-env-value-2", ValueFrom: &corev1.EnvVarSource{
				ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
					Key: "host-inject-env-from-2",
				},
			}},
		},
		NodeSelector: map[string]string{
			"host-inject-node-selector-key": "host-inject-node-selector-value",
		},
		PriorityClassName: "host-inject-priority-class",
		Args: []string{
			"host-inject-arg-1",
			"host-inject-arg-2",
		},
		OneAgentResources: corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceCPU: *resource.NewScaledQuantity(1, 1)},
		},
		SecCompProfile: "seccomp",
	}
}

func getOldAppInjectionSpec() oneagent.AppInjectionSpec {
	return oneagent.AppInjectionSpec{
		InitResources: &corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceCPU: *resource.NewScaledQuantity(2, 1)},
		},
		CodeModulesImage:  "app-injection-image",
		NamespaceSelector: getTestNamespaceSelector(),
	}
}

func getOldCloudNativeSpec() oneagent.CloudNativeFullStackSpec {
	return oneagent.CloudNativeFullStackSpec{
		AppInjectionSpec: getOldAppInjectionSpec(),
		HostInjectSpec:   getOldHostInjectSpec(),
	}
}

func getOldApplicationMonitoringSpec() oneagent.ApplicationMonitoringSpec {
	return oneagent.ApplicationMonitoringSpec{
		AppInjectionSpec: getOldAppInjectionSpec(),
		Version:          "app-monitoring-version",
	}
}

func getOldActiveGateSpec() activegate.Spec {
	return activegate.Spec{
		DNSPolicy: corev1.DNSClusterFirstWithHostNet,
		Annotations: map[string]string{
			"activegate-annotation-key": "activegate-annotation-value",
		},
		PersistentVolumeClaim: getPersistentVolumeClaimSpec(),
		TLSSecretName:         "activegate-tls-secret-name",
		PriorityClassName:     "activegate-priority-class-name",
		Capabilities: []activegate.CapabilityDisplayName{
			activegate.DynatraceAPICapability.DisplayName,
			activegate.KubeMonCapability.DisplayName,
			activegate.MetricsIngestCapability.DisplayName,
		},
		CapabilityProperties: activegate.CapabilityProperties{
			Labels: map[string]string{
				"activegate-label-key": "activegate-label-value",
			},
			Env: []corev1.EnvVar{
				{Name: "host-inject-env-1", Value: "activegate-env-value-1", ValueFrom: &corev1.EnvVarSource{
					ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
						Key: "activegate-env-from-1",
					},
				}},
				{Name: "activegate-env-2", Value: "activegate-env-value-2", ValueFrom: &corev1.EnvVarSource{
					ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
						Key: "activegate-env-from-2",
					},
				}},
			},
			NodeSelector: map[string]string{
				"activegate-node-selector-key": "activegate-node-selector-value",
			},
			Image:    "activegate-image",
			Replicas: ptr.To(int32(42)),
			Group:    "activegate-group",
			CustomProperties: &value.Source{
				Value:     "activegate-cp-value",
				ValueFrom: "activegate-cp-value-from",
			},
			Resources: corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					corev1.ResourceCPU: *resource.NewScaledQuantity(3, 1)},
			},
			Tolerations: []corev1.Toleration{
				{Key: "activegate-toleration-key", Operator: "In", Value: "activegate-toleration-value"},
			},
			TopologySpreadConstraints: []corev1.TopologySpreadConstraint{
				{MaxSkew: 1},
			},
		},
	}
}

func getOldLogMonitoringSpec() *logmonitoring.Spec {
	oldSpec := logmonitoring.Spec{
		IngestRuleMatchers: make([]logmonitoring.IngestRuleMatchers, 0),
	}

	oldSpec.IngestRuleMatchers = append(oldSpec.IngestRuleMatchers, logmonitoring.IngestRuleMatchers{
		Attribute: "attribute1",
		Values:    []string{"matcher1", "matcher2", "matcher3"},
	})

	oldSpec.IngestRuleMatchers = append(oldSpec.IngestRuleMatchers, logmonitoring.IngestRuleMatchers{
		Attribute: "attribute2",
		Values:    []string{"matcher1", "matcher2", "matcher3"},
	})

	oldSpec.IngestRuleMatchers = append(oldSpec.IngestRuleMatchers, logmonitoring.IngestRuleMatchers{
		Attribute: "attribute3",
		Values:    []string{"matcher1", "matcher2", "matcher3"},
	})

	return &oldSpec
}

func getOldOpenTelemetryTemplateSpec() OpenTelemetryCollectorSpec {
	return OpenTelemetryCollectorSpec{
		Labels: map[string]string{
			"otelc-label-key1": "otelc-label-value1",
			"otelc-label-key2": "otelc-label-value2",
		},
		Annotations: map[string]string{
			"otelc-annotation-key1": "otelc-annotation-value1",
			"otelc-annotation-key2": "otelc-annotation-value2",
		},
		Replicas: ptr.To(int32(42)),
		ImageRef: image.Ref{
			Repository: "image-repo.repohost.test/repo",
			Tag:        "image-tag",
		},
		TLSRefName: "tls-ref-name",
		Resources: corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceCPU: *resource.NewScaledQuantity(3, 1),
			},
			Requests: corev1.ResourceList{
				corev1.ResourceCPU: *resource.NewScaledQuantity(3, 1),
			},
			Claims: []corev1.ResourceClaim{{
				Name:    "claim-name",
				Request: "claim-request",
			}},
		},
		Tolerations: []corev1.Toleration{
			{Key: "otelc-toleration-key", Operator: "In", Value: "otelc-toleration-value"},
		},
		TopologySpreadConstraints: []corev1.TopologySpreadConstraint{
			{MaxSkew: 1},
		},
	}
}

func getOldExtensionExecutionControllerSpec() ExtensionExecutionControllerSpec {
	return ExtensionExecutionControllerSpec{
		PersistentVolumeClaim: getPersistentVolumeClaimSpec(),
		Labels: map[string]string{
			"eec-label-key1": "eec-label-value1",
			"eec-label-key2": "eec-label-value2",
		},
		Annotations: map[string]string{
			"eec-annotation-key1": "eec-annotation-value1",
			"eec-annotation-key2": "eec-annotation-value2",
		},
		ImageRef: image.Ref{
			Repository: "image-repo.repohost.test/repo",
			Tag:        "image-tag",
		},
		TLSRefName: "tls-ref-name",
		Resources: corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceCPU: *resource.NewScaledQuantity(3, 1),
			},
			Requests: corev1.ResourceList{
				corev1.ResourceCPU: *resource.NewScaledQuantity(3, 1),
			},
			Claims: []corev1.ResourceClaim{{
				Name:    "claim-name",
				Request: "claim-request",
			}},
		},
		Tolerations: []corev1.Toleration{
			{Key: "otelc-toleration-key", Operator: "In", Value: "otelc-toleration-value"},
		},
		TopologySpreadConstraints: []corev1.TopologySpreadConstraint{
			{MaxSkew: 1},
		},
		CustomConfig:                "custom-eec-config",
		CustomExtensionCertificates: "custom-eec-certificates",
		UseEphemeralVolume:          true,
	}
}

func getOldLogMonitoringTemplateSpec() *logmonitoring.TemplateSpec {
	return &logmonitoring.TemplateSpec{
		Labels: map[string]string{
			"logagent-label-key1": "logagent-label-value1",
			"logagent-label-key2": "logagent-label-value2",
		},
		Annotations: map[string]string{
			"logagent-annotation-key1": "logagent-annotation-value1",
			"logagent-annotation-key2": "logagent-annotation-value2",
		},
		NodeSelector: map[string]string{
			"selector1": "node1",
			"selector2": "node2",
		},
		ImageRef: image.Ref{
			Repository: "image-repo.repohost.test/repo",
			Tag:        "image-tag",
		},
		DNSPolicy:         "dns-policy",
		PriorityClassName: "priority-class-name",
		SecCompProfile:    "sec-comp-profile",
		Resources: corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceCPU: *resource.NewScaledQuantity(3, 1),
			},
			Requests: corev1.ResourceList{
				corev1.ResourceCPU: *resource.NewScaledQuantity(3, 1),
			},
			Claims: []corev1.ResourceClaim{{
				Name:    "claim-name",
				Request: "claim-request",
			}},
		},
		Tolerations: []corev1.Toleration{
			{Key: "otelc-toleration-key", Operator: "In", Value: "otelc-toleration-value"},
		},
		Args: []string{"--log-level", "debug", "--log-format", "json"},
	}
}

func getOldNodeConfigurationCollectorTemplateSpec() kspm.NodeConfigurationCollectorSpec {
	return kspm.NodeConfigurationCollectorSpec{
		UpdateStrategy: &appsv1.DaemonSetUpdateStrategy{
			Type: "daemonset-update-strategy-type",
			RollingUpdate: &appsv1.RollingUpdateDaemonSet{
				MaxUnavailable: &intstr.IntOrString{
					Type:   0,
					IntVal: 42,
				},
				MaxSurge: &intstr.IntOrString{
					Type:   1,
					StrVal: "42",
				},
			},
		},
		Labels: map[string]string{
			"ncc-label-key1": "ncc-label-value1",
			"ncc-label-key2": "ncc-label-value2",
		},
		Annotations: map[string]string{
			"ncc-annotation-key1": "ncc-annotation-value1",
			"ncc-annotation-key2": "ncc-annotation-value2",
		},
		NodeSelector: map[string]string{
			"selector1": "node1",
			"selector2": "node2",
		},
		ImageRef: image.Ref{
			Repository: "image-repo.repohost.test/repo",
			Tag:        "image-tag",
		},
		PriorityClassName: "priority-class-name",
		Resources: corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceCPU: *resource.NewScaledQuantity(3, 1),
			},
			Requests: corev1.ResourceList{
				corev1.ResourceCPU: *resource.NewScaledQuantity(3, 1),
			},
			Claims: []corev1.ResourceClaim{{
				Name:    "claim-name",
				Request: "claim-request",
			}},
		},
		NodeAffinity: corev1.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
				NodeSelectorTerms: []corev1.NodeSelectorTerm{{
					MatchExpressions: []corev1.NodeSelectorRequirement{
						{
							Key:      "node-selector-match-key",
							Operator: "node-selector-match-operator",
							Values:   []string{"node-match-value-1", "node-match-value2"},
						},
					},
					MatchFields: []corev1.NodeSelectorRequirement{
						{
							Key:      "node-selector-field-key",
							Operator: "node-selector-field-operator",
							Values:   []string{"node-field-value-1", "node-field-value2"},
						},
					},
				}},
			},
			PreferredDuringSchedulingIgnoredDuringExecution: nil,
		},
		Tolerations: []corev1.Toleration{
			{Key: "otelc-toleration-key", Operator: "In", Value: "otelc-toleration-value"},
		},
		Args: []string{"--log-level", "debug", "--log-format", "json"},
		Env: []corev1.EnvVar{
			{
				Name:  "ENV1",
				Value: "VAL1",
			},
			{
				Name:  "ENV2",
				Value: "VAL2",
			},
			{
				Name:  "ENV2",
				Value: "VAL2",
			},
		},
	}
}

func getOldStatus() DynaKubeStatus {
	return DynaKubeStatus{
		OneAgent: oneagent.Status{
			VersionStatus: status.VersionStatus{
				ImageID:            "oa-image-id",
				Version:            "oa-version",
				Type:               "oa-image-type",
				Source:             status.CustomImageVersionSource,
				LastProbeTimestamp: &testTime,
			},
			Instances: map[string]oneagent.Instance{
				"oa-instance-key-1": {
					PodName:   "oa-instance-pod-1",
					IPAddress: "oa-instance-ip-1",
				},
				"oa-instance-key-2": {
					PodName:   "oa-instance-pod-2",
					IPAddress: "oa-instance-ip-2",
				},
			},
			LastInstanceStatusUpdate: &testTime,
			Healthcheck: &registryv1.HealthConfig{
				Test: []string{"oa-health-check-test"},
			},
			ConnectionInfoStatus: oneagent.ConnectionInfoStatus{
				ConnectionInfo: communication.ConnectionInfo{
					LastRequest: testTime,
					TenantUUID:  "oa-tenant-uuid",
					Endpoints:   "oa-endpoints",
				},
				CommunicationHosts: []oneagent.CommunicationHostStatus{
					{
						Protocol: "oa-protocol-1",
						Host:     "oa-host-1",
						Port:     1,
					},
					{
						Protocol: "oa-protocol-2",
						Host:     "oa-host-2",
						Port:     2,
					},
				},
			},
		},
		ActiveGate: activegate.Status{
			VersionStatus: status.VersionStatus{
				ImageID:            "ag-image-id",
				Version:            "ag-version",
				Type:               "ag-image-type",
				Source:             status.CustomVersionVersionSource,
				LastProbeTimestamp: &testTime,
			},
		},
		CodeModules: oneagent.CodeModulesStatus{
			VersionStatus: status.VersionStatus{
				ImageID:            "cm-image-id",
				Version:            "cm-version",
				Type:               "cm-image-type",
				Source:             status.TenantRegistryVersionSource,
				LastProbeTimestamp: &testTime,
			},
		},
		DynatraceAPI: DynatraceAPIStatus{
			LastTokenScopeRequest: testTime,
		},
		Conditions: []metav1.Condition{
			{
				Type:               "condition-type-1",
				Status:             "condition-status-1",
				Reason:             "condition-reason-1",
				LastTransitionTime: testTime,
			},
			{
				Type:               "condition-type-2",
				Status:             "condition-status-2",
				Reason:             "condition-reason-2",
				LastTransitionTime: testTime,
			},
		},
		KubeSystemUUID:   "kube-system-uuid",
		Phase:            status.Deploying,
		UpdatedTimestamp: testTime,
	}
}
