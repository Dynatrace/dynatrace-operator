package dynakube

import (
	"slices"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/communication"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/image"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/value"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube/kspm"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube/logmonitoring"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube/oneagent"
	dynakubev1beta5 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta5/dynakube"
	activegatev1beta5 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta5/dynakube/activegate"
	kspmv1beta5 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta5/dynakube/kspm"
	logmonitoringv1beta6 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta5/dynakube/logmonitoring"
	oneagentv1beta5 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta5/dynakube/oneagent"
	registryv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
)

func TestConvertFrom(t *testing.T) {
	t.Run("migrate base from v1beta5 to v1beta4", func(t *testing.T) {
		from := getNewDynakubeBase()
		to := DynaKube{}

		err := to.ConvertFrom(&from)
		require.NoError(t, err)

		compareBase(t, to, from)
	})

	t.Run("migrate metadata-enrichment from v1beta5 to v1beta4", func(t *testing.T) {
		from := getNewDynakubeBase()
		to := DynaKube{}

		err := to.ConvertFrom(&from)
		require.NoError(t, err)

		assert.True(t, to.MetadataEnrichmentEnabled())
		compareBase(t, to, from)
	})

	t.Run("migrate host-monitoring from v1beta5 to v1beta4", func(t *testing.T) {
		from := getNewDynakubeBase()
		hostSpec := getNewHostInjectSpec()
		from.Spec.OneAgent.HostMonitoring = &hostSpec
		to := DynaKube{}

		err := to.ConvertFrom(&from)
		require.NoError(t, err)

		compareHostInjectSpec(t, *to.Spec.OneAgent.HostMonitoring, *from.Spec.OneAgent.HostMonitoring)
		compareBase(t, to, from)
	})

	t.Run("migrate classic-fullstack from v1beta5 to v1beta4", func(t *testing.T) {
		from := getNewDynakubeBase()
		hostSpec := getNewHostInjectSpec()
		from.Spec.OneAgent.ClassicFullStack = &hostSpec
		to := DynaKube{}

		err := to.ConvertFrom(&from)
		require.NoError(t, err)

		assert.Nil(t, to.Spec.OneAgent.CloudNativeFullStack)
		assert.Nil(t, to.Spec.OneAgent.ApplicationMonitoring)
		assert.Nil(t, to.Spec.OneAgent.HostMonitoring)

		compareHostInjectSpec(t, *to.Spec.OneAgent.ClassicFullStack, *from.Spec.OneAgent.ClassicFullStack)
		compareBase(t, to, from)
	})

	t.Run("migrate cloud-native from v1beta5 to v1beta4", func(t *testing.T) {
		from := getNewDynakubeBase()
		spec := getNewCloudNativeSpec()
		from.Spec.OneAgent.CloudNativeFullStack = &spec
		to := DynaKube{}

		err := to.ConvertFrom(&from)
		require.NoError(t, err)

		assert.Nil(t, to.Spec.OneAgent.ClassicFullStack)
		assert.Nil(t, to.Spec.OneAgent.ApplicationMonitoring)
		assert.Nil(t, to.Spec.OneAgent.HostMonitoring)

		compareCloudNativeSpec(t, *to.Spec.OneAgent.CloudNativeFullStack, *from.Spec.OneAgent.CloudNativeFullStack)
		compareBase(t, to, from)
	})

	t.Run("migrate application-monitoring from v1beta5 to v1beta4", func(t *testing.T) {
		from := getNewDynakubeBase()
		appSpec := getNewApplicationMonitoringSpec()
		from.Spec.OneAgent.ApplicationMonitoring = &appSpec
		to := DynaKube{}

		err := to.ConvertFrom(&from)
		require.NoError(t, err)

		assert.Nil(t, to.Spec.OneAgent.ClassicFullStack)
		assert.Nil(t, to.Spec.OneAgent.CloudNativeFullStack)
		assert.Nil(t, to.Spec.OneAgent.HostMonitoring)

		compareApplicationMonitoringSpec(t, *to.Spec.OneAgent.ApplicationMonitoring, *from.Spec.OneAgent.ApplicationMonitoring)
		compareBase(t, to, from)
	})

	t.Run("migrate activegate from v1beta5 to v1beta4", func(t *testing.T) {
		from := getNewDynakubeBase()
		agSpec := getNewActiveGateSpec()
		from.Spec.ActiveGate = agSpec
		to := DynaKube{}

		err := to.ConvertFrom(&from)
		require.NoError(t, err)

		compareActiveGateSpec(t, to.Spec.ActiveGate, from.Spec.ActiveGate)
		compareBase(t, to, from)
	})

	t.Run("migrate extensions from v1beta5 to v1beta4", func(t *testing.T) {
		from := getNewDynakubeBase()
		from.Spec.Extensions = &dynakubev1beta5.ExtensionsSpec{}
		to := DynaKube{}

		err := to.ConvertFrom(&from)
		require.NoError(t, err)

		assert.NotNil(t, to.Spec.Extensions)
		compareBase(t, to, from)
	})

	t.Run("migrate log-monitoring from v1beta5 to v1beta4", func(t *testing.T) {
		from := getNewDynakubeBase()
		from.Spec.LogMonitoring = getNewLogMonitoringSpec()
		to := DynaKube{}

		err := to.ConvertFrom(&from)
		require.NoError(t, err)

		compareLogMonitoringSpec(t, to.Spec.LogMonitoring, from.Spec.LogMonitoring)
		compareBase(t, to, from)
	})

	t.Run("migrate kspm from v1beta5 to v1beta3", func(t *testing.T) {
		from := getNewDynakubeBase()
		from.Spec.Kspm = &kspmv1beta5.Spec{}
		to := DynaKube{}

		err := to.ConvertFrom(&from)
		require.NoError(t, err)

		assert.NotNil(t, to.Spec.Kspm)
		compareBase(t, to, from)
	})

	t.Run("migrate extensions templates from v1beta5 to v1beta4", func(t *testing.T) {
		from := getNewDynakubeBase()
		from.Spec.Templates.OpenTelemetryCollector = getNewOpenTelemetryTemplateSpec()
		from.Spec.Templates.ExtensionExecutionController = getNewExtensionExecutionControllerSpec()

		to := DynaKube{}

		err := to.ConvertFrom(&from)
		require.NoError(t, err)

		compareOpenTelemetryTemplateSpec(t, to.Spec.Templates.OpenTelemetryCollector, from.Spec.Templates.OpenTelemetryCollector)
		compareExtensionsExecutionControllerTemplateSpec(t, to.Spec.Templates.ExtensionExecutionController, from.Spec.Templates.ExtensionExecutionController)

		compareBase(t, to, from)
	})

	t.Run("migrate log-monitoring templates from v1beta5 to v1beta4", func(t *testing.T) {
		from := getNewDynakubeBase()
		from.Spec.Templates.LogMonitoring = getNewLogMonitoringTemplateSpec()

		to := DynaKube{}

		err := to.ConvertFrom(&from)
		require.NoError(t, err)

		compareLogMonitoringTemplateSpec(t, to.Spec.Templates.LogMonitoring, from.Spec.Templates.LogMonitoring)
		compareBase(t, to, from)
	})

	t.Run("migrate kspm templates from v1beta5 to v1beta4", func(t *testing.T) {
		from := getNewDynakubeBase()
		from.Spec.Templates.KspmNodeConfigurationCollector = getNewNodeConfigurationCollectorTemplateSpec()

		to := DynaKube{}

		err := to.ConvertFrom(&from)
		require.NoError(t, err)

		compareNodeConfigurationCollectorTemplateSpec(t, to.Spec.Templates.KspmNodeConfigurationCollector, from.Spec.Templates.KspmNodeConfigurationCollector)
		compareBase(t, to, from)
	})

	t.Run("migrate status from v1beta5 to v1beta4", func(t *testing.T) {
		from := getNewDynakubeBase()
		from.Status = getNewStatus()
		to := DynaKube{}

		err := to.ConvertFrom(&from)
		require.NoError(t, err)

		compareStatus(t, to.Status, from.Status)
	})
	t.Run("migrate hostGroup", func(t *testing.T) {
		from := getNewDynakubeBase()
		from.Status = getNewStatus()
		to := DynaKube{}

		err := to.ConvertFrom(&from)
		require.NoError(t, err)

		assert.Equal(t, to.Spec.OneAgent.HostGroup, from.Spec.OneAgent.HostGroup)
	})
}

