package dynakube

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/metadataenrichment"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/communication"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/value"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/installconfig"
	registryv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func TestConvertFrom(t *testing.T) {
	t.Run("migrate base from latest to v1beta2", func(t *testing.T) {
		from := getNewDynakubeBase()
		to := DynaKube{}

		err := to.ConvertFrom(&from)
		require.NoError(t, err)

		compareBase(t, to, from)
	})

	t.Run("migrate host-monitoring from latest to v1beta2", func(t *testing.T) {
		from := getNewDynakubeBase()
		hostSpec := getNewHostInjectSpec()
		from.Spec.OneAgent.HostMonitoring = &hostSpec
		to := DynaKube{}

		err := to.ConvertFrom(&from)
		require.NoError(t, err)

		assert.Nil(t, to.Spec.OneAgent.ClassicFullStack)
		assert.Nil(t, to.Spec.OneAgent.CloudNativeFullStack)
		assert.Nil(t, to.Spec.OneAgent.ApplicationMonitoring)
		assert.Equal(t, from.Spec.OneAgent.HostGroup, to.Spec.OneAgent.HostGroup)

		compareHostInjectSpec(t, *to.Spec.OneAgent.HostMonitoring, *from.Spec.OneAgent.HostMonitoring)
		compareBase(t, to, from)
	})

	t.Run("migrate classic-fullstack from latest to v1beta2", func(t *testing.T) {
		from := getNewDynakubeBase()
		hostSpec := getNewHostInjectSpec()
		from.Spec.OneAgent.ClassicFullStack = &hostSpec
		to := DynaKube{}

		err := to.ConvertFrom(&from)
		require.NoError(t, err)

		assert.Nil(t, to.Spec.OneAgent.CloudNativeFullStack)
		assert.Nil(t, to.Spec.OneAgent.ApplicationMonitoring)
		assert.Nil(t, to.Spec.OneAgent.HostMonitoring)
		assert.Equal(t, from.Spec.OneAgent.HostGroup, to.Spec.OneAgent.HostGroup)

		compareHostInjectSpec(t, *to.Spec.OneAgent.ClassicFullStack, *from.Spec.OneAgent.ClassicFullStack)
		compareBase(t, to, from)
	})

	t.Run("migrate cloud-native from latest to v1beta2", func(t *testing.T) {
		from := getNewDynakubeBase()
		spec := getNewCloudNativeSpec()
		from.Spec.OneAgent.CloudNativeFullStack = &spec
		to := DynaKube{}

		err := to.ConvertFrom(&from)
		require.NoError(t, err)

		assert.Nil(t, to.Spec.OneAgent.ClassicFullStack)
		assert.Nil(t, to.Spec.OneAgent.ApplicationMonitoring)
		assert.Nil(t, to.Spec.OneAgent.HostMonitoring)
		assert.Equal(t, from.Spec.OneAgent.HostGroup, to.Spec.OneAgent.HostGroup)

		compareCloudNativeSpec(t, *to.Spec.OneAgent.CloudNativeFullStack, *from.Spec.OneAgent.CloudNativeFullStack)
		compareBase(t, to, from)
	})

	t.Run("migrate application-monitoring from latest to v1beta2", func(t *testing.T) {
		from := getNewDynakubeBase()
		appSpec := getNewApplicationMonitoringSpec()
		from.Spec.OneAgent.ApplicationMonitoring = &appSpec
		to := DynaKube{}

		err := to.ConvertFrom(&from)
		require.NoError(t, err)

		assert.Nil(t, to.Spec.OneAgent.ClassicFullStack)
		assert.Nil(t, to.Spec.OneAgent.CloudNativeFullStack)
		assert.Nil(t, to.Spec.OneAgent.HostMonitoring)
		assert.Equal(t, from.Spec.OneAgent.HostGroup, to.Spec.OneAgent.HostGroup)

		compareApplicationMonitoringSpec(t, *to.Spec.OneAgent.ApplicationMonitoring, *from.Spec.OneAgent.ApplicationMonitoring)
	})

	t.Run("migrate activegate from latest to v1beta2", func(t *testing.T) {
		from := getNewDynakubeBase()
		agSpec := getNewActiveGateSpec()
		from.Spec.ActiveGate = agSpec
		to := DynaKube{}

		err := to.ConvertFrom(&from)
		require.NoError(t, err)

		compareActiveGateSpec(t, to.Spec.ActiveGate, from.Spec.ActiveGate)
	})

	t.Run("migrate status from latest to v1beta2", func(t *testing.T) {
		from := getNewDynakubeBase()
		from.Status = getNewStatus()
		to := DynaKube{}

		err := to.ConvertFrom(&from)
		require.NoError(t, err)

		compareStatus(t, to.Status, from.Status)
	})
}

