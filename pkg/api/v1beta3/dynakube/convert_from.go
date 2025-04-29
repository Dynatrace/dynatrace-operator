package dynakube

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube/kspm"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube/logmonitoring"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube/oneagent"
	dynakubev1beta4 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube"
	kspmv1beta4 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube/kspm"
	logmonitoringv1beta4 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube/logmonitoring"
	oneagentv1beta4 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube/oneagent"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

// ConvertFrom converts from the Hub version (v1beta4) to this version (v1beta3).
func (dst *DynaKube) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*dynakubev1beta4.DynaKube)

	dst.fromStatus(src)

	dst.fromBase(src)
	dst.fromMetadataEnrichment(src)
	dst.fromLogMonitoringSpec(src)
	dst.fromKspmSpec(src)
	dst.fromExtensionsSpec(src)
	dst.fromOneAgentSpec(src)
	dst.fromActiveGateSpec(src)
	dst.fromTemplatesSpec(src)

	return nil
}

func (dst *DynaKube) fromBase(src *dynakubev1beta4.DynaKube) {
	if src.Annotations == nil {
		src.Annotations = map[string]string{}
	}

	dst.ObjectMeta = *src.ObjectMeta.DeepCopy() // DeepCopy mainly relevant for testing

	dst.Spec.Proxy = src.Spec.Proxy
	dst.Spec.DynatraceApiRequestThreshold = src.Spec.DynatraceApiRequestThreshold
	dst.Spec.APIURL = src.Spec.APIURL
	dst.Spec.Tokens = src.Spec.Tokens
	dst.Spec.TrustedCAs = src.Spec.TrustedCAs
	dst.Spec.NetworkZone = src.Spec.NetworkZone
	dst.Spec.CustomPullSecret = src.Spec.CustomPullSecret
	dst.Spec.SkipCertCheck = src.Spec.SkipCertCheck
	dst.Spec.EnableIstio = src.Spec.EnableIstio
}

func (dst *DynaKube) fromLogMonitoringSpec(src *dynakubev1beta4.DynaKube) {
	if src.Spec.LogMonitoring != nil {
		dst.Spec.LogMonitoring = &logmonitoring.Spec{}
		dst.Spec.LogMonitoring.IngestRuleMatchers = make([]logmonitoring.IngestRuleMatchers, 0)

		for _, rule := range src.Spec.LogMonitoring.IngestRuleMatchers {
			dst.Spec.LogMonitoring.IngestRuleMatchers = append(dst.Spec.LogMonitoring.IngestRuleMatchers, logmonitoring.IngestRuleMatchers{
				Attribute: rule.Attribute,
				Values:    rule.Values,
			})
		}
	}
}

func (dst *DynaKube) fromKspmSpec(src *dynakubev1beta4.DynaKube) {
	if src.Spec.Kspm != nil {
		dst.Spec.Kspm = &kspm.Spec{}
	}
}

func (dst *DynaKube) fromExtensionsSpec(src *dynakubev1beta4.DynaKube) {
	if src.Spec.Extensions != nil {
		dst.Spec.Extensions = &ExtensionsSpec{}
	}
}

