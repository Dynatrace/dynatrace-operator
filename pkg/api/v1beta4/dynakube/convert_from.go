package dynakube

import (
	dynakubelatest "github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	kspmlatest "github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/kspm"
	logmonitoringlatest "github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/logmonitoring"
	oneagentlatest "github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube/kspm"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube/logmonitoring"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube/oneagent"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

// ConvertFrom converts from the Hub version (latest) to this version (v1beta4).
func (dk *DynaKube) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*dynakubelatest.DynaKube)

	dk.fromStatus(src)

	dk.fromBase(src)
	dk.fromMetadataEnrichment(src)
	dk.fromLogMonitoringSpec(src)
	dk.fromKspmSpec(src)
	dk.fromExtensionsSpec(src)
	dk.fromOneAgentSpec(src)
	dk.fromActiveGateSpec(src)
	dk.fromTemplatesSpec(src)

	return nil
}

func (dk *DynaKube) fromBase(src *dynakubelatest.DynaKube) {
	if src.Annotations == nil {
		src.Annotations = map[string]string{}
	}

	dk.ObjectMeta = *src.ObjectMeta.DeepCopy() // DeepCopy mainly relevant for testing

	dk.Spec.Proxy = src.Spec.Proxy
	dk.Spec.DynatraceAPIRequestThreshold = src.Spec.DynatraceAPIRequestThreshold
	dk.Spec.APIURL = src.Spec.APIURL
	dk.Spec.Tokens = src.Spec.Tokens
	dk.Spec.TrustedCAs = src.Spec.TrustedCAs
	dk.Spec.NetworkZone = src.Spec.NetworkZone
	dk.Spec.CustomPullSecret = src.Spec.CustomPullSecret
	dk.Spec.SkipCertCheck = src.Spec.SkipCertCheck
	dk.Spec.EnableIstio = src.Spec.EnableIstio
}

func (dk *DynaKube) fromLogMonitoringSpec(src *dynakubelatest.DynaKube) {
	if src.Spec.LogMonitoring != nil {
		dk.Spec.LogMonitoring = &logmonitoring.Spec{}
		dk.Spec.LogMonitoring.IngestRuleMatchers = make([]logmonitoring.IngestRuleMatchers, 0)

		for _, rule := range src.Spec.LogMonitoring.IngestRuleMatchers {
			dk.Spec.LogMonitoring.IngestRuleMatchers = append(dk.Spec.LogMonitoring.IngestRuleMatchers, logmonitoring.IngestRuleMatchers{
				Attribute: rule.Attribute,
				Values:    rule.Values,
			})
		}
	}
}

func (dk *DynaKube) fromKspmSpec(src *dynakubelatest.DynaKube) {
	if src.Spec.Kspm != nil {
		dk.Spec.Kspm = &kspm.Spec{}
	}
}

func (dk *DynaKube) fromExtensionsSpec(src *dynakubelatest.DynaKube) {
	if src.Spec.Extensions != nil {
		dk.Spec.Extensions = &ExtensionsSpec{}
	}
}

func (dk *DynaKube) fromOneAgentSpec(src *dynakubelatest.DynaKube) { //nolint:dupl
	switch {
	case src.OneAgent().IsClassicFullStackMode():
		dk.Spec.OneAgent.ClassicFullStack = fromHostInjectSpec(*src.Spec.OneAgent.ClassicFullStack)
	case src.OneAgent().IsCloudNativeFullstackMode():
		dk.Spec.OneAgent.CloudNativeFullStack = &oneagent.CloudNativeFullStackSpec{}
		dk.Spec.OneAgent.CloudNativeFullStack.HostInjectSpec = *fromHostInjectSpec(src.Spec.OneAgent.CloudNativeFullStack.HostInjectSpec)
		dk.Spec.OneAgent.CloudNativeFullStack.AppInjectionSpec = *fromAppInjectSpec(src.Spec.OneAgent.CloudNativeFullStack.AppInjectionSpec)
	case src.OneAgent().IsApplicationMonitoringMode():
		dk.Spec.OneAgent.ApplicationMonitoring = &oneagent.ApplicationMonitoringSpec{}
		dk.Spec.OneAgent.ApplicationMonitoring.Version = src.Spec.OneAgent.ApplicationMonitoring.Version
		dk.Spec.OneAgent.ApplicationMonitoring.AppInjectionSpec = *fromAppInjectSpec(src.Spec.OneAgent.ApplicationMonitoring.AppInjectionSpec)
	case src.OneAgent().IsHostMonitoringMode():
		dk.Spec.OneAgent.HostMonitoring = fromHostInjectSpec(*src.Spec.OneAgent.HostMonitoring)
	}

	dk.Spec.OneAgent.HostGroup = src.Spec.OneAgent.HostGroup
}

