package dynakube

import (
	dynakubelatest "github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	activegatelatest "github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
	kspmlatest "github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/kspm"
	logmonitoringlatest "github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/logmonitoring"
	oneagentlatest "github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	telemetryingestlatest "github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/telemetryingest"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube/kspm"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube/logmonitoring"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube/oneagent"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

// Convertto converts this version (src=v1beta4) to the Hub version.
func (dk *DynaKube) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*dynakubelatest.DynaKube)

	dk.toStatus(dst)

	dk.toBase(dst)
	dk.toMetadataEnrichment(dst)
	dk.toLogMonitoringSpec(dst)
	dk.toKspmSpec(dst)
	dk.toExtensionsSpec(dst)
	dk.toOneAgentSpec(dst)
	dk.toActiveGateSpec(dst)
	dk.toTemplatesSpec(dst)
	dk.toTelemetryIngestSpec(dst)

	return nil
}

func (dk *DynaKube) toBase(dst *dynakubelatest.DynaKube) {
	if dk.Annotations == nil {
		dst.Annotations = map[string]string{}
	}

	dst.ObjectMeta = *dk.ObjectMeta.DeepCopy() // DeepCopy mainly relevant for testing

	dst.Spec.Proxy = dk.Spec.Proxy
	dst.Spec.DynatraceAPIRequestThreshold = dk.Spec.DynatraceAPIRequestThreshold
	dst.Spec.APIURL = dk.Spec.APIURL
	dst.Spec.Tokens = dk.Spec.Tokens
	dst.Spec.TrustedCAs = dk.Spec.TrustedCAs
	dst.Spec.NetworkZone = dk.Spec.NetworkZone
	dst.Spec.CustomPullSecret = dk.Spec.CustomPullSecret
	dst.Spec.SkipCertCheck = dk.Spec.SkipCertCheck
	dst.Spec.EnableIstio = dk.Spec.EnableIstio
}

func (dk *DynaKube) toLogMonitoringSpec(dst *dynakubelatest.DynaKube) {
	if dk.Spec.LogMonitoring != nil {
		dst.Spec.LogMonitoring = &logmonitoringlatest.Spec{}
		dst.Spec.LogMonitoring.IngestRuleMatchers = make([]logmonitoringlatest.IngestRuleMatchers, 0)

		for _, rule := range dk.Spec.LogMonitoring.IngestRuleMatchers {
			dst.Spec.LogMonitoring.IngestRuleMatchers = append(dst.Spec.LogMonitoring.IngestRuleMatchers, logmonitoringlatest.IngestRuleMatchers{
				Attribute: rule.Attribute,
				Values:    rule.Values,
			})
		}
	}
}

func (dk *DynaKube) toKspmSpec(dst *dynakubelatest.DynaKube) {
	if dk.Spec.Kspm != nil {
		dst.Spec.Kspm = &kspmlatest.Spec{}
		dst.Spec.Kspm.MappedHostPaths = []string{"/"}
	}
}

func (dk *DynaKube) toExtensionsSpec(dst *dynakubelatest.DynaKube) {
	if dk.Spec.Extensions != nil {
		dst.Spec.Extensions = &dynakubelatest.ExtensionsSpec{}
	}
}

func (dk *DynaKube) toOneAgentSpec(dst *dynakubelatest.DynaKube) { //nolint:dupl
	switch {
	case dk.OneAgent().IsClassicFullStackMode():
		dst.Spec.OneAgent.ClassicFullStack = toHostInjectSpec(*dk.Spec.OneAgent.ClassicFullStack)
	case dk.OneAgent().IsCloudNativeFullstackMode():
		dst.Spec.OneAgent.CloudNativeFullStack = &oneagentlatest.CloudNativeFullStackSpec{}
		dst.Spec.OneAgent.CloudNativeFullStack.HostInjectSpec = *toHostInjectSpec(dk.Spec.OneAgent.CloudNativeFullStack.HostInjectSpec)
		dst.Spec.OneAgent.CloudNativeFullStack.AppInjectionSpec = *toAppInjectSpec(dk.Spec.OneAgent.CloudNativeFullStack.AppInjectionSpec)
	case dk.OneAgent().IsApplicationMonitoringMode():
		dst.Spec.OneAgent.ApplicationMonitoring = &oneagentlatest.ApplicationMonitoringSpec{}
		dst.Spec.OneAgent.ApplicationMonitoring.Version = dk.Spec.OneAgent.ApplicationMonitoring.Version
		dst.Spec.OneAgent.ApplicationMonitoring.AppInjectionSpec = *toAppInjectSpec(dk.Spec.OneAgent.ApplicationMonitoring.AppInjectionSpec)
	case dk.OneAgent().IsHostMonitoringMode():
		dst.Spec.OneAgent.HostMonitoring = toHostInjectSpec(*dk.Spec.OneAgent.HostMonitoring)
	}

	dst.Spec.OneAgent.HostGroup = dk.Spec.OneAgent.HostGroup
}