func (dst *DynaKube) fromOneAgentSpec(src *dynakubev1beta4.DynaKube) { //nolint:dupl
	switch {
	case src.OneAgent().IsClassicFullStackMode():
		dst.Spec.OneAgent.ClassicFullStack = fromHostInjectSpec(*src.Spec.OneAgent.ClassicFullStack)
	case src.OneAgent().IsCloudNativeFullstackMode():
		dst.Spec.OneAgent.CloudNativeFullStack = &oneagent.CloudNativeFullStackSpec{}
		dst.Spec.OneAgent.CloudNativeFullStack.HostInjectSpec = *fromHostInjectSpec(src.Spec.OneAgent.CloudNativeFullStack.HostInjectSpec)
		dst.Spec.OneAgent.CloudNativeFullStack.AppInjectionSpec = *fromAppInjectSpec(src.Spec.OneAgent.CloudNativeFullStack.AppInjectionSpec)
	case src.OneAgent().IsApplicationMonitoringMode():
		dst.Spec.OneAgent.ApplicationMonitoring = &oneagent.ApplicationMonitoringSpec{}
		dst.Spec.OneAgent.ApplicationMonitoring.Version = src.Spec.OneAgent.ApplicationMonitoring.Version
		dst.Spec.OneAgent.ApplicationMonitoring.AppInjectionSpec = *fromAppInjectSpec(src.Spec.OneAgent.ApplicationMonitoring.AppInjectionSpec)
	case src.OneAgent().IsHostMonitoringMode():
		dst.Spec.OneAgent.HostMonitoring = fromHostInjectSpec(*src.Spec.OneAgent.HostMonitoring)
	}

	dst.Spec.OneAgent.HostGroup = src.Spec.OneAgent.HostGroup
}

func (dst *DynaKube) fromTemplatesSpec(src *dynakubev1beta4.DynaKube) {
	dst.Spec.Templates.LogMonitoring = fromLogMonitoringTemplate(src.Spec.Templates.LogMonitoring)
	dst.Spec.Templates.KspmNodeConfigurationCollector = fromKspmNodeConfigurationCollectorTemplate(src.Spec.Templates.KspmNodeConfigurationCollector)
	dst.Spec.Templates.OpenTelemetryCollector = fromOpenTelemetryCollectorTemplate(src.Spec.Templates.OpenTelemetryCollector)
	dst.Spec.Templates.ExtensionExecutionController = fromExtensionControllerTemplate(src.Spec.Templates.ExtensionExecutionController)
}

func fromLogMonitoringTemplate(src *logmonitoringv1beta4.TemplateSpec) *logmonitoring.TemplateSpec {
	if src == nil {
		return nil
	}

	dst := &logmonitoring.TemplateSpec{}

	dst.Annotations = src.Annotations
	dst.Labels = src.Labels
	dst.NodeSelector = src.NodeSelector
	dst.ImageRef = src.ImageRef
	dst.DNSPolicy = src.DNSPolicy
	dst.PriorityClassName = src.PriorityClassName
	dst.SecCompProfile = src.SecCompProfile
	dst.Resources = src.Resources
	dst.Tolerations = src.Tolerations
	dst.Args = src.Args

	return dst
}

func fromKspmNodeConfigurationCollectorTemplate(src kspmv1beta4.NodeConfigurationCollectorSpec) kspm.NodeConfigurationCollectorSpec {
	dst := kspm.NodeConfigurationCollectorSpec{}

	dst.UpdateStrategy = src.UpdateStrategy
	dst.Labels = src.Labels
	dst.Annotations = src.Annotations
	dst.NodeSelector = src.NodeSelector
	dst.ImageRef = src.ImageRef
	dst.PriorityClassName = src.PriorityClassName
	dst.Resources = src.Resources
	dst.NodeAffinity = src.NodeAffinity
	dst.Tolerations = src.Tolerations
	dst.Args = src.Args
	dst.Env = src.Env

	return dst
}

func fromOpenTelemetryCollectorTemplate(src dynakubev1beta4.OpenTelemetryCollectorSpec) OpenTelemetryCollectorSpec {
	dst := OpenTelemetryCollectorSpec{}

	dst.Labels = src.Labels
	dst.Annotations = src.Annotations
	dst.Replicas = src.Replicas
	dst.ImageRef = src.ImageRef
	dst.TlsRefName = src.TlsRefName
	dst.Resources = src.Resources
	dst.Tolerations = src.Tolerations
	dst.TopologySpreadConstraints = src.TopologySpreadConstraints

	return dst
}