func compareBase(t *testing.T, oldDk DynaKube, newDk dynakube.DynaKube) {
	require.NotEmpty(t, oldDk)
	require.NotEmpty(t, newDk)

	// Some feature-flags are moved, so ObjectMeta will differ in that 1 field
	oldAnnotations := oldDk.Annotations
	newAnnotations := newDk.Annotations
	oldDk.Annotations = nil
	newDk.Annotations = nil

	assert.Equal(t, oldDk.ObjectMeta, newDk.ObjectMeta)

	oldDk.Annotations = oldAnnotations
	newDk.Annotations = newAnnotations

	assert.Equal(t, oldDk.Spec.APIURL, newDk.Spec.APIURL)
	assert.Equal(t, oldDk.Spec.Tokens, newDk.Spec.Tokens)
	assert.Equal(t, oldDk.Spec.CustomPullSecret, newDk.Spec.CustomPullSecret)
	assert.Equal(t, oldDk.Spec.EnableIstio, newDk.Spec.EnableIstio)
	assert.Equal(t, oldDk.Spec.SkipCertCheck, newDk.Spec.SkipCertCheck)
	assert.Equal(t, oldDk.Spec.TrustedCAs, newDk.Spec.TrustedCAs)
	assert.Equal(t, uint16(oldDk.Spec.DynatraceAPIRequestThreshold), *newDk.Spec.DynatraceAPIRequestThreshold) //nolint:gosec
	assert.Equal(t, oldDk.Spec.NetworkZone, newDk.Spec.NetworkZone)

	if newDk.OneAgent().IsAppInjectionNeeded() {
		assert.Equal(t, oldDk.OneAgentNamespaceSelector(), newDk.OneAgent().GetNamespaceSelector())
	}

	assert.Equal(t, oldDk.MetadataEnrichmentEnabled(), newDk.MetadataEnrichment().IsEnabled())
	assert.Equal(t, oldDk.Spec.MetadataEnrichment.NamespaceSelector, newDk.Spec.MetadataEnrichment.NamespaceSelector)

	if oldDk.Spec.Proxy != nil || newDk.Spec.Proxy != nil { // necessary so we don't explode with nil pointer when not set
		require.NotNil(t, oldDk.Spec.Proxy)
		require.NotNil(t, newDk.Spec.Proxy)
		assert.Equal(t, oldDk.Spec.Proxy.Value, newDk.Spec.Proxy.Value)
		assert.Equal(t, oldDk.Spec.Proxy.ValueFrom, newDk.Spec.Proxy.ValueFrom)
	}

	if oldDk.FF().GetCSIMaxFailedMountAttempts() != exp.DefaultCSIMaxFailedMountAttempts {
		assert.Equal(t, exp.MountAttemptsToTimeout(oldDk.FF().GetCSIMaxFailedMountAttempts()), newDk.FF().GetCSIMaxRetryTimeout().String())
	}
}

func compareHostInjectSpec(t *testing.T, oldSpec HostInjectSpec, newSpec oneagent.HostInjectSpec) {
	assert.Equal(t, oldSpec.Annotations, newSpec.Annotations)
	assert.Equal(t, oldSpec.Labels, newSpec.Labels)
	assert.Equal(t, oldSpec.NodeSelector, newSpec.NodeSelector)
	assert.Equal(t, oldSpec.AutoUpdate, *newSpec.AutoUpdate)
	assert.Equal(t, oldSpec.Version, newSpec.Version)
	assert.Equal(t, oldSpec.Image, newSpec.Image)
	assert.Equal(t, oldSpec.DNSPolicy, newSpec.DNSPolicy)
	assert.Equal(t, oldSpec.PriorityClassName, newSpec.PriorityClassName)
	assert.Equal(t, oldSpec.SecCompProfile, newSpec.SecCompProfile)
	assert.Equal(t, oldSpec.OneAgentResources, newSpec.OneAgentResources)
	assert.Equal(t, oldSpec.Tolerations, newSpec.Tolerations)
	assert.Equal(t, oldSpec.Env, newSpec.Env)
	assert.Equal(t, oldSpec.Args, newSpec.Args)
}