func compareBase(t *testing.T, oldDk DynaKube, newDk dynakubev1beta5.DynaKube) {
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

	if oldDk.Spec.Proxy != nil || newDk.Spec.Proxy != nil { // necessary so we don't explode with nil pointer when not set
		require.NotNil(t, oldDk.Spec.Proxy)
		require.NotNil(t, newDk.Spec.Proxy)
		assert.Equal(t, oldDk.Spec.Proxy.Value, newDk.Spec.Proxy.Value)
		assert.Equal(t, oldDk.Spec.Proxy.ValueFrom, newDk.Spec.Proxy.ValueFrom)
	}

	assert.Equal(t, oldDk.Spec.DynatraceApiRequestThreshold, newDk.Spec.DynatraceApiRequestThreshold)
	assert.Equal(t, oldDk.Spec.APIURL, newDk.Spec.APIURL)
	assert.Equal(t, oldDk.Spec.Tokens, newDk.Spec.Tokens)
	assert.Equal(t, oldDk.Spec.TrustedCAs, newDk.Spec.TrustedCAs)
	assert.Equal(t, oldDk.Spec.NetworkZone, newDk.Spec.NetworkZone)
	assert.Equal(t, oldDk.Spec.CustomPullSecret, newDk.Spec.CustomPullSecret)
	assert.Equal(t, oldDk.Spec.SkipCertCheck, newDk.Spec.SkipCertCheck)
	assert.Equal(t, oldDk.Spec.EnableIstio, newDk.Spec.EnableIstio)

	if newDk.OneAgent().IsAppInjectionNeeded() {
		assert.Equal(t, oldDk.OneAgent().GetNamespaceSelector(), newDk.OneAgent().GetNamespaceSelector())
	}

	assert.Equal(t, oldDk.MetadataEnrichmentEnabled(), newDk.MetadataEnrichmentEnabled())
	assert.Equal(t, oldDk.Spec.MetadataEnrichment.NamespaceSelector, newDk.Spec.MetadataEnrichment.NamespaceSelector)

	if oldDk.FF().GetCSIMaxFailedMountAttempts() != exp.DefaultCSIMaxFailedMountAttempts {
		assert.Equal(t, exp.MountAttemptsToTimeout(oldDk.FF().GetCSIMaxFailedMountAttempts()), newDk.FF().GetCSIMaxRetryTimeout().String())
	}
}