func (dk *DynaKube) toTemplatesSpec(dst *dynakubelatest.DynaKube) {
	dst.Spec.Templates.LogMonitoring = toLogMonitoringTemplate(dk.Spec.Templates.LogMonitoring)
	dst.Spec.Templates.KspmNodeConfigurationCollector = toKspmNodeConfigurationCollectorTemplate(dk.Spec.Templates.KspmNodeConfigurationCollector)
	dst.Spec.Templates.OpenTelemetryCollector = toOpenTelemetryCollectorTemplate(dk.Spec.Templates.OpenTelemetryCollector)
	dst.Spec.Templates.ExtensionExecutionController = toExtensionControllerTemplate(dk.Spec.Templates.ExtensionExecutionController)
}

func toLogMonitoringTemplate(src *logmonitoring.TemplateSpec) *logmonitoringlatest.TemplateSpec {
	if src == nil {
		return nil
	}

	dst := &logmonitoringlatest.TemplateSpec{}

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

func toKspmNodeConfigurationCollectorTemplate(src kspm.NodeConfigurationCollectorSpec) kspmlatest.NodeConfigurationCollectorSpec {
	dst := kspmlatest.NodeConfigurationCollectorSpec{}

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

func toOpenTelemetryCollectorTemplate(src OpenTelemetryCollectorSpec) dynakubelatest.OpenTelemetryCollectorSpec {
	dst := dynakubelatest.OpenTelemetryCollectorSpec{}

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

func toExtensionControllerTemplate(src ExtensionExecutionControllerSpec) dynakubelatest.ExtensionExecutionControllerSpec {
	dst := dynakubelatest.ExtensionExecutionControllerSpec{}

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

func (dk *DynaKube) toActiveGateSpec(dst *dynakubelatest.DynaKube) { //nolint:dupl
	dst.Spec.ActiveGate.Annotations = dk.Spec.ActiveGate.Annotations
	dst.Spec.ActiveGate.TLSSecretName = dk.Spec.ActiveGate.TLSSecretName
	dst.Spec.ActiveGate.DNSPolicy = dk.Spec.ActiveGate.DNSPolicy
	dst.Spec.ActiveGate.PriorityClassName = dk.Spec.ActiveGate.PriorityClassName
	dst.Spec.ActiveGate.VolumeClaimTemplate = dk.Spec.ActiveGate.PersistentVolumeClaim

	dst.Spec.ActiveGate.CustomProperties = dk.Spec.ActiveGate.CustomProperties
	dst.Spec.ActiveGate.NodeSelector = dk.Spec.ActiveGate.NodeSelector
	dst.Spec.ActiveGate.Labels = dk.Spec.ActiveGate.Labels
	dst.Spec.ActiveGate.Replicas = dk.Spec.ActiveGate.Replicas
	dst.Spec.ActiveGate.Image = dk.Spec.ActiveGate.Image
	dst.Spec.ActiveGate.Group = dk.Spec.ActiveGate.Group
	dst.Spec.ActiveGate.Resources = dk.Spec.ActiveGate.Resources
	dst.Spec.ActiveGate.Tolerations = dk.Spec.ActiveGate.Tolerations
	dst.Spec.ActiveGate.Env = dk.Spec.ActiveGate.Env
	dst.Spec.ActiveGate.TopologySpreadConstraints = dk.Spec.ActiveGate.TopologySpreadConstraints

	dst.Spec.ActiveGate.Capabilities = make([]activegatelatest.CapabilityDisplayName, 0)
	for _, capability := range dk.Spec.ActiveGate.Capabilities {
		dst.Spec.ActiveGate.Capabilities = append(dst.Spec.ActiveGate.Capabilities, activegatelatest.CapabilityDisplayName(capability))
	}
}

func (dk *DynaKube) toStatus(dst *dynakubelatest.DynaKube) {
	dk.toOneAgentStatus(dst)
	dk.toActiveGateStatus(dst)
	dst.Status.CodeModules = oneagentlatest.CodeModulesStatus{
		VersionStatus: dk.Status.CodeModules.VersionStatus,
	}

	dst.Status.MetadataEnrichment.Rules = make([]dynakubelatest.EnrichmentRule, 0)
	for _, rule := range dk.Status.MetadataEnrichment.Rules {
		dst.Status.MetadataEnrichment.Rules = append(dst.Status.MetadataEnrichment.Rules,
			dynakubelatest.EnrichmentRule{
				Type:   dynakubelatest.EnrichmentRuleType(rule.Type),
				Source: rule.Source,
				Target: rule.Target,
			})
	}

	dst.Status.Kspm.TokenSecretHash = dk.Status.Kspm.TokenSecretHash
	dst.Status.UpdatedTimestamp = dk.Status.UpdatedTimestamp
	dst.Status.DynatraceAPI = dynakubelatest.DynatraceAPIStatus{
		LastTokenScopeRequest: dk.Status.DynatraceAPI.LastTokenScopeRequest,
	}
	dst.Status.Phase = dk.Status.Phase
	dst.Status.KubeSystemUUID = dk.Status.KubeSystemUUID
	dst.Status.KubernetesClusterMEID = dk.Status.KubernetesClusterMEID
	dst.Status.KubernetesClusterName = dk.Status.KubernetesClusterName
	dst.Status.Conditions = dk.Status.Conditions
}

func (dk *DynaKube) toOneAgentStatus(dst *dynakubelatest.DynaKube) { //nolint:dupl
	dst.Status.OneAgent.VersionStatus = dk.Status.OneAgent.VersionStatus

	dst.Status.OneAgent.Instances = map[string]oneagentlatest.Instance{}
	for key, instance := range dk.Status.OneAgent.Instances {
		dst.Status.OneAgent.Instances[key] = oneagentlatest.Instance{
			PodName:   instance.PodName,
			IPAddress: instance.IPAddress,
		}
	}

	dst.Status.OneAgent.LastInstanceStatusUpdate = dk.Status.OneAgent.LastInstanceStatusUpdate
	dst.Status.OneAgent.Healthcheck = dk.Status.OneAgent.Healthcheck

	dst.Status.OneAgent.ConnectionInfoStatus.ConnectionInfo = dk.Status.OneAgent.ConnectionInfoStatus.ConnectionInfo
	dst.Status.OneAgent.ConnectionInfoStatus.CommunicationHosts = make([]oneagentlatest.CommunicationHostStatus, 0)

	for _, host := range dk.Status.OneAgent.ConnectionInfoStatus.CommunicationHosts {
		dst.Status.OneAgent.ConnectionInfoStatus.CommunicationHosts =
			append(dst.Status.OneAgent.ConnectionInfoStatus.CommunicationHosts, oneagentlatest.CommunicationHostStatus{
				Protocol: host.Protocol,
				Host:     host.Host,
				Port:     host.Port,
			})
	}
}

func (dk *DynaKube) toActiveGateStatus(dst *dynakubelatest.DynaKube) {
	dst.Status.ActiveGate.VersionStatus = dk.Status.ActiveGate.VersionStatus
	dst.Status.ActiveGate.ConnectionInfo = dk.Status.ActiveGate.ConnectionInfo
	dst.Status.ActiveGate.ServiceIPs = dk.Status.ActiveGate.ServiceIPs
}

func toHostInjectSpec(src oneagent.HostInjectSpec) *oneagentlatest.HostInjectSpec {
	dst := &oneagentlatest.HostInjectSpec{}

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

func toAppInjectSpec(src oneagent.AppInjectionSpec) *oneagentlatest.AppInjectionSpec {
	dst := &oneagentlatest.AppInjectionSpec{}

	dst.InitResources = src.InitResources
	dst.CodeModulesImage = src.CodeModulesImage
	dst.NamespaceSelector = src.NamespaceSelector

	return dst
}

func (dk *DynaKube) toMetadataEnrichment(dst *dynakubelatest.DynaKube) {
	dst.Spec.MetadataEnrichment.Enabled = dk.Spec.MetadataEnrichment.Enabled
	dst.Spec.MetadataEnrichment.NamespaceSelector = dk.Spec.MetadataEnrichment.NamespaceSelector
}

func (dk *DynaKube) toTelemetryIngestSpec(dst *dynakubelatest.DynaKube) {
	if dk.Spec.TelemetryIngest != nil {
		dst.Spec.TelemetryIngest = &telemetryingestlatest.Spec{}
		dst.Spec.TelemetryIngest.Protocols = dk.Spec.TelemetryIngest.Protocols
		dst.Spec.TelemetryIngest.ServiceName = dk.Spec.TelemetryIngest.ServiceName
		dst.Spec.TelemetryIngest.TLSRefName = dk.Spec.TelemetryIngest.TLSRefName
	} else {
		dst.Spec.TelemetryIngest = nil
	}
}