func fromExtensionControllerTemplate(src dynakubev1beta4.ExtensionExecutionControllerSpec) ExtensionExecutionControllerSpec {
	dst := ExtensionExecutionControllerSpec{}

	dst.PersistentVolumeClaim = src.PersistentVolumeClaim
	dst.Labels = src.Labels
	dst.Annotations = src.Annotations
	dst.ImageRef = src.ImageRef
	dst.TlsRefName = src.TlsRefName
	dst.CustomConfig = src.CustomConfig
	dst.CustomExtensionCertificates = src.CustomExtensionCertificates
	dst.Resources = src.Resources
	dst.Tolerations = src.Tolerations
	dst.TopologySpreadConstraints = src.TopologySpreadConstraints
	dst.UseEphemeralVolume = src.UseEphemeralVolume

	return dst
}

func (dst *DynaKube) fromActiveGateSpec(src *dynakubev1beta4.DynaKube) { //nolint:dupl
	dst.Spec.ActiveGate.Annotations = src.Spec.ActiveGate.Annotations
	dst.Spec.ActiveGate.TlsSecretName = src.Spec.ActiveGate.TlsSecretName
	dst.Spec.ActiveGate.DNSPolicy = src.Spec.ActiveGate.DNSPolicy
	dst.Spec.ActiveGate.PriorityClassName = src.Spec.ActiveGate.PriorityClassName

	dst.Spec.ActiveGate.CapabilityProperties.CustomProperties = src.Spec.ActiveGate.CapabilityProperties.CustomProperties
	dst.Spec.ActiveGate.CapabilityProperties.NodeSelector = src.Spec.ActiveGate.CapabilityProperties.NodeSelector
	dst.Spec.ActiveGate.CapabilityProperties.Labels = src.Spec.ActiveGate.CapabilityProperties.Labels
	dst.Spec.ActiveGate.CapabilityProperties.Replicas = src.Spec.ActiveGate.CapabilityProperties.Replicas
	dst.Spec.ActiveGate.CapabilityProperties.Image = src.Spec.ActiveGate.CapabilityProperties.Image
	dst.Spec.ActiveGate.CapabilityProperties.Group = src.Spec.ActiveGate.CapabilityProperties.Group
	dst.Spec.ActiveGate.CapabilityProperties.Resources = src.Spec.ActiveGate.CapabilityProperties.Resources
	dst.Spec.ActiveGate.CapabilityProperties.Tolerations = src.Spec.ActiveGate.CapabilityProperties.Tolerations
	dst.Spec.ActiveGate.CapabilityProperties.Env = src.Spec.ActiveGate.CapabilityProperties.Env
	dst.Spec.ActiveGate.CapabilityProperties.TopologySpreadConstraints = src.Spec.ActiveGate.CapabilityProperties.TopologySpreadConstraints

	dst.Spec.ActiveGate.Capabilities = make([]activegate.CapabilityDisplayName, 0)
	for _, capability := range src.Spec.ActiveGate.Capabilities {
		dst.Spec.ActiveGate.Capabilities = append(dst.Spec.ActiveGate.Capabilities, activegate.CapabilityDisplayName(capability))
	}
}

func (dst *DynaKube) fromStatus(src *dynakubev1beta4.DynaKube) {
	dst.fromOneAgentStatus(*src)
	dst.fromActiveGateStatus(*src)
	dst.Status.CodeModules = oneagent.CodeModulesStatus{
		VersionStatus: src.Status.CodeModules.VersionStatus,
	}

	dst.Status.MetadataEnrichment.Rules = make([]EnrichmentRule, 0)
	for _, rule := range src.Status.MetadataEnrichment.Rules {
		dst.Status.MetadataEnrichment.Rules = append(dst.Status.MetadataEnrichment.Rules,
			EnrichmentRule{
				Type:   EnrichmentRuleType(rule.Type),
				Source: rule.Source,
				Target: rule.Target,
			})
	}

	dst.Status.Kspm.TokenSecretHash = src.Status.Kspm.TokenSecretHash
	dst.Status.UpdatedTimestamp = src.Status.UpdatedTimestamp
	dst.Status.DynatraceApi = DynatraceApiStatus{
		LastTokenScopeRequest: src.Status.DynatraceApi.LastTokenScopeRequest,
	}
	dst.Status.Phase = src.Status.Phase
	dst.Status.KubeSystemUUID = src.Status.KubeSystemUUID
	dst.Status.KubernetesClusterMEID = src.Status.KubernetesClusterMEID
	dst.Status.KubernetesClusterName = src.Status.KubernetesClusterName
	dst.Status.Conditions = src.Status.Conditions
}