func compareHostInjectSpec(t *testing.T, oldSpec oneagent.HostInjectSpec, newSpec oneagentv1beta5.HostInjectSpec) {
	assert.Equal(t, oldSpec.Annotations, newSpec.Annotations)
	assert.Equal(t, oldSpec.Args, newSpec.Args)
	assert.Equal(t, *oldSpec.AutoUpdate, *newSpec.AutoUpdate)
	assert.Equal(t, oldSpec.DNSPolicy, newSpec.DNSPolicy)
	assert.Equal(t, oldSpec.Env, newSpec.Env)
	assert.Equal(t, oldSpec.Image, newSpec.Image)
	assert.Equal(t, oldSpec.Labels, newSpec.Labels)
	assert.Equal(t, oldSpec.NodeSelector, newSpec.NodeSelector)
	assert.Equal(t, oldSpec.OneAgentResources, newSpec.OneAgentResources)
	assert.Equal(t, oldSpec.PriorityClassName, newSpec.PriorityClassName)
	assert.Equal(t, oldSpec.Tolerations, newSpec.Tolerations)
	assert.Equal(t, oldSpec.Version, newSpec.Version)
	assert.Equal(t, oldSpec.SecCompProfile, newSpec.SecCompProfile)
}

func compareAppInjectionSpec(t *testing.T, oldSpec oneagent.AppInjectionSpec, newSpec oneagentv1beta5.AppInjectionSpec) {
	assert.Equal(t, oldSpec.InitResources, newSpec.InitResources)
	assert.Equal(t, oldSpec.CodeModulesImage, newSpec.CodeModulesImage)
	assert.Equal(t, oldSpec.NamespaceSelector, newSpec.NamespaceSelector)
}

