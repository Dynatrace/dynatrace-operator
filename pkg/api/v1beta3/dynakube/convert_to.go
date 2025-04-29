package dynakube

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube/kspm"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube/logmonitoring"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube/oneagent"
	dynakubev1beta4 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube"
	activegatev1beta4 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube/activegate"
	kspmv1beta4 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube/kspm"
	logmonitoringv1beta4 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube/logmonitoring"
	oneagentv1beta4 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube/oneagent"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

// Convertto converts this version (src=v1beta3) to the Hub version.
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
	src.toTemplatesSpec(dst)

	return nil
}

func (src *DynaKube) toBase(dst *dynakubev1beta4.DynaKube) {
	if src.Annotations == nil {
		dst.Annotations = map[string]string{}
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

func (src *DynaKube) toLogMonitoringSpec(dst *dynakubev1beta4.DynaKube) {
	if src.Spec.LogMonitoring != nil {
		dst.Spec.LogMonitoring = &logmonitoringv1beta4.Spec{}
		dst.Spec.LogMonitoring.IngestRuleMatchers = make([]logmonitoringv1beta4.IngestRuleMatchers, 0)

		for _, rule := range src.Spec.LogMonitoring.IngestRuleMatchers {
			dst.Spec.LogMonitoring.IngestRuleMatchers = append(dst.Spec.LogMonitoring.IngestRuleMatchers, logmonitoringv1beta4.IngestRuleMatchers{
				Attribute: rule.Attribute,
				Values:    rule.Values,
			})
		}
	}
}

func (src *DynaKube) toKspmSpec(dst *dynakubev1beta4.DynaKube) {
	if src.Spec.Kspm != nil {
		dst.Spec.Kspm = &kspmv1beta4.Spec{}
	}
}

func (src *DynaKube) toExtensionsSpec(dst *dynakubev1beta4.DynaKube) {
	if src.Spec.Extensions != nil {
		dst.Spec.Extensions = &dynakubev1beta4.ExtensionsSpec{}
	}
}

func (src *DynaKube) toOneAgentSpec(dst *dynakubev1beta4.DynaKube) { //nolint:dupl
	switch {
	case src.OneAgent().IsClassicFullStackMode():
		dst.Spec.OneAgent.ClassicFullStack = toHostInjectSpec(*src.Spec.OneAgent.ClassicFullStack)
	case src.OneAgent().IsCloudNativeFullstackMode():
		dst.Spec.OneAgent.CloudNativeFullStack = &oneagentv1beta4.CloudNativeFullStackSpec{}
		dst.Spec.OneAgent.CloudNativeFullStack.HostInjectSpec = *toHostInjectSpec(src.Spec.OneAgent.CloudNativeFullStack.HostInjectSpec)
		dst.Spec.OneAgent.CloudNativeFullStack.AppInjectionSpec = *toAppInjectSpec(src.Spec.OneAgent.CloudNativeFullStack.AppInjectionSpec)
	case src.OneAgent().IsApplicationMonitoringMode():
		dst.Spec.OneAgent.ApplicationMonitoring = &oneagentv1beta4.ApplicationMonitoringSpec{}
		dst.Spec.OneAgent.ApplicationMonitoring.Version = src.Spec.OneAgent.ApplicationMonitoring.Version
		dst.Spec.OneAgent.ApplicationMonitoring.AppInjectionSpec = *toAppInjectSpec(src.Spec.OneAgent.ApplicationMonitoring.AppInjectionSpec)
	case src.OneAgent().IsHostMonitoringMode():
		dst.Spec.OneAgent.HostMonitoring = toHostInjectSpec(*src.Spec.OneAgent.HostMonitoring)
	}

	dst.Spec.OneAgent.HostGroup = src.Spec.OneAgent.HostGroup
}

func (src *DynaKube) toTemplatesSpec(dst *dynakubev1beta4.DynaKube) {
	dst.Spec.Templates.LogMonitoring = toLogMonitoringTemplate(src.Spec.Templates.LogMonitoring)
	dst.Spec.Templates.KspmNodeConfigurationCollector = toKspmNodeConfigurationCollectorTemplate(src.Spec.Templates.KspmNodeConfigurationCollector)
	dst.Spec.Templates.OpenTelemetryCollector = toOpenTelemetryCollectorTemplate(src.Spec.Templates.OpenTelemetryCollector)
	dst.Spec.Templates.ExtensionExecutionController = toExtensionControllerTemplate(src.Spec.Templates.ExtensionExecutionController)
}

func toLogMonitoringTemplate(src *logmonitoring.TemplateSpec) *logmonitoringv1beta4.TemplateSpec {
	if src == nil {
		return nil
	}

	dst := &logmonitoringv1beta4.TemplateSpec{}

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

func toKspmNodeConfigurationCollectorTemplate(src kspm.NodeConfigurationCollectorSpec) kspmv1beta4.NodeConfigurationCollectorSpec {
	dst := kspmv1beta4.NodeConfigurationCollectorSpec{}

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

func toOpenTelemetryCollectorTemplate(src OpenTelemetryCollectorSpec) dynakubev1beta4.OpenTelemetryCollectorSpec {
	dst := dynakubev1beta4.OpenTelemetryCollectorSpec{}

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

func toExtensionControllerTemplate(src ExtensionExecutionControllerSpec) dynakubev1beta4.ExtensionExecutionControllerSpec {
	dst := dynakubev1beta4.ExtensionExecutionControllerSpec{}

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

func (src *DynaKube) toActiveGateSpec(dst *dynakubev1beta4.DynaKube) { //nolint:dupl
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

	dst.Spec.ActiveGate.Capabilities = make([]activegatev1beta4.CapabilityDisplayName, 0)
	for _, capability := range src.Spec.ActiveGate.Capabilities {
		dst.Spec.ActiveGate.Capabilities = append(dst.Spec.ActiveGate.Capabilities, activegatev1beta4.CapabilityDisplayName(capability))
	}
}

func (src *DynaKube) toStatus(dst *dynakubev1beta4.DynaKube) {
	src.toOneAgentStatus(dst)
	src.toActiveGateStatus(dst)
	dst.Status.CodeModules = oneagentv1beta4.CodeModulesStatus{
		VersionStatus: src.Status.CodeModules.VersionStatus,
	}

	dst.Status.MetadataEnrichment.Rules = make([]dynakubev1beta4.EnrichmentRule, 0)
	for _, rule := range src.Status.MetadataEnrichment.Rules {
		dst.Status.MetadataEnrichment.Rules = append(dst.Status.MetadataEnrichment.Rules,
			dynakubev1beta4.EnrichmentRule{
				Type:   dynakubev1beta4.EnrichmentRuleType(rule.Type),
				Source: rule.Source,
				Target: rule.Target,
			})
	}

	dst.Status.Kspm.TokenSecretHash = src.Status.Kspm.TokenSecretHash
	dst.Status.UpdatedTimestamp = src.Status.UpdatedTimestamp
	dst.Status.DynatraceApi = dynakubev1beta4.DynatraceApiStatus{
		LastTokenScopeRequest: src.Status.DynatraceApi.LastTokenScopeRequest,
	}
	dst.Status.Phase = src.Status.Phase
	dst.Status.KubeSystemUUID = src.Status.KubeSystemUUID
	dst.Status.KubernetesClusterMEID = src.Status.KubernetesClusterMEID
	dst.Status.KubernetesClusterName = src.Status.KubernetesClusterName
	dst.Status.Conditions = src.Status.Conditions
}

func (src *DynaKube) toOneAgentStatus(dst *dynakubev1beta4.DynaKube) { //nolint:dupl
	dst.Status.OneAgent.VersionStatus = src.Status.OneAgent.VersionStatus

	dst.Status.OneAgent.Instances = map[string]oneagentv1beta4.Instance{}
	for key, instance := range src.Status.OneAgent.Instances {
		dst.Status.OneAgent.Instances[key] = oneagentv1beta4.Instance{
			PodName:   instance.PodName,
			IPAddress: instance.IPAddress,
		}
	}

	dst.Status.OneAgent.LastInstanceStatusUpdate = src.Status.OneAgent.LastInstanceStatusUpdate
	dst.Status.OneAgent.Healthcheck = src.Status.OneAgent.Healthcheck

	dst.Status.OneAgent.ConnectionInfoStatus.ConnectionInfo = src.Status.OneAgent.ConnectionInfoStatus.ConnectionInfo
	dst.Status.OneAgent.ConnectionInfoStatus.CommunicationHosts = make([]oneagentv1beta4.CommunicationHostStatus, 0)

	for _, host := range src.Status.OneAgent.ConnectionInfoStatus.CommunicationHosts {
		dst.Status.OneAgent.ConnectionInfoStatus.CommunicationHosts =
			append(dst.Status.OneAgent.ConnectionInfoStatus.CommunicationHosts, oneagentv1beta4.CommunicationHostStatus{
				Protocol: host.Protocol,
				Host:     host.Host,
				Port:     host.Port,
			})
	}
}

func (src *DynaKube) toActiveGateStatus(dst *dynakubev1beta4.DynaKube) {
	dst.Status.ActiveGate.VersionStatus = src.Status.ActiveGate.VersionStatus
	dst.Status.ActiveGate.ConnectionInfo = src.Status.ActiveGate.ConnectionInfo
	dst.Status.ActiveGate.ServiceIPs = src.Status.ActiveGate.ServiceIPs
}

func toHostInjectSpec(src oneagent.HostInjectSpec) *oneagentv1beta4.HostInjectSpec {
	dst := &oneagentv1beta4.HostInjectSpec{}

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

func toAppInjectSpec(src oneagent.AppInjectionSpec) *oneagentv1beta4.AppInjectionSpec {
	dst := &oneagentv1beta4.AppInjectionSpec{}

	dst.InitResources = src.InitResources
	dst.CodeModulesImage = src.CodeModulesImage
	dst.NamespaceSelector = src.NamespaceSelector

	return dst
}

func (src *DynaKube) toMetadataEnrichment(dst *dynakubev1beta4.DynaKube) {
	dst.Spec.MetadataEnrichment.Enabled = src.Spec.MetadataEnrichment.Enabled
	dst.Spec.MetadataEnrichment.NamespaceSelector = src.Spec.MetadataEnrichment.NamespaceSelector
}