func (dst *DynaKube) fromOneAgentStatus(src dynakubev1beta4.DynaKube) { //nolint:dupl
	dst.Status.OneAgent.VersionStatus = src.Status.OneAgent.VersionStatus

	dst.Status.OneAgent.Instances = map[string]oneagent.Instance{}
	for key, instance := range src.Status.OneAgent.Instances {
		dst.Status.OneAgent.Instances[key] = oneagent.Instance{
			PodName:   instance.PodName,
			IPAddress: instance.IPAddress,
		}
	}

	dst.Status.OneAgent.LastInstanceStatusUpdate = src.Status.OneAgent.LastInstanceStatusUpdate
	dst.Status.OneAgent.Healthcheck = src.Status.OneAgent.Healthcheck
	dst.Status.OneAgent.ConnectionInfoStatus.ConnectionInfo = src.Status.OneAgent.ConnectionInfoStatus.ConnectionInfo
	dst.Status.OneAgent.ConnectionInfoStatus.CommunicationHosts = make([]oneagent.CommunicationHostStatus, 0)

	for _, host := range src.Status.OneAgent.ConnectionInfoStatus.CommunicationHosts {
		dst.Status.OneAgent.ConnectionInfoStatus.CommunicationHosts =
			append(dst.Status.OneAgent.ConnectionInfoStatus.CommunicationHosts, oneagent.CommunicationHostStatus{
				Protocol: host.Protocol,
				Host:     host.Host,
				Port:     host.Port,
			})
	}
}

func (dst *DynaKube) fromActiveGateStatus(src dynakubev1beta4.DynaKube) {
	dst.Status.ActiveGate.VersionStatus = src.Status.ActiveGate.VersionStatus
	dst.Status.ActiveGate.ConnectionInfo = src.Status.ActiveGate.ConnectionInfo
	dst.Status.ActiveGate.ServiceIPs = src.Status.ActiveGate.ServiceIPs
}

func fromHostInjectSpec(src oneagentv1beta4.HostInjectSpec) *oneagent.HostInjectSpec {
	dst := &oneagent.HostInjectSpec{}

	dst.Annotations = src.Annotations
	dst.Labels = src.Labels
	dst.NodeSelector = src.NodeSelector
	dst.AutoUpdate = src.AutoUpdate
	dst.Version = src.Version
	dst.Image = src.Image
	dst.DNSPolicy = src.DNSPolicy
	dst.PriorityClassName = src.PriorityClassName
	dst.SecCompProfile = src.SecCompProfile
	dst.OneAgentResources = src.OneAgentResources
	dst.Tolerations = src.Tolerations
	dst.Env = src.Env
	dst.Args = src.Args

	return dst
}

func fromAppInjectSpec(src oneagentv1beta4.AppInjectionSpec) *oneagent.AppInjectionSpec {
	dst := &oneagent.AppInjectionSpec{}

	dst.InitResources = src.InitResources
	dst.CodeModulesImage = src.CodeModulesImage
	dst.NamespaceSelector = src.NamespaceSelector

	return dst
}

func (dst *DynaKube) fromMetadataEnrichment(src *dynakubev1beta4.DynaKube) {
	dst.Spec.MetadataEnrichment.Enabled = src.Spec.MetadataEnrichment.Enabled
	dst.Spec.MetadataEnrichment.NamespaceSelector = src.Spec.MetadataEnrichment.NamespaceSelector
}