func compareCloudNativeSpec(t *testing.T, oldSpec oneagent.CloudNativeFullStackSpec, newSpec oneagentv1beta5.CloudNativeFullStackSpec) {
	compareHostInjectSpec(t, oldSpec.HostInjectSpec, newSpec.HostInjectSpec)
	compareAppInjectionSpec(t, oldSpec.AppInjectionSpec, newSpec.AppInjectionSpec)
}

func compareApplicationMonitoringSpec(t *testing.T, oldSpec oneagent.ApplicationMonitoringSpec, newSpec oneagentv1beta5.ApplicationMonitoringSpec) {
	assert.Equal(t, oldSpec.Version, newSpec.Version)
	compareAppInjectionSpec(t, oldSpec.AppInjectionSpec, newSpec.AppInjectionSpec)
}

func compareActiveGateSpec(t *testing.T, oldSpec activegate.Spec, newSpec activegatev1beta5.Spec) {
	assert.Equal(t, oldSpec.Annotations, newSpec.Annotations)
	assert.Equal(t, oldSpec.TlsSecretName, newSpec.TlsSecretName)
	assert.Equal(t, oldSpec.DNSPolicy, newSpec.DNSPolicy)
	assert.Equal(t, oldSpec.PriorityClassName, newSpec.PriorityClassName)

	if oldSpec.CapabilityProperties.CustomProperties != nil || newSpec.CapabilityProperties.CustomProperties != nil { // necessary so we don't explode with nil pointer when not set
		require.NotNil(t, oldSpec.CapabilityProperties.CustomProperties)
		require.NotNil(t, newSpec.CapabilityProperties.CustomProperties)
		assert.Equal(t, oldSpec.CapabilityProperties.CustomProperties.Value, newSpec.CapabilityProperties.CustomProperties.Value)
		assert.Equal(t, oldSpec.CapabilityProperties.CustomProperties.ValueFrom, newSpec.CapabilityProperties.CustomProperties.ValueFrom)
	}

	assert.Equal(t, oldSpec.CapabilityProperties.NodeSelector, newSpec.CapabilityProperties.NodeSelector)
	assert.Equal(t, oldSpec.CapabilityProperties.Labels, newSpec.CapabilityProperties.Labels)
	assert.Equal(t, *oldSpec.CapabilityProperties.Replicas, *newSpec.CapabilityProperties.Replicas)
	assert.Equal(t, oldSpec.CapabilityProperties.Image, newSpec.CapabilityProperties.Image)
	assert.Equal(t, oldSpec.CapabilityProperties.Group, newSpec.CapabilityProperties.Group)
	assert.Equal(t, oldSpec.CapabilityProperties.Resources, newSpec.CapabilityProperties.Resources)
	assert.Equal(t, oldSpec.CapabilityProperties.Tolerations, newSpec.CapabilityProperties.Tolerations)
	assert.Equal(t, oldSpec.CapabilityProperties.Env, newSpec.CapabilityProperties.Env)
	assert.Equal(t, oldSpec.CapabilityProperties.TopologySpreadConstraints, newSpec.CapabilityProperties.TopologySpreadConstraints)

	assert.Len(t, newSpec.Capabilities, len(oldSpec.Capabilities))

	for _, oldCapability := range oldSpec.Capabilities {
		assert.Contains(t, newSpec.Capabilities, activegatev1beta5.CapabilityDisplayName(oldCapability))
	}
}

