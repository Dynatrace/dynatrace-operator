package dynakube

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/address"
	registryv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var testTime = metav1.Now()

func TestConvertTo(t *testing.T) {
	t.Run("migrate from v1beta1 to v1beta2", func(t *testing.T) {
		to := dynakube.DynaKube{}
		from := getOldDynakubeBase()

		from.ConvertTo(&to)

		compareBase(t, from, to)
	})

	t.Run("migrate host-monitoring from v1beta1 to v1beta2", func(t *testing.T) {
		from := getOldDynakubeBase()
		hostSpec := getOldHostInjectSpec()
		from.Spec.OneAgent.HostMonitoring = &hostSpec
		to := dynakube.DynaKube{}

		from.ConvertTo(&to)

		compareHostInjectSpec(t, *from.Spec.OneAgent.HostMonitoring, *to.Spec.OneAgent.HostMonitoring)
		compareMovedFields(t, from, to)
		assert.False(t, to.MetadataEnrichmentEnabled())
	})

	t.Run("migrate classic-fullstack from v1beta1 to v1beta2", func(t *testing.T) {
		from := getOldDynakubeBase()
		hostSpec := getOldHostInjectSpec()
		from.Spec.OneAgent.ClassicFullStack = &hostSpec
		to := dynakube.DynaKube{}

		from.ConvertTo(&to)

		compareHostInjectSpec(t, *from.Spec.OneAgent.ClassicFullStack, *to.Spec.OneAgent.ClassicFullStack)
		compareMovedFields(t, from, to)
		assert.False(t, to.MetadataEnrichmentEnabled())
	})

	t.Run("migrate cloud-native from v1beta1 to v1beta2", func(t *testing.T) {
		from := getOldDynakubeBase()
		spec := getOldCloudNativeSpec()
		from.Spec.OneAgent.CloudNativeFullStack = &spec
		to := dynakube.DynaKube{}

		from.ConvertTo(&to)

		compareCloudNativeSpec(t, *from.Spec.OneAgent.CloudNativeFullStack, *to.Spec.OneAgent.CloudNativeFullStack)
		compareMovedFields(t, from, to)
	})

	t.Run("migrate application-monitoring from v1beta1 to v1beta2", func(t *testing.T) {
		from := getOldDynakubeBase()
		appSpec := getOldApplicationMonitoringSpec()
		from.Spec.OneAgent.ApplicationMonitoring = &appSpec
		to := dynakube.DynaKube{}

		from.ConvertTo(&to)

		compareApplicationMonitoringSpec(t, *from.Spec.OneAgent.ApplicationMonitoring, *to.Spec.OneAgent.ApplicationMonitoring)
	})

	t.Run("migrate activegate from v1beta2 to v1beta1", func(t *testing.T) {
		from := getOldDynakubeBase()
		agSpec := getOldActiveGateSpec()
		from.Spec.ActiveGate = agSpec
		to := dynakube.DynaKube{}

		from.ConvertTo(&to)

		compareActiveGateSpec(t, from.Spec.ActiveGate, to.Spec.ActiveGate)
		assert.False(t, to.MetadataEnrichmentEnabled())
	})

	t.Run("migrate status from v1beta2 to v1beta1", func(t *testing.T) {
		from := getOldDynakubeBase()
		from.Status = getOldStatus()
		to := dynakube.DynaKube{}

		from.ConvertTo(&to)

		compareStatus(t, from.Status, to.Status)
	})

	t.Run("migrate hostGroup", func(t *testing.T) {
		from := getOldDynakubeBase()
		from.Status = getOldStatus()
		to := dynakube.DynaKube{}

		from.ConvertTo(&to)

		assert.Equal(t, from.Spec.OneAgent.HostGroup, to.Spec.OneAgent.HostGroup)
	})
}