func compareAppInjectionSpec(t *testing.T, oldSpec AppInjectionSpec, newSpec oneagent.AppInjectionSpec) {
	assert.Equal(t, oldSpec.CodeModulesImage, newSpec.CodeModulesImage)
	assert.Equal(t, oldSpec.InitResources, newSpec.InitResources)
	assert.Equal(t, oldSpec.NamespaceSelector, newSpec.NamespaceSelector)
}

func compareCloudNativeSpec(t *testing.T, oldSpec CloudNativeFullStackSpec, newSpec oneagent.CloudNativeFullStackSpec) {
	compareAppInjectionSpec(t, oldSpec.AppInjectionSpec, newSpec.AppInjectionSpec)
	compareHostInjectSpec(t, oldSpec.HostInjectSpec, newSpec.HostInjectSpec)
}

func compareApplicationMonitoringSpec(t *testing.T, oldSpec ApplicationMonitoringSpec, newSpec oneagent.ApplicationMonitoringSpec) {
	compareAppInjectionSpec(t, oldSpec.AppInjectionSpec, newSpec.AppInjectionSpec)
	assert.Equal(t, oldSpec.UseCSIDriver, installconfig.GetModules().CSIDriver)
	assert.Equal(t, oldSpec.Version, newSpec.Version)
}

func compareActiveGateSpec(t *testing.T, oldSpec ActiveGateSpec, newSpec activegate.Spec) {
	assert.Equal(t, oldSpec.Annotations, newSpec.Annotations)
	assert.Equal(t, oldSpec.DNSPolicy, newSpec.DNSPolicy)
	assert.Equal(t, oldSpec.Env, newSpec.Env)
	assert.Equal(t, oldSpec.Image, newSpec.Image)
	assert.Equal(t, oldSpec.Labels, newSpec.Labels)
	assert.Equal(t, oldSpec.NodeSelector, newSpec.NodeSelector)
	assert.Equal(t, oldSpec.Resources, newSpec.Resources)
	assert.Equal(t, oldSpec.PriorityClassName, newSpec.PriorityClassName)
	assert.Equal(t, oldSpec.Tolerations, newSpec.Tolerations)
	assert.Equal(t, len(oldSpec.Capabilities), len(newSpec.Capabilities))
	assert.Equal(t, oldSpec.TLSSecretName, newSpec.TLSSecretName)
	assert.Equal(t, oldSpec.TopologySpreadConstraints, newSpec.TopologySpreadConstraints)
	assert.Equal(t, oldSpec.Group, newSpec.Group)
	assert.Equal(t, oldSpec.Replicas, *newSpec.Replicas)

	if oldSpec.CustomProperties != nil || newSpec.CustomProperties != nil { // necessary so we don't explode with nil pointer when not set
		require.NotNil(t, oldSpec.CustomProperties)
		require.NotNil(t, newSpec.CustomProperties)
		assert.Equal(t, oldSpec.CustomProperties.Value, newSpec.CustomProperties.Value)
		assert.Equal(t, oldSpec.CustomProperties.ValueFrom, newSpec.CustomProperties.ValueFrom)
	}
}