func compareStatus(t *testing.T, oldStatus DynaKubeStatus, newStatus dynakubev1beta5.DynaKubeStatus) {
	// Base
	assert.Equal(t, oldStatus.Conditions, newStatus.Conditions)
	assert.Equal(t, oldStatus.DynatraceApi.LastTokenScopeRequest, newStatus.DynatraceApi.LastTokenScopeRequest)
	assert.Equal(t, oldStatus.KubeSystemUUID, newStatus.KubeSystemUUID)
	assert.Equal(t, oldStatus.Phase, newStatus.Phase)
	assert.Equal(t, oldStatus.UpdatedTimestamp, newStatus.UpdatedTimestamp)

	// CodeModule
	assert.Equal(t, oldStatus.CodeModules.VersionStatus, newStatus.CodeModules.VersionStatus)

	// ActiveGate
	assert.Equal(t, oldStatus.ActiveGate.VersionStatus, newStatus.ActiveGate.VersionStatus)
	assert.Equal(t, oldStatus.ActiveGate.ConnectionInfo.Endpoints, newStatus.ActiveGate.ConnectionInfo.Endpoints)
	assert.Equal(t, oldStatus.ActiveGate.ConnectionInfo.LastRequest, newStatus.ActiveGate.ConnectionInfo.LastRequest)
	assert.Equal(t, oldStatus.ActiveGate.ConnectionInfo.TenantUUID, newStatus.ActiveGate.ConnectionInfo.TenantUUID)

	// OneAgent
	assert.Equal(t, oldStatus.OneAgent.VersionStatus, newStatus.OneAgent.VersionStatus)
	assert.Equal(t, oldStatus.OneAgent.ConnectionInfoStatus.Endpoints, newStatus.OneAgent.ConnectionInfoStatus.Endpoints)
	assert.Equal(t, oldStatus.OneAgent.ConnectionInfoStatus.LastRequest, newStatus.OneAgent.ConnectionInfoStatus.LastRequest)
	assert.Equal(t, oldStatus.OneAgent.ConnectionInfoStatus.TenantUUID, newStatus.OneAgent.ConnectionInfoStatus.TenantUUID)
	assert.Equal(t, oldStatus.OneAgent.Healthcheck, newStatus.OneAgent.Healthcheck)
	assert.Equal(t, oldStatus.OneAgent.LastInstanceStatusUpdate, newStatus.OneAgent.LastInstanceStatusUpdate)

	require.Equal(t, len(oldStatus.OneAgent.Instances), len(newStatus.OneAgent.Instances)) //nolint:testifylint

	for key, value := range oldStatus.OneAgent.Instances {
		assert.Equal(t, value.IPAddress, newStatus.OneAgent.Instances[key].IPAddress)
		assert.Equal(t, value.PodName, newStatus.OneAgent.Instances[key].PodName)
	}

	require.Equal(t, len(oldStatus.OneAgent.ConnectionInfoStatus.CommunicationHosts), len(newStatus.OneAgent.ConnectionInfoStatus.CommunicationHosts)) //nolint:testifylint

	for i, oldHost := range oldStatus.OneAgent.ConnectionInfoStatus.CommunicationHosts {
		newHost := newStatus.OneAgent.ConnectionInfoStatus.CommunicationHosts[i]
		assert.Equal(t, oldHost.Host, newHost.Host)
		assert.Equal(t, oldHost.Port, newHost.Port)
		assert.Equal(t, oldHost.Protocol, newHost.Protocol)
	}
}

func compareLogMonitoringSpec(t *testing.T, oldSpec *logmonitoring.Spec, newSpec *logmonitoringv1beta6.Spec) {
	if oldSpec == nil {
		assert.Nil(t, newSpec)

		return
	} else {
		require.NotNil(t, newSpec)
	}

	assert.Len(t, newSpec.IngestRuleMatchers, len(oldSpec.IngestRuleMatchers))

	for _, oldMatchers := range oldSpec.IngestRuleMatchers {
		assert.True(t, slices.ContainsFunc(newSpec.IngestRuleMatchers, func(newMatchers logmonitoringv1beta6.IngestRuleMatchers) bool {
			return slices.Equal(newMatchers.Values, oldMatchers.Values) && newMatchers.Attribute == oldMatchers.Attribute
		}))
	}
}