func getMovedFeatureFlagList() []string {
	return []string{
		AnnotationFeatureApiRequestThreshold,
		AnnotationFeatureOneAgentSecCompProfile,
		AnnotationFeatureMetadataEnrichment,
	}
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
				AnnotationFeatureApiRequestThreshold:       "42",
				AnnotationFeatureOneAgentSecCompProfile:    "seccomp",
				AnnotationFeatureMetadataEnrichment:        "false",
				AnnotationFeatureActiveGateIgnoreProxy:     "true",
				AnnotationFeatureAutomaticK8sApiMonitoring: "true",
				AnnotationFeatureMaxFailedCsiMountAttempts: "9",
			},
			Labels: map[string]string{
				"label": "label-value",
			},
		},
		Spec: DynaKubeSpec{
			OneAgent:         OneAgentSpec{HostGroup: "hostgroup-value"},
			APIURL:           "api-url",
			Tokens:           "token",
			CustomPullSecret: "pull-secret",
			EnableIstio:      true,
			SkipCertCheck:    true,
			Proxy: &DynaKubeProxy{
				Value:     "proxy-value",
				ValueFrom: "proxy-from",
			},
			TrustedCAs:        "trusted-ca",
			NetworkZone:       "network-zone",
			NamespaceSelector: getTestNamespaceSelector(),
		},
	}
}

func getOldHostInjectSpec() HostInjectSpec {
	return HostInjectSpec{
		Version: "host-inject-version",
		Image:   "host-inject-image",
		Tolerations: []corev1.Toleration{
			{Key: "host-inject-toleration-key", Operator: "In", Value: "host-inject-toleration-value"},
		},
		AutoUpdate: address.Of(false),
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
	}
}

func getOldAppInjectionSpec() AppInjectionSpec {
	return AppInjectionSpec{
		InitResources: &corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceCPU: *resource.NewScaledQuantity(2, 1)},
		},
		CodeModulesImage: "app-injection-image",
	}
}

func getOldCloudNativeSpec() CloudNativeFullStackSpec {
	return CloudNativeFullStackSpec{
		AppInjectionSpec: getOldAppInjectionSpec(),
		HostInjectSpec:   getOldHostInjectSpec(),
	}
}

func getOldApplicationMonitoringSpec() ApplicationMonitoringSpec {
	return ApplicationMonitoringSpec{
		AppInjectionSpec: getOldAppInjectionSpec(),
		UseCSIDriver:     address.Of(true),
		Version:          "app-monitoring-version",
	}
}

func getOldActiveGateSpec() ActiveGateSpec {
	return ActiveGateSpec{
		DNSPolicy: corev1.DNSClusterFirstWithHostNet,
		Annotations: map[string]string{
			"activegate-annotation-key": "activegate-annotation-value",
		},
		TlsSecretName:     "activegate-tls-secret-name",
		PriorityClassName: "activegate-priority-class-name",
		Capabilities: []CapabilityDisplayName{
			DynatraceApiCapability.DisplayName,
			KubeMonCapability.DisplayName,
			MetricsIngestCapability.DisplayName,
		},
		CapabilityProperties: CapabilityProperties{
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
			Replicas: address.Of(int32(42)),
			Group:    "activegate-group",
			CustomProperties: &DynaKubeValueSource{
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

func getOldStatus() DynaKubeStatus {
	return DynaKubeStatus{
		OneAgent: OneAgentStatus{
			VersionStatus: status.VersionStatus{
				ImageID:            "oa-image-id",
				Version:            "oa-version",
				Type:               "oa-image-type",
				Source:             status.CustomImageVersionSource,
				LastProbeTimestamp: &testTime,
			},
			Instances: map[string]OneAgentInstance{
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
			ConnectionInfoStatus: OneAgentConnectionInfoStatus{
				ConnectionInfoStatus: ConnectionInfoStatus{
					LastRequest: testTime,
					TenantUUID:  "oa-tenant-uuid",
					Endpoints:   "oa-endpoints",
				},
				CommunicationHosts: []CommunicationHostStatus{
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
		ActiveGate: ActiveGateStatus{
			VersionStatus: status.VersionStatus{
				ImageID:            "ag-image-id",
				Version:            "ag-version",
				Type:               "ag-image-type",
				Source:             status.CustomVersionVersionSource,
				LastProbeTimestamp: &testTime,
			},
		},
		CodeModules: CodeModulesStatus{
			VersionStatus: status.VersionStatus{
				ImageID:            "cm-image-id",
				Version:            "cm-version",
				Type:               "cm-image-type",
				Source:             status.TenantRegistryVersionSource,
				LastProbeTimestamp: &testTime,
			},
		},
		DynatraceApi: DynatraceApiStatus{
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
		KubeSystemUUID:          "kube-system-uuid",
		Phase:                   status.Deploying,
		LastTokenProbeTimestamp: &testTime,
		UpdatedTimestamp:        testTime,
	}
}