func compareStatus(t *testing.T, oldStatus DynaKubeStatus, newStatus dynakube.DynaKubeStatus) {
	// Base
	assert.Equal(t, oldStatus.Conditions, newStatus.Conditions)
	assert.Equal(t, oldStatus.DynatraceAPI.LastTokenScopeRequest, newStatus.DynatraceAPI.LastTokenScopeRequest)
	assert.Equal(t, oldStatus.KubeSystemUUID, newStatus.KubeSystemUUID)
	assert.Equal(t, oldStatus.Phase, newStatus.Phase)
	assert.Equal(t, oldStatus.UpdatedTimestamp, newStatus.UpdatedTimestamp)

	// CodeModule
	assert.Equal(t, oldStatus.CodeModules.VersionStatus, newStatus.CodeModules.VersionStatus)

	// ActiveGate
	assert.Equal(t, oldStatus.ActiveGate.VersionStatus, newStatus.ActiveGate.VersionStatus)
	assert.Equal(t, oldStatus.ActiveGate.ConnectionInfoStatus.Endpoints, newStatus.ActiveGate.ConnectionInfo.Endpoints)
	assert.Equal(t, oldStatus.ActiveGate.ConnectionInfoStatus.LastRequest, newStatus.ActiveGate.ConnectionInfo.LastRequest)
	assert.Equal(t, oldStatus.ActiveGate.ConnectionInfoStatus.TenantUUID, newStatus.ActiveGate.ConnectionInfo.TenantUUID)

	// OneAgent
	assert.Equal(t, oldStatus.OneAgent.VersionStatus, newStatus.OneAgent.VersionStatus)
	assert.Equal(t, oldStatus.OneAgent.ConnectionInfoStatus.Endpoints, newStatus.OneAgent.ConnectionInfoStatus.Endpoints)
	assert.Equal(t, oldStatus.OneAgent.ConnectionInfoStatus.LastRequest, newStatus.OneAgent.ConnectionInfoStatus.LastRequest)
	assert.Equal(t, oldStatus.OneAgent.ConnectionInfoStatus.TenantUUID, newStatus.OneAgent.ConnectionInfoStatus.TenantUUID)
	assert.Equal(t, oldStatus.OneAgent.Healthcheck, newStatus.OneAgent.Healthcheck)
	assert.Equal(t, oldStatus.OneAgent.LastInstanceStatusUpdate, newStatus.OneAgent.LastInstanceStatusUpdate)

	require.Equal(t, len(oldStatus.OneAgent.Instances), len(newStatus.OneAgent.Instances))

	for key, value := range oldStatus.OneAgent.Instances {
		assert.Equal(t, value.IPAddress, newStatus.OneAgent.Instances[key].IPAddress)
		assert.Equal(t, value.PodName, newStatus.OneAgent.Instances[key].PodName)
	}

	require.Equal(t, len(oldStatus.OneAgent.ConnectionInfoStatus.CommunicationHosts), len(newStatus.OneAgent.ConnectionInfoStatus.CommunicationHosts))

	for i, oldHost := range oldStatus.OneAgent.ConnectionInfoStatus.CommunicationHosts {
		newHost := newStatus.OneAgent.ConnectionInfoStatus.CommunicationHosts[i]
		assert.Equal(t, oldHost.Host, newHost.Host)
		assert.Equal(t, oldHost.Port, newHost.Port)
		assert.Equal(t, oldHost.Protocol, newHost.Protocol)
	}
}

func getNewDynakubeBase() dynakube.DynaKube {
	return dynakube.DynaKube{
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
		Spec: dynakube.DynaKubeSpec{
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
			MetadataEnrichment: metadataenrichment.Spec{
				Enabled:           ptr.To(true),
				NamespaceSelector: getTestNamespaceSelector(),
			},
			OneAgent: oneagent.Spec{
				HostGroup: "host-group",
			},
		},
	}
}

func getNewHostInjectSpec() oneagent.HostInjectSpec {
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

func getNewAppInjectionSpec() oneagent.AppInjectionSpec {
	return oneagent.AppInjectionSpec{
		InitResources: &corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceCPU: *resource.NewScaledQuantity(2, 1)},
		},
		CodeModulesImage:  "app-injection-image",
		NamespaceSelector: getTestNamespaceSelector(),
	}
}

func getNewCloudNativeSpec() oneagent.CloudNativeFullStackSpec {
	return oneagent.CloudNativeFullStackSpec{
		AppInjectionSpec: getNewAppInjectionSpec(),
		HostInjectSpec:   getNewHostInjectSpec(),
	}
}

func getNewApplicationMonitoringSpec() oneagent.ApplicationMonitoringSpec {
	return oneagent.ApplicationMonitoringSpec{
		AppInjectionSpec: getNewAppInjectionSpec(),
		Version:          "app-monitoring-version",
	}
}

func getNewActiveGateSpec() activegate.Spec {
	return activegate.Spec{
		DNSPolicy: corev1.DNSClusterFirstWithHostNet,
		Annotations: map[string]string{
			"activegate-annotation-key": "activegate-annotation-value",
		},
		TLSSecretName:     "activegate-tls-secret-name",
		PriorityClassName: "activegate-priority-class-name",
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

func getNewStatus() dynakube.DynaKubeStatus {
	return dynakube.DynaKubeStatus{
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
		DynatraceAPI: dynakube.DynatraceAPIStatus{
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