func compareOpenTelemetryTemplateSpec(t *testing.T, oldSpec OpenTelemetryCollectorSpec, newSpec dynakubev1beta5.OpenTelemetryCollectorSpec) {
	assert.Equal(t, oldSpec.Labels, newSpec.Labels)
	assert.Equal(t, oldSpec.Annotations, newSpec.Annotations)
	assert.Equal(t, *oldSpec.Replicas, *newSpec.Replicas)
	assert.Equal(t, oldSpec.ImageRef, newSpec.ImageRef)
	assert.Equal(t, oldSpec.TlsRefName, newSpec.TlsRefName)
	assert.Equal(t, oldSpec.Resources, newSpec.Resources)
	assert.Equal(t, oldSpec.Tolerations, newSpec.Tolerations)
	assert.Equal(t, oldSpec.TopologySpreadConstraints, newSpec.TopologySpreadConstraints)
}

func compareExtensionsExecutionControllerTemplateSpec(t *testing.T, oldSpec ExtensionExecutionControllerSpec, newSpec dynakubev1beta5.ExtensionExecutionControllerSpec) {
	assert.Equal(t, *oldSpec.PersistentVolumeClaim, *newSpec.PersistentVolumeClaim)
	assert.Equal(t, oldSpec.Labels, newSpec.Labels)
	assert.Equal(t, oldSpec.Annotations, newSpec.Annotations)
	assert.Equal(t, oldSpec.ImageRef, newSpec.ImageRef)
	assert.Equal(t, oldSpec.TlsRefName, newSpec.TlsRefName)
	assert.Equal(t, oldSpec.CustomConfig, newSpec.CustomConfig)
	assert.Equal(t, oldSpec.CustomExtensionCertificates, newSpec.CustomExtensionCertificates)
	assert.Equal(t, oldSpec.Resources, newSpec.Resources)
	assert.Equal(t, oldSpec.Tolerations, newSpec.Tolerations)
	assert.Equal(t, oldSpec.TopologySpreadConstraints, newSpec.TopologySpreadConstraints)
	assert.Equal(t, oldSpec.UseEphemeralVolume, newSpec.UseEphemeralVolume)
}

func compareLogMonitoringTemplateSpec(t *testing.T, oldSpec *logmonitoring.TemplateSpec, newSpec *logmonitoringv1beta6.TemplateSpec) {
	if oldSpec == nil {
		assert.Nil(t, newSpec)

		return
	} else {
		require.NotNil(t, newSpec)
	}

	assert.Equal(t, oldSpec.Annotations, newSpec.Annotations)
	assert.Equal(t, oldSpec.Labels, newSpec.Labels)
	assert.Equal(t, oldSpec.NodeSelector, newSpec.NodeSelector)
	assert.Equal(t, oldSpec.ImageRef, newSpec.ImageRef)
	assert.Equal(t, oldSpec.DNSPolicy, newSpec.DNSPolicy)
	assert.Equal(t, oldSpec.PriorityClassName, newSpec.PriorityClassName)
	assert.Equal(t, oldSpec.SecCompProfile, newSpec.SecCompProfile)
	assert.Equal(t, oldSpec.Resources, newSpec.Resources)
	assert.Equal(t, oldSpec.Tolerations, newSpec.Tolerations)
	assert.Equal(t, oldSpec.Args, newSpec.Args)
}

func compareNodeConfigurationCollectorTemplateSpec(t *testing.T, oldSpec kspm.NodeConfigurationCollectorSpec, newSpec kspmv1beta5.NodeConfigurationCollectorSpec) {
	assert.Equal(t, oldSpec.UpdateStrategy, newSpec.UpdateStrategy)
	assert.Equal(t, oldSpec.Labels, newSpec.Labels)
	assert.Equal(t, oldSpec.Annotations, newSpec.Annotations)
	assert.Equal(t, oldSpec.NodeSelector, newSpec.NodeSelector)
	assert.Equal(t, oldSpec.ImageRef, newSpec.ImageRef)
	assert.Equal(t, oldSpec.PriorityClassName, newSpec.PriorityClassName)
	assert.Equal(t, oldSpec.Resources, newSpec.Resources)
	assert.Equal(t, oldSpec.NodeAffinity, newSpec.NodeAffinity)
	assert.Equal(t, oldSpec.Tolerations, newSpec.Tolerations)
	assert.Equal(t, oldSpec.Args, newSpec.Args)
	assert.Equal(t, oldSpec.Env, newSpec.Env)
}

