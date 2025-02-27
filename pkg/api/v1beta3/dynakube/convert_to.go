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

// Convertto converts to the Hub version (v1beta4) to this version (v1beta3).
func (src *DynaKube) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*dynakubev1beta4.DynaKube)

	src.toStatus(dst)

	src.toBase(dst)
	src.toMetadataEnrichment(dst)
	src.toLogMonitoringSpec(dst)
	src.toKspmSpec(dst)
	src.toExtensionsSpec(dst)
	src.toOneAgentSpec(dst)
	src.toActiveGateSpec(dst)

	return nil
}

func (src *DynaKube) toBase(dst *dynakubev1beta4.DynaKube) {
	if dst.Annotations == nil {
		dst.Annotations = map[string]string{}
	}

	src.ObjectMeta = *dst.ObjectMeta.DeepCopy() // DeepCopy mainly relevant for testing

	src.Spec.Proxy = dst.Spec.Proxy
	src.Spec.DynatraceApiRequestThreshold = dst.Spec.DynatraceApiRequestThreshold
	src.Spec.APIURL = dst.Spec.APIURL
	src.Spec.Tokens = dst.Spec.Tokens
	src.Spec.TrustedCAs = dst.Spec.TrustedCAs
	src.Spec.NetworkZone = dst.Spec.NetworkZone
	src.Spec.CustomPullSecret = dst.Spec.CustomPullSecret
	src.Spec.SkipCertCheck = dst.Spec.SkipCertCheck
	src.Spec.EnableIstio = dst.Spec.EnableIstio
}

func (src *DynaKube) toLogMonitoringSpec(dst *dynakubev1beta4.DynaKube) {
	if dst.Spec.LogMonitoring != nil {
		src.Spec.LogMonitoring = &logmonitoring.Spec{}
		src.Spec.LogMonitoring.IngestRuleMatchers = make([]logmonitoring.IngestRuleMatchers, 0)

		for _, rule := range dst.Spec.LogMonitoring.IngestRuleMatchers {
			src.Spec.LogMonitoring.IngestRuleMatchers = append(src.Spec.LogMonitoring.IngestRuleMatchers, logmonitoring.IngestRuleMatchers{
				Attribute: rule.Attribute,
				Values:    rule.Values,
			})
		}
	}
}

func (src *DynaKube) toKspmSpec(dst *dynakubev1beta4.DynaKube) {
	if dst.Spec.Kspm != nil {
		src.Spec.Kspm = &kspm.Spec{}
	}
}

func (src *DynaKube) toExtensionsSpec(dst *dynakubev1beta4.DynaKube) {
	if dst.Spec.Extensions != nil {
		src.Spec.Extensions = &ExtensionsSpec{}
	}
}

func (src *DynaKube) toOneAgentSpec(dst *dynakubev1beta4.DynaKube) { //nolint:dupl
	switch {
	case dst.OneAgent().IsClassicFullStackMode():
		src.Spec.OneAgent.ClassicFullStack = toHostInjectSpec(*dst.Spec.OneAgent.ClassicFullStack)
	case dst.OneAgent().IsCloudNativeFullstackMode():
		src.Spec.OneAgent.CloudNativeFullStack = &oneagent.CloudNativeFullStackSpec{}
		src.Spec.OneAgent.CloudNativeFullStack.HostInjectSpec = *toHostInjectSpec(dst.Spec.OneAgent.CloudNativeFullStack.HostInjectSpec)
		src.Spec.OneAgent.CloudNativeFullStack.AppInjectionSpec = *toAppInjectSpec(dst.Spec.OneAgent.CloudNativeFullStack.AppInjectionSpec)
	case dst.OneAgent().IsApplicationMonitoringMode():
		src.Spec.OneAgent.ApplicationMonitoring = &oneagent.ApplicationMonitoringSpec{}
		src.Spec.OneAgent.ApplicationMonitoring.Version = dst.Spec.OneAgent.ApplicationMonitoring.Version
		src.Spec.OneAgent.ApplicationMonitoring.AppInjectionSpec = *toAppInjectSpec(dst.Spec.OneAgent.ApplicationMonitoring.AppInjectionSpec)
	case dst.OneAgent().IsHostMonitoringMode():
		src.Spec.OneAgent.HostMonitoring = toHostInjectSpec(*dst.Spec.OneAgent.HostMonitoring)
	}
	src.Spec.OneAgent.HostGroup = dst.Spec.OneAgent.HostGroup
}

func (src *DynaKube) toTemplatesSpec(dst *dynakubev1beta4.DynaKube) {
	src.Spec.Templates.LogMonitoring = toLogMonitoringTemplate(dst.Spec.Templates.LogMonitoring)
	src.Spec.Templates.KspmNodeConfigurationCollector = toKspmNodeConfigurationCollectorTemplate(dst.Spec.Templates.KspmNodeConfigurationCollector)
	src.Spec.Templates.OpenTelemetryCollector = toOpenTelemetryCollectorTemplate(dst.Spec.Templates.OpenTelemetryCollector)
	src.Spec.Templates.ExtensionExecutionController = toExtensionControllerTemplate(dst.Spec.Templates.ExtensionExecutionController)
}

