package dynakube

import (
	dynakubelatest "github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	activegatelatest "github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
	extensionslatest "github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/extensions"
	kspmlatest "github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/kspm"
	logmonitoringlatest "github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/logmonitoring"
	metadataenrichmentlatest "github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/metadataenrichment"
	oneagentlatest "github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube/kspm"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube/logmonitoring"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube/oneagent"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

// Convertto converts this version (src=v1beta3) to the Hub version.
func (src *DynaKube) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*dynakubelatest.DynaKube)

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

func (src *DynaKube) toBase(dst *dynakubelatest.DynaKube) {
	if src.Annotations == nil {
		dst.Annotations = map[string]string{}
	}

	dst.ObjectMeta = *src.ObjectMeta.DeepCopy() // DeepCopy mainly relevant for testing

	dst.Spec.Proxy = src.Spec.Proxy
	dst.Spec.DynatraceAPIRequestThreshold = src.Spec.DynatraceAPIRequestThreshold
	dst.Spec.APIURL = src.Spec.APIURL
	dst.Spec.Tokens = src.Spec.Tokens
	dst.Spec.TrustedCAs = src.Spec.TrustedCAs
	dst.Spec.NetworkZone = src.Spec.NetworkZone
	dst.Spec.CustomPullSecret = src.Spec.CustomPullSecret
	dst.Spec.SkipCertCheck = src.Spec.SkipCertCheck
	dst.Spec.EnableIstio = src.Spec.EnableIstio
}

func (src *DynaKube) toLogMonitoringSpec(dst *dynakubelatest.DynaKube) {
	if src.Spec.LogMonitoring != nil {
		dst.Spec.LogMonitoring = &logmonitoringlatest.Spec{}
		dst.Spec.LogMonitoring.IngestRuleMatchers = make([]logmonitoringlatest.IngestRuleMatchers, 0)

		for _, rule := range src.Spec.LogMonitoring.IngestRuleMatchers {
			dst.Spec.LogMonitoring.IngestRuleMatchers = append(dst.Spec.LogMonitoring.IngestRuleMatchers, logmonitoringlatest.IngestRuleMatchers{
				Attribute: rule.Attribute,
				Values:    rule.Values,
			})
		}
	}
}

func (src *DynaKube) toKspmSpec(dst *dynakubelatest.DynaKube) {
	if src.Spec.Kspm != nil {
		dst.Spec.Kspm = &kspmlatest.Spec{}
		dst.Spec.Kspm.MappedHostPaths = []string{"/"}
	}
}

func (src *DynaKube) toExtensionsSpec(dst *dynakubelatest.DynaKube) {
	if src.Spec.Extensions != nil {
		dst.Spec.Extensions = &extensionslatest.Spec{
			Prometheus: &extensionslatest.PrometheusSpec{},
		}
	}
}

func (src *DynaKube) toOneAgentSpec(dst *dynakubelatest.DynaKube) { //nolint:dupl
	switch {
	case src.OneAgent().IsClassicFullStackMode():
		dst.Spec.OneAgent.ClassicFullStack = toHostInjectSpec(*src.Spec.OneAgent.ClassicFullStack)
		dst.RemovedFields().AutoUpdate.Set(src.Spec.OneAgent.ClassicFullStack.AutoUpdate)
	case src.OneAgent().IsCloudNativeFullstackMode():
		dst.Spec.OneAgent.CloudNativeFullStack = &oneagentlatest.CloudNativeFullStackSpec{}
		dst.Spec.OneAgent.CloudNativeFullStack.HostInjectSpec = *toHostInjectSpec(src.Spec.OneAgent.CloudNativeFullStack.HostInjectSpec)
		dst.RemovedFields().AutoUpdate.Set(src.Spec.OneAgent.CloudNativeFullStack.AutoUpdate)
		dst.Spec.OneAgent.CloudNativeFullStack.AppInjectionSpec = *toAppInjectSpec(src.Spec.OneAgent.CloudNativeFullStack.AppInjectionSpec)
	case src.OneAgent().IsApplicationMonitoringMode():
		dst.Spec.OneAgent.ApplicationMonitoring = &oneagentlatest.ApplicationMonitoringSpec{}
		dst.Spec.OneAgent.ApplicationMonitoring.Version = src.Spec.OneAgent.ApplicationMonitoring.Version
		dst.Spec.OneAgent.ApplicationMonitoring.AppInjectionSpec = *toAppInjectSpec(src.Spec.OneAgent.ApplicationMonitoring.AppInjectionSpec)
	case src.OneAgent().IsHostMonitoringMode():
		dst.Spec.OneAgent.HostMonitoring = toHostInjectSpec(*src.Spec.OneAgent.HostMonitoring)
		dst.RemovedFields().AutoUpdate.Set(src.Spec.OneAgent.HostMonitoring.AutoUpdate)
	}

	dst.Spec.OneAgent.HostGroup = src.Spec.OneAgent.HostGroup
}