func getNewDynakubeBase() dynakubev1beta5.DynaKube {
	return dynakubev1beta5.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:         "name",
			GenerateName: "generateName",
			Namespace:    "namespace",
			Generation:   0xDEADBEEF,
			Annotations: map[string]string{
				exp.AGIgnoreProxyKey:               "true", //nolint:staticcheck
				exp.AGAutomaticK8sApiMonitoringKey: "true",
			},
			Labels: map[string]string{
				"label": "label-value",
			},
			Finalizers: []string{"finalizer1", "finalizer2"},
		},
		Spec: dynakubev1beta5.DynaKubeSpec{
			Proxy: &value.Source{
				Value:     "proxy-value",
				ValueFrom: "proxy-from",
			},
			DynatraceApiRequestThreshold: ptr.To(uint16(42)),
			APIURL:                       "api-url",
			Tokens:                       "token",
			TrustedCAs:                   "trusted-ca",
			NetworkZone:                  "network-zone",
			CustomPullSecret:             "pull-secret",
			SkipCertCheck:                true,
			EnableIstio:                  true,
			MetadataEnrichment: dynakubev1beta5.MetadataEnrichment{
				Enabled:           ptr.To(true),
				NamespaceSelector: getTestNamespaceSelector(),
			},
		},
	}
}

func getNewHostInjectSpec() oneagentv1beta5.HostInjectSpec {
	return oneagentv1beta5.HostInjectSpec{
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

func getNewAppInjectionSpec() oneagentv1beta5.AppInjectionSpec {
	return oneagentv1beta5.AppInjectionSpec{
		InitResources: &corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceCPU: *resource.NewScaledQuantity(2, 1)},
		},
		CodeModulesImage:  "app-injection-image",
		NamespaceSelector: getTestNamespaceSelector(),
	}
}

func getNewCloudNativeSpec() oneagentv1beta5.CloudNativeFullStackSpec {
	return oneagentv1beta5.CloudNativeFullStackSpec{
		AppInjectionSpec: getNewAppInjectionSpec(),
		HostInjectSpec:   getNewHostInjectSpec(),
	}
}

func getNewApplicationMonitoringSpec() oneagentv1beta5.ApplicationMonitoringSpec {
	return oneagentv1beta5.ApplicationMonitoringSpec{
		AppInjectionSpec: getNewAppInjectionSpec(),
		Version:          "app-monitoring-version",
	}
}