func toLogMonitoringTemplate(dst *logmonitoringv1beta4.TemplateSpec) *logmonitoring.TemplateSpec {
	if dst == nil {
		return nil
	}

	src := &logmonitoring.TemplateSpec{}

	src.Annotations = dst.Annotations
	src.Labels = dst.Labels
	src.NodeSelector = dst.NodeSelector
	src.ImageRef = dst.ImageRef
	src.DNSPolicy = dst.DNSPolicy
	src.PriorityClassName = dst.PriorityClassName
	src.SecCompProfile = dst.SecCompProfile
	src.Resources = dst.Resources
	src.Tolerations = dst.Tolerations
	src.Args = dst.Args

	return src
}

func toKspmNodeConfigurationCollectorTemplate(dst kspmv1beta4.NodeConfigurationCollectorSpec) kspm.NodeConfigurationCollectorSpec {
	src := kspm.NodeConfigurationCollectorSpec{}

	src.UpdateStrategy = dst.UpdateStrategy
	src.Labels = dst.Labels
	src.Annotations = dst.Annotations
	src.NodeSelector = dst.NodeSelector
	src.ImageRef = dst.ImageRef
	src.PriorityClassName = dst.PriorityClassName
	src.Resources = dst.Resources
	src.NodeAffinity = dst.NodeAffinity
	src.Tolerations = dst.Tolerations
	src.Args = dst.Args
	src.Env = dst.Env

	return src
}

func toOpenTelemetryCollectorTemplate(dst dynakubev1beta4.OpenTelemetryCollectorSpec) OpenTelemetryCollectorSpec {
	src := OpenTelemetryCollectorSpec{}

	src.Labels = dst.Labels
	src.Annotations = dst.Annotations
	src.Replicas = dst.Replicas
	src.ImageRef = dst.ImageRef
	src.TlsRefName = dst.TlsRefName
	src.Resources = dst.Resources
	src.Tolerations = dst.Tolerations
	src.TopologySpreadConstraints = dst.TopologySpreadConstraints

	return src
}

func toExtensionControllerTemplate(dst dynakubev1beta4.ExtensionExecutionControllerSpec) ExtensionExecutionControllerSpec {
	src := ExtensionExecutionControllerSpec{}

	src.PersistentVolumeClaim = dst.PersistentVolumeClaim
	src.Labels = dst.Labels
	src.Annotations = dst.Annotations
	src.ImageRef = dst.ImageRef
	src.TlsRefName = dst.TlsRefName
	src.CustomConfig = dst.CustomConfig
	src.CustomExtensionCertificates = dst.CustomExtensionCertificates
	src.Resources = dst.Resources
	src.Tolerations = dst.Tolerations
	src.TopologySpreadConstraints = dst.TopologySpreadConstraints
	src.UseEphemeralVolume = dst.UseEphemeralVolume

	return src
}

func (src *DynaKube) toActiveGateSpec(dst *dynakubev1beta4.DynaKube) {
	src.Spec.ActiveGate.Annotations = dst.Spec.ActiveGate.Annotations
	src.Spec.ActiveGate.TlsSecretName = dst.Spec.ActiveGate.TlsSecretName
	src.Spec.ActiveGate.DNSPolicy = dst.Spec.ActiveGate.DNSPolicy
	src.Spec.ActiveGate.PriorityClassName = dst.Spec.ActiveGate.PriorityClassName

	src.Spec.ActiveGate.CapabilityProperties.CustomProperties = dst.Spec.ActiveGate.CapabilityProperties.CustomProperties
	src.Spec.ActiveGate.CapabilityProperties.NodeSelector = dst.Spec.ActiveGate.CapabilityProperties.NodeSelector
	src.Spec.ActiveGate.CapabilityProperties.Labels = dst.Spec.ActiveGate.CapabilityProperties.Labels
	src.Spec.ActiveGate.CapabilityProperties.Replicas = dst.Spec.ActiveGate.CapabilityProperties.Replicas
	src.Spec.ActiveGate.CapabilityProperties.Image = dst.Spec.ActiveGate.CapabilityProperties.Image
	src.Spec.ActiveGate.CapabilityProperties.Group = dst.Spec.ActiveGate.CapabilityProperties.Group
	src.Spec.ActiveGate.CapabilityProperties.Resources = dst.Spec.ActiveGate.CapabilityProperties.Resources
	src.Spec.ActiveGate.CapabilityProperties.Tolerations = dst.Spec.ActiveGate.CapabilityProperties.Tolerations
	src.Spec.ActiveGate.CapabilityProperties.Env = dst.Spec.ActiveGate.CapabilityProperties.Env
	src.Spec.ActiveGate.CapabilityProperties.TopologySpreadConstraints = dst.Spec.ActiveGate.CapabilityProperties.TopologySpreadConstraints

	src.Spec.ActiveGate.Capabilities = make([]activegate.CapabilityDisplayName, 0)
	for _, capability := range dst.Spec.ActiveGate.Capabilities {
		src.Spec.ActiveGate.Capabilities = append(src.Spec.ActiveGate.Capabilities, activegate.CapabilityDisplayName(capability))
	}
}