func (dk *DynaKube) fromTemplatesSpec(src *dynakubelatest.DynaKube) {
	dk.Spec.Templates.LogMonitoring = fromLogMonitoringTemplate(src.Spec.Templates.LogMonitoring)
	dk.Spec.Templates.KspmNodeConfigurationCollector = fromKspmNodeConfigurationCollectorTemplate(src.Spec.Templates.KspmNodeConfigurationCollector)
	dk.Spec.Templates.OpenTelemetryCollector = fromOpenTelemetryCollectorTemplate(src.Spec.Templates.OpenTelemetryCollector)
	dk.Spec.Templates.ExtensionExecutionController = fromExtensionControllerTemplate(src.Spec.Templates.ExtensionExecutionController)
}

func fromLogMonitoringTemplate(src *logmonitoringlatest.TemplateSpec) *logmonitoring.TemplateSpec {
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

func fromKspmNodeConfigurationCollectorTemplate(src kspmlatest.NodeConfigurationCollectorSpec) kspm.NodeConfigurationCollectorSpec {
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

func fromOpenTelemetryCollectorTemplate(src dynakubelatest.OpenTelemetryCollectorSpec) OpenTelemetryCollectorSpec {
	dst := OpenTelemetryCollectorSpec{}

	dst.Labels = src.Labels
	dst.Annotations = src.Annotations
	dst.Replicas = src.Replicas
	dst.ImageRef = src.ImageRef
	dst.TLSRefName = src.TLSRefName
	dst.Resources = src.Resources
	dst.Tolerations = src.Tolerations
	dst.TopologySpreadConstraints = src.TopologySpreadConstraints

	return dst
}

func fromExtensionControllerTemplate(src dynakubelatest.ExtensionExecutionControllerSpec) ExtensionExecutionControllerSpec {
	dst := ExtensionExecutionControllerSpec{}

	dst.PersistentVolumeClaim = src.PersistentVolumeClaim
	dst.Labels = src.Labels
	dst.Annotations = src.Annotations
	dst.ImageRef = src.ImageRef
	dst.TLSRefName = src.TLSRefName
	dst.CustomConfig = src.CustomConfig
	dst.CustomExtensionCertificates = src.CustomExtensionCertificates
	dst.Resources = src.Resources
	dst.Tolerations = src.Tolerations
	dst.TopologySpreadConstraints = src.TopologySpreadConstraints
	dst.UseEphemeralVolume = src.UseEphemeralVolume

	return dst
}

func (dk *DynaKube) fromActiveGateSpec(src *dynakubelatest.DynaKube) { //nolint:dupl
	dk.Spec.ActiveGate.Annotations = src.Spec.ActiveGate.Annotations
	dk.Spec.ActiveGate.TLSSecretName = src.Spec.ActiveGate.TLSSecretName
	dk.Spec.ActiveGate.DNSPolicy = src.Spec.ActiveGate.DNSPolicy
	dk.Spec.ActiveGate.PriorityClassName = src.Spec.ActiveGate.PriorityClassName
	dk.Spec.ActiveGate.PersistentVolumeClaim = src.Spec.ActiveGate.VolumeClaimTemplate

	dk.Spec.ActiveGate.CustomProperties = src.Spec.ActiveGate.CustomProperties
	dk.Spec.ActiveGate.NodeSelector = src.Spec.ActiveGate.NodeSelector
	dk.Spec.ActiveGate.Labels = src.Spec.ActiveGate.Labels
	dk.Spec.ActiveGate.Replicas = src.Spec.ActiveGate.Replicas
	dk.Spec.ActiveGate.Image = src.Spec.ActiveGate.Image
	dk.Spec.ActiveGate.Group = src.Spec.ActiveGate.Group
	dk.Spec.ActiveGate.Resources = src.Spec.ActiveGate.Resources
	dk.Spec.ActiveGate.Tolerations = src.Spec.ActiveGate.Tolerations
	dk.Spec.ActiveGate.Env = src.Spec.ActiveGate.Env
	dk.Spec.ActiveGate.TopologySpreadConstraints = src.Spec.ActiveGate.TopologySpreadConstraints

	dk.Spec.ActiveGate.Capabilities = make([]activegate.CapabilityDisplayName, 0)
	for _, capability := range src.Spec.ActiveGate.Capabilities {
		dk.Spec.ActiveGate.Capabilities = append(dk.Spec.ActiveGate.Capabilities, activegate.CapabilityDisplayName(capability))
	}
}

func (dk *DynaKube) fromStatus(src *dynakubelatest.DynaKube) {
	dk.fromOneAgentStatus(*src)
	dk.fromActiveGateStatus(*src)
	dk.Status.CodeModules = oneagent.CodeModulesStatus{
		VersionStatus: src.Status.CodeModules.VersionStatus,
	}

	dk.Status.MetadataEnrichment.Rules = make([]EnrichmentRule, 0)
	for _, rule := range src.Status.MetadataEnrichment.Rules {
		dk.Status.MetadataEnrichment.Rules = append(dk.Status.MetadataEnrichment.Rules,
			EnrichmentRule{
				Type:   EnrichmentRuleType(rule.Type),
				Source: rule.Source,
				Target: rule.Target,
			})
	}

	dk.Status.Kspm.TokenSecretHash = src.Status.Kspm.TokenSecretHash
	dk.Status.UpdatedTimestamp = src.Status.UpdatedTimestamp
	dk.Status.DynatraceAPI = DynatraceAPIStatus{
		LastTokenScopeRequest: src.Status.DynatraceAPI.LastTokenScopeRequest,
	}
	dk.Status.Phase = src.Status.Phase
	dk.Status.KubeSystemUUID = src.Status.KubeSystemUUID
	dk.Status.KubernetesClusterMEID = src.Status.KubernetesClusterMEID
	dk.Status.KubernetesClusterName = src.Status.KubernetesClusterName
	dk.Status.Conditions = src.Status.Conditions
}

func (dk *DynaKube) fromOneAgentStatus(src dynakubelatest.DynaKube) { //nolint:dupl
	dk.Status.OneAgent.VersionStatus = src.Status.OneAgent.VersionStatus

	dk.Status.OneAgent.Instances = map[string]oneagent.Instance{}
	for key, instance := range src.Status.OneAgent.Instances {
		dk.Status.OneAgent.Instances[key] = oneagent.Instance{
			PodName:   instance.PodName,
			IPAddress: instance.IPAddress,
		}
	}

	dk.Status.OneAgent.LastInstanceStatusUpdate = src.Status.OneAgent.LastInstanceStatusUpdate
	dk.Status.OneAgent.Healthcheck = src.Status.OneAgent.Healthcheck
	dk.Status.OneAgent.ConnectionInfoStatus.ConnectionInfo = src.Status.OneAgent.ConnectionInfoStatus.ConnectionInfo
	dk.Status.OneAgent.ConnectionInfoStatus.CommunicationHosts = make([]oneagent.CommunicationHostStatus, 0)

	for _, host := range src.Status.OneAgent.ConnectionInfoStatus.CommunicationHosts {
		dk.Status.OneAgent.ConnectionInfoStatus.CommunicationHosts =
			append(dk.Status.OneAgent.ConnectionInfoStatus.CommunicationHosts, oneagent.CommunicationHostStatus{
				Protocol: host.Protocol,
				Host:     host.Host,
				Port:     host.Port,
			})
	}
}

func (dk *DynaKube) fromActiveGateStatus(src dynakubelatest.DynaKube) {
	dk.Status.ActiveGate.VersionStatus = src.Status.ActiveGate.VersionStatus
	dk.Status.ActiveGate.ConnectionInfo = src.Status.ActiveGate.ConnectionInfo
	dk.Status.ActiveGate.ServiceIPs = src.Status.ActiveGate.ServiceIPs
}

func fromHostInjectSpec(src oneagentlatest.HostInjectSpec) *oneagent.HostInjectSpec {
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

func fromAppInjectSpec(src oneagentlatest.AppInjectionSpec) *oneagent.AppInjectionSpec {
	dst := &oneagent.AppInjectionSpec{}

	dst.InitResources = src.InitResources
	dst.CodeModulesImage = src.CodeModulesImage
	dst.NamespaceSelector = src.NamespaceSelector

	return dst
}

func (dk *DynaKube) fromMetadataEnrichment(src *dynakubelatest.DynaKube) {
	dk.Spec.MetadataEnrichment.Enabled = src.Spec.MetadataEnrichment.Enabled
	dk.Spec.MetadataEnrichment.NamespaceSelector = src.Spec.MetadataEnrichment.NamespaceSelector
}