func getNewActiveGateSpec() activegatev1beta5.Spec {
	return activegatev1beta5.Spec{
		DNSPolicy: corev1.DNSClusterFirstWithHostNet,
		Annotations: map[string]string{
			"activegate-annotation-key": "activegate-annotation-value",
		},
		TlsSecretName:     "activegate-tls-secret-name",
		PriorityClassName: "activegate-priority-class-name",
		Capabilities: []activegatev1beta5.CapabilityDisplayName{
			activegatev1beta5.DynatraceApiCapability.DisplayName,
			activegatev1beta5.KubeMonCapability.DisplayName,
			activegatev1beta5.MetricsIngestCapability.DisplayName,
		},
		CapabilityProperties: activegatev1beta5.CapabilityProperties{
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
				Requests: corev1.ResourceList{
					corev1.ResourceCPU: *resource.NewScaledQuantity(3, 1),
				},
				Claims: []corev1.ResourceClaim{{
					Name:    "claim-name",
					Request: "claim-request",
				}},
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

func getNewLogMonitoringSpec() *logmonitoringv1beta6.Spec {
	newSpec := logmonitoringv1beta6.Spec{
		IngestRuleMatchers: make([]logmonitoringv1beta6.IngestRuleMatchers, 0),
	}

	newSpec.IngestRuleMatchers = append(newSpec.IngestRuleMatchers, logmonitoringv1beta6.IngestRuleMatchers{
		Attribute: "attribute1",
		Values:    []string{"matcher1", "matcher2", "matcher3"},
	})

	newSpec.IngestRuleMatchers = append(newSpec.IngestRuleMatchers, logmonitoringv1beta6.IngestRuleMatchers{
		Attribute: "attribute2",
		Values:    []string{"matcher1", "matcher2", "matcher3"},
	})

	newSpec.IngestRuleMatchers = append(newSpec.IngestRuleMatchers, logmonitoringv1beta6.IngestRuleMatchers{
		Attribute: "attribute3",
		Values:    []string{"matcher1", "matcher2", "matcher3"},
	})

	return &newSpec
}

func getNewOpenTelemetryTemplateSpec() dynakubev1beta5.OpenTelemetryCollectorSpec {
	return dynakubev1beta5.OpenTelemetryCollectorSpec{
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
		TlsRefName: "tls-ref-name",
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

func getNewExtensionExecutionControllerSpec() dynakubev1beta5.ExtensionExecutionControllerSpec {
	return dynakubev1beta5.ExtensionExecutionControllerSpec{
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
		TlsRefName: "tls-ref-name",
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

func getPersistentVolumeClaimSpec() *corev1.PersistentVolumeClaimSpec {
	return &corev1.PersistentVolumeClaimSpec{
		AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
		Selector: &metav1.LabelSelector{
			MatchLabels: map[string]string{
				"pvc-label-key1": "pvc-label-value1",
				"pvc-label-key2": "pvc-label-value2",
			},
			MatchExpressions: []metav1.LabelSelectorRequirement{
				{
					Key:      "label-selector-key",
					Operator: "label-selector-value",
					Values:   []string{"pvc-value-1", "pvc-value-key2"},
				},
			},
		},
		Resources: corev1.VolumeResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceStorage: *resource.NewScaledQuantity(3, 1),
			},
			Requests: corev1.ResourceList{
				corev1.ResourceStorage: *resource.NewScaledQuantity(3, 1),
			},
		},
		VolumeName:                "volume-name",
		StorageClassName:          ptr.To("localstorage"),
		VolumeMode:                ptr.To(corev1.PersistentVolumeFilesystem),
		VolumeAttributesClassName: ptr.To("volume-attributes-class-name"),
	}
}

func getNewLogMonitoringTemplateSpec() *logmonitoringv1beta6.TemplateSpec {
	return &logmonitoringv1beta6.TemplateSpec{
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

func getNewNodeConfigurationCollectorTemplateSpec() kspmv1beta5.NodeConfigurationCollectorSpec {
	return kspmv1beta5.NodeConfigurationCollectorSpec{
		UpdateStrategy: &v1.DaemonSetUpdateStrategy{
			Type: "daemonset-update-strategy-type",
			RollingUpdate: &v1.RollingUpdateDaemonSet{
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

func getNewStatus() dynakubev1beta5.DynaKubeStatus {
	return dynakubev1beta5.DynaKubeStatus{
		OneAgent: oneagentv1beta5.Status{
			VersionStatus: status.VersionStatus{
				ImageID:            "oa-image-id",
				Version:            "oa-version",
				Type:               "oa-image-type",
				Source:             status.CustomImageVersionSource,
				LastProbeTimestamp: &testTime,
			},
			Instances: map[string]oneagentv1beta5.Instance{
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
			ConnectionInfoStatus: oneagentv1beta5.ConnectionInfoStatus{
				ConnectionInfo: communication.ConnectionInfo{
					LastRequest: testTime,
					TenantUUID:  "oa-tenant-uuid",
					Endpoints:   "oa-endpoints",
				},
				CommunicationHosts: []oneagentv1beta5.CommunicationHostStatus{
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
		ActiveGate: activegatev1beta5.Status{
			VersionStatus: status.VersionStatus{
				ImageID:            "ag-image-id",
				Version:            "ag-version",
				Type:               "ag-image-type",
				Source:             status.CustomVersionVersionSource,
				LastProbeTimestamp: &testTime,
			},
		},
		CodeModules: oneagentv1beta5.CodeModulesStatus{
			VersionStatus: status.VersionStatus{
				ImageID:            "cm-image-id",
				Version:            "cm-version",
				Type:               "cm-image-type",
				Source:             status.TenantRegistryVersionSource,
				LastProbeTimestamp: &testTime,
			},
		},
		DynatraceApi: dynakubev1beta5.DynatraceApiStatus{
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