func (src *DynaKube) toTemplatesSpec(dst *dynakubelatest.DynaKube) {
	dst.Spec.Templates.LogMonitoring = toLogMonitoringTemplate(src.Spec.Templates.LogMonitoring)
	dst.Spec.Templates.KspmNodeConfigurationCollector = toKspmNodeConfigurationCollectorTemplate(src.Spec.Templates.KspmNodeConfigurationCollector)
	dst.Spec.Templates.OpenTelemetryCollector = toOpenTelemetryCollectorTemplate(dst, src.Spec.Templates.OpenTelemetryCollector)
	dst.Spec.Templates.ExtensionExecutionController = toExtensionControllerTemplate(src.Spec.Templates.ExtensionExecutionController)
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
	dst.NodeAffinity = &src.NodeAffinity
	dst.Tolerations = src.Tolerations
	dst.Args = src.Args
	dst.Env = src.Env

	return dst
}

func toOpenTelemetryCollectorTemplate(dk *dynakubelatest.DynaKube, src OpenTelemetryCollectorSpec) dynakubelatest.OpenTelemetryCollectorSpec {
	dst := dynakubelatest.OpenTelemetryCollectorSpec{}

	dst.Labels = src.Labels
	dst.Annotations = src.Annotations
	dst.Replicas = src.Replicas
	dst.ImageRef = src.ImageRef
	if dst.ImageRef.IsZero() {
		dst.ImageRef.Repository = "public.ecr.aws/dynatrace/dynatrace-otel-collector"
		dst.ImageRef.Tag = "latest"
		dk.RemovedFields().DefaultOTELCImage.Set(ptr.To(true))
	}
	dst.TLSRefName = src.TlsRefName
	dst.Resources = src.Resources
	dst.Tolerations = src.Tolerations
	dst.TopologySpreadConstraints = src.TopologySpreadConstraints

	return dst
}

func toExtensionControllerTemplate(src ExtensionExecutionControllerSpec) extensionslatest.ExecutionControllerSpec {
	dst := extensionslatest.ExecutionControllerSpec{}

	dst.PersistentVolumeClaim = src.PersistentVolumeClaim
	dst.Labels = src.Labels
	dst.Annotations = src.Annotations
	dst.ImageRef = src.ImageRef
	dst.TLSRefName = src.TlsRefName
	dst.CustomConfig = src.CustomConfig
	dst.CustomExtensionCertificates = src.CustomExtensionCertificates
	dst.Resources = src.Resources
	dst.Tolerations = src.Tolerations
	dst.TopologySpreadConstraints = src.TopologySpreadConstraints
	dst.UseEphemeralVolume = src.UseEphemeralVolume

	return dst
}

func (src *DynaKube) toActiveGateSpec(dst *dynakubelatest.DynaKube) { //nolint:dupl
	dst.Spec.ActiveGate.Annotations = src.Spec.ActiveGate.Annotations
	dst.Spec.ActiveGate.TLSSecretName = src.Spec.ActiveGate.TLSSecretName
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

	dst.Spec.ActiveGate.Capabilities = make([]activegatelatest.CapabilityDisplayName, 0)
	for _, capability := range src.Spec.ActiveGate.Capabilities {
		dst.Spec.ActiveGate.Capabilities = append(dst.Spec.ActiveGate.Capabilities, activegatelatest.CapabilityDisplayName(capability))
	}
}