func (src *DynaKube) toStatus(dst *dynakubev1beta4.DynaKube) {
	src.toOneAgentStatus(*dst)
	src.toActiveGateStatus(*dst)
	src.Status.CodeModules = oneagent.CodeModulesStatus{
		VersionStatus: dst.Status.CodeModules.VersionStatus,
	}

	src.Status.MetadataEnrichment.Rules = make([]EnrichmentRule, 0)
	for _, rule := range dst.Status.MetadataEnrichment.Rules {
		src.Status.MetadataEnrichment.Rules = append(src.Status.MetadataEnrichment.Rules,
			EnrichmentRule{
				Type:    EnrichmentRuleType(rule.Type),
				Source:  rule.Source,
				Target:  rule.Target,
				Enabled: rule.Enabled,
			})
	}

	src.Status.Kspm.TokenSecretHash = dst.Status.Kspm.TokenSecretHash
	src.Status.UpdatedTimestamp = dst.Status.UpdatedTimestamp
	src.Status.DynatraceApi = DynatraceApiStatus{
		LastTokenScopeRequest: dst.Status.DynatraceApi.LastTokenScopeRequest,
	}
	src.Status.Phase = dst.Status.Phase
	src.Status.KubeSystemUUID = dst.Status.KubeSystemUUID
	src.Status.KubernetesClusterMEID = dst.Status.KubernetesClusterMEID
	src.Status.KubernetesClusterName = dst.Status.KubernetesClusterName
	src.Status.Conditions = dst.Status.Conditions
}

func (src *DynaKube) toOneAgentStatus(dst dynakubev1beta4.DynaKube) {
	src.Status.OneAgent.VersionStatus = dst.Status.OneAgent.VersionStatus

	src.Status.OneAgent.Instances = map[string]oneagent.Instance{}
	for key, instance := range dst.Status.OneAgent.Instances {
		tmp := oneagent.Instance{
			PodName:   instance.PodName,
			IPAddress: instance.IPAddress,
		}
		src.Status.OneAgent.Instances[key] = tmp
	}

	src.Status.OneAgent.LastInstanceStatusUpdate = dst.Status.OneAgent.LastInstanceStatusUpdate
	src.Status.OneAgent.Healthcheck = dst.Status.OneAgent.Healthcheck

	src.Status.OneAgent.ConnectionInfoStatus.ConnectionInfo = dst.Status.OneAgent.ConnectionInfoStatus.ConnectionInfo
	src.Status.OneAgent.ConnectionInfoStatus.CommunicationHosts = make([]oneagent.CommunicationHostStatus, 0)
	for _, host := range dst.Status.OneAgent.ConnectionInfoStatus.CommunicationHosts {
		src.Status.OneAgent.ConnectionInfoStatus.CommunicationHosts =
			append(src.Status.OneAgent.ConnectionInfoStatus.CommunicationHosts, oneagent.CommunicationHostStatus{
				Protocol: host.Protocol,
				Host:     host.Host,
				Port:     host.Port,
			})
	}
}

func (src *DynaKube) toActiveGateStatus(dst dynakubev1beta4.DynaKube) {
	src.Status.ActiveGate.VersionStatus = dst.Status.ActiveGate.VersionStatus
	src.Status.ActiveGate.ConnectionInfo = dst.Status.ActiveGate.ConnectionInfo
	src.Status.ActiveGate.ServiceIPs = dst.Status.ActiveGate.ServiceIPs
}

func toHostInjectSpec(dst oneagentv1beta4.HostInjectSpec) *oneagent.HostInjectSpec {
	src := &oneagent.HostInjectSpec{}

	src.Annotations = dst.Annotations
	src.Labels = dst.Labels
	src.NodeSelector = dst.NodeSelector
	src.AutoUpdate = dst.AutoUpdate
	src.Version = dst.Version
	src.Image = dst.Image
	src.DNSPolicy = dst.DNSPolicy
	src.PriorityClassName = dst.PriorityClassName
	src.SecCompProfile = dst.SecCompProfile
	src.OneAgentResources = dst.OneAgentResources
	src.Tolerations = dst.Tolerations
	src.Env = dst.Env
	src.Args = dst.Args

	return src
}

func toAppInjectSpec(dst oneagentv1beta4.AppInjectionSpec) *oneagent.AppInjectionSpec {
	src := &oneagent.AppInjectionSpec{}

	src.InitResources = dst.InitResources
	src.CodeModulesImage = dst.CodeModulesImage
	src.NamespaceSelector = dst.NamespaceSelector

	return src
}

func (src *DynaKube) toMetadataEnrichment(dst *dynakubev1beta4.DynaKube) {
	src.Spec.MetadataEnrichment.Enabled = dst.Spec.MetadataEnrichment.Enabled
	src.Spec.MetadataEnrichment.NamespaceSelector = dst.Spec.MetadataEnrichment.NamespaceSelector
}