func (src *DynaKube) toStatus(dst *dynakubelatest.DynaKube) {
	src.toOneAgentStatus(dst)
	src.toActiveGateStatus(dst)
	dst.Status.CodeModules = oneagentlatest.CodeModulesStatus{
		VersionStatus: src.Status.CodeModules.VersionStatus,
	}

	dst.Status.MetadataEnrichment.Rules = make([]metadataenrichmentlatest.Rule, 0)
	for _, rule := range src.Status.MetadataEnrichment.Rules {
		dst.Status.MetadataEnrichment.Rules = append(dst.Status.MetadataEnrichment.Rules,
			metadataenrichmentlatest.Rule{
				Type:   metadataenrichmentlatest.RuleType(rule.Type),
				Source: rule.Source,
				Target: rule.Target,
			})
	}

	dst.Status.Kspm.TokenSecretHash = src.Status.Kspm.TokenSecretHash
	dst.Status.UpdatedTimestamp = src.Status.UpdatedTimestamp
	dst.Status.DynatraceAPI = dynakubelatest.DynatraceAPIStatus{
		LastTokenScopeRequest: src.Status.DynatraceAPI.LastTokenScopeRequest,
	}
	dst.Status.Phase = src.Status.Phase
	dst.Status.KubeSystemUUID = src.Status.KubeSystemUUID
	dst.Status.KubernetesClusterMEID = src.Status.KubernetesClusterMEID
	dst.Status.KubernetesClusterName = src.Status.KubernetesClusterName
	dst.Status.Conditions = src.Status.Conditions
}

func (src *DynaKube) toOneAgentStatus(dst *dynakubelatest.DynaKube) { //nolint:dupl
	dst.Status.OneAgent.VersionStatus = src.Status.OneAgent.VersionStatus

	dst.Status.OneAgent.Instances = map[string]oneagentlatest.Instance{}
	for key, instance := range src.Status.OneAgent.Instances {
		dst.Status.OneAgent.Instances[key] = oneagentlatest.Instance{
			PodName:   instance.PodName,
			IPAddress: instance.IPAddress,
		}
	}

	dst.Status.OneAgent.LastInstanceStatusUpdate = src.Status.OneAgent.LastInstanceStatusUpdate
	dst.Status.OneAgent.Healthcheck = src.Status.OneAgent.Healthcheck

	dst.Status.OneAgent.ConnectionInfoStatus.ConnectionInfo = src.Status.OneAgent.ConnectionInfoStatus.ConnectionInfo
	dst.Status.OneAgent.ConnectionInfoStatus.CommunicationHosts = make([]oneagentlatest.CommunicationHostStatus, 0)

	for _, host := range src.Status.OneAgent.ConnectionInfoStatus.CommunicationHosts {
		dst.Status.OneAgent.ConnectionInfoStatus.CommunicationHosts =
			append(dst.Status.OneAgent.ConnectionInfoStatus.CommunicationHosts, oneagentlatest.CommunicationHostStatus{
				Protocol: host.Protocol,
				Host:     host.Host,
				Port:     host.Port,
			})
	}
}

func (src *DynaKube) toActiveGateStatus(dst *dynakubelatest.DynaKube) {
	dst.Status.ActiveGate.VersionStatus = src.Status.ActiveGate.VersionStatus
	dst.Status.ActiveGate.ConnectionInfo = src.Status.ActiveGate.ConnectionInfo
	dst.Status.ActiveGate.ServiceIPs = src.Status.ActiveGate.ServiceIPs
}

func toHostInjectSpec(src oneagent.HostInjectSpec) *oneagentlatest.HostInjectSpec {
	dst := &oneagentlatest.HostInjectSpec{}

	dst.Annotations = src.Annotations
	dst.Labels = src.Labels
	dst.NodeSelector = src.NodeSelector
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

func (src *DynaKube) toMetadataEnrichment(dst *dynakubelatest.DynaKube) {
	dst.Spec.MetadataEnrichment.Enabled = src.Spec.MetadataEnrichment.Enabled
	dst.Spec.MetadataEnrichment.NamespaceSelector = src.Spec.MetadataEnrichment.NamespaceSelector
}
