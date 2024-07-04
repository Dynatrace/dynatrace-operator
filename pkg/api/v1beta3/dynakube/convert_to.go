package dynakube

import (
	"encoding/json"

	"github.com/Dynatrace/dynatrace-operator/pkg/api"
	v1beta2 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta2/dynakube"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

// ConvertTo converts this v1beta3.DynaKube to the Hub version (v1beta2.DynaKube).
func (src *DynaKube) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*v1beta2.DynaKube)

	src.toBase(dst)
	src.toOneAgentSpec(dst)
	src.toActiveGateSpec(dst)
	src.toMetadataEnrichment(dst)

	if err := src.toExtensions(dst); err != nil {
		return err
	}

	src.toStatus(dst)

	return nil
}

func (src *DynaKube) toBase(dst *v1beta2.DynaKube) {
	if dst.Annotations == nil {
		dst.Annotations = map[string]string{}
	}

	dst.ObjectMeta = *src.ObjectMeta.DeepCopy() // DeepCopy mainly relevant for testing

	dst.Spec.APIURL = src.Spec.APIURL
	dst.Spec.Tokens = src.Spec.Tokens
	dst.Spec.CustomPullSecret = src.Spec.CustomPullSecret
	dst.Spec.SkipCertCheck = src.Spec.SkipCertCheck
	dst.Spec.Proxy = (*v1beta2.DynaKubeProxy)(src.Spec.Proxy)
	dst.Spec.TrustedCAs = src.Spec.TrustedCAs
	dst.Spec.NetworkZone = src.Spec.NetworkZone
	dst.Spec.EnableIstio = src.Spec.EnableIstio
	dst.Spec.DynatraceApiRequestThreshold = src.Spec.DynatraceApiRequestThreshold
}

func (src *DynaKube) toOneAgentSpec(dst *v1beta2.DynaKube) {
	dst.Spec.OneAgent.HostGroup = src.Spec.OneAgent.HostGroup

	switch {
	case src.HostMonitoringMode():
		dst.Spec.OneAgent.HostMonitoring = toHostInjectSpec(*src.Spec.OneAgent.HostMonitoring)
	case src.ClassicFullStackMode():
		dst.Spec.OneAgent.ClassicFullStack = toHostInjectSpec(*src.Spec.OneAgent.ClassicFullStack)
	case src.CloudNativeFullstackMode():
		dst.Spec.OneAgent.CloudNativeFullStack = &v1beta2.CloudNativeFullStackSpec{}
		dst.Spec.OneAgent.CloudNativeFullStack.HostInjectSpec = *toHostInjectSpec(src.Spec.OneAgent.CloudNativeFullStack.HostInjectSpec)
		dst.Spec.OneAgent.CloudNativeFullStack.AppInjectionSpec = *toAppInjectSpec(src.Spec.OneAgent.CloudNativeFullStack.AppInjectionSpec)
	case src.ApplicationMonitoringMode():
		dst.Spec.OneAgent.ApplicationMonitoring = &v1beta2.ApplicationMonitoringSpec{}
		dst.Spec.OneAgent.ApplicationMonitoring.AppInjectionSpec = *toAppInjectSpec(src.Spec.OneAgent.ApplicationMonitoring.AppInjectionSpec)
		dst.Spec.OneAgent.ApplicationMonitoring.Version = src.Spec.OneAgent.ApplicationMonitoring.Version
		dst.Spec.OneAgent.ApplicationMonitoring.UseCSIDriver = src.Spec.OneAgent.ApplicationMonitoring.UseCSIDriver
	}
}

func (src *DynaKube) toActiveGateSpec(dst *v1beta2.DynaKube) {
	dst.Spec.ActiveGate.Image = src.Spec.ActiveGate.Image
	dst.Spec.ActiveGate.PriorityClassName = src.Spec.ActiveGate.PriorityClassName
	dst.Spec.ActiveGate.TlsSecretName = src.Spec.ActiveGate.TlsSecretName
	dst.Spec.ActiveGate.Group = src.Spec.ActiveGate.Group
	dst.Spec.ActiveGate.Annotations = src.Spec.ActiveGate.Annotations
	dst.Spec.ActiveGate.Tolerations = src.Spec.ActiveGate.Tolerations
	dst.Spec.ActiveGate.NodeSelector = src.Spec.ActiveGate.NodeSelector
	dst.Spec.ActiveGate.Labels = src.Spec.ActiveGate.Labels
	dst.Spec.ActiveGate.Env = src.Spec.ActiveGate.Env
	dst.Spec.ActiveGate.DNSPolicy = src.Spec.ActiveGate.DNSPolicy
	dst.Spec.ActiveGate.TopologySpreadConstraints = src.Spec.ActiveGate.TopologySpreadConstraints
	dst.Spec.ActiveGate.Resources = src.Spec.ActiveGate.Resources
	dst.Spec.ActiveGate.Replicas = src.Spec.ActiveGate.Replicas

	for _, capability := range src.Spec.ActiveGate.Capabilities {
		dst.Spec.ActiveGate.Capabilities = append(dst.Spec.ActiveGate.Capabilities, v1beta2.CapabilityDisplayName(capability))
	}

	if src.Spec.ActiveGate.CustomProperties != nil {
		dst.Spec.ActiveGate.CustomProperties = &v1beta2.DynaKubeValueSource{
			Value:     src.Spec.ActiveGate.CustomProperties.Value,
			ValueFrom: src.Spec.ActiveGate.CustomProperties.ValueFrom,
		}
	}
}

func (src *DynaKube) toStatus(dst *v1beta2.DynaKube) {
	src.toOneAgentStatus(dst)
	src.toActiveGateStatus(dst)
	dst.Status.CodeModules = v1beta2.CodeModulesStatus{
		VersionStatus: src.Status.CodeModules.VersionStatus,
	}

	dst.Status.DynatraceApi = v1beta2.DynatraceApiStatus{
		LastTokenScopeRequest: src.Status.DynatraceApi.LastTokenScopeRequest,
	}

	dst.Status.Conditions = src.Status.Conditions
	dst.Status.Phase = src.Status.Phase
	dst.Status.UpdatedTimestamp = src.Status.UpdatedTimestamp
	dst.Status.KubeSystemUUID = src.Status.KubeSystemUUID
}

func (src *DynaKube) toOneAgentStatus(dst *v1beta2.DynaKube) {
	dst.Status.OneAgent.Instances = map[string]v1beta2.OneAgentInstance{}

	// Instance
	for key, instance := range src.Status.OneAgent.Instances {
		tmp := v1beta2.OneAgentInstance{
			PodName:   instance.PodName,
			IPAddress: instance.IPAddress,
		}
		dst.Status.OneAgent.Instances[key] = tmp
	}

	dst.Status.OneAgent.LastInstanceStatusUpdate = src.Status.OneAgent.LastInstanceStatusUpdate

	// Connection-Info
	dst.Status.OneAgent.ConnectionInfoStatus.ConnectionInfoStatus = (v1beta2.ConnectionInfoStatus)(src.Status.OneAgent.ConnectionInfoStatus.ConnectionInfoStatus)

	for _, host := range src.Status.OneAgent.ConnectionInfoStatus.CommunicationHosts {
		tmp := v1beta2.CommunicationHostStatus{
			Host:     host.Host,
			Port:     host.Port,
			Protocol: host.Protocol,
		}
		dst.Status.OneAgent.ConnectionInfoStatus.CommunicationHosts = append(dst.Status.OneAgent.ConnectionInfoStatus.CommunicationHosts, tmp)
	}

	// Version
	dst.Status.OneAgent.VersionStatus = src.Status.OneAgent.VersionStatus
	dst.Status.OneAgent.Healthcheck = src.Status.OneAgent.Healthcheck
}

func (src *DynaKube) toActiveGateStatus(dst *v1beta2.DynaKube) {
	dst.Status.ActiveGate.ConnectionInfoStatus.ConnectionInfoStatus = (v1beta2.ConnectionInfoStatus)(src.Status.ActiveGate.ConnectionInfoStatus.ConnectionInfoStatus)
	dst.Status.ActiveGate.ServiceIPs = src.Status.ActiveGate.ServiceIPs
	dst.Status.ActiveGate.VersionStatus = src.Status.ActiveGate.VersionStatus
}

func toHostInjectSpec(src HostInjectSpec) *v1beta2.HostInjectSpec {
	dst := &v1beta2.HostInjectSpec{}
	dst.AutoUpdate = src.AutoUpdate

	dst.OneAgentResources = src.OneAgentResources
	dst.Args = src.Args
	dst.Version = src.Version
	dst.Annotations = src.Annotations
	dst.DNSPolicy = src.DNSPolicy
	dst.Env = src.Env
	dst.Image = src.Image
	dst.Labels = src.Labels
	dst.NodeSelector = src.NodeSelector
	dst.PriorityClassName = src.PriorityClassName
	dst.Tolerations = src.Tolerations
	dst.SecCompProfile = src.SecCompProfile

	return dst
}

func toAppInjectSpec(src AppInjectionSpec) *v1beta2.AppInjectionSpec {
	dst := &v1beta2.AppInjectionSpec{}

	dst.CodeModulesImage = src.CodeModulesImage
	dst.InitResources = src.InitResources
	dst.NamespaceSelector = src.NamespaceSelector

	return dst
}

func (src *DynaKube) toMetadataEnrichment(dst *v1beta2.DynaKube) {
	dst.Spec.MetadataEnrichment.Enabled = src.Spec.MetadataEnrichment.Enabled
	dst.Spec.MetadataEnrichment.NamespaceSelector = src.Spec.MetadataEnrichment.NamespaceSelector
}

func (src *DynaKube) toExtensions(dst *v1beta2.DynaKube) error {
	e := src.Spec.Extensions

	ex, err := json.Marshal(e)
	if err != nil {
		return err
	}

	dst.Annotations[api.AnnotationDynatraceExtensions] = string(ex)

	o := src.Spec.Templates.OpenTelemetryCollector

	otel, err := json.Marshal(o)
	if err != nil {
		return err
	}

	dst.Annotations[api.AnnotationDynatraceOpenTelemetryCollector] = string(otel)

	ee := src.Spec.Templates.ExtensionExecutionController

	eec, err := json.Marshal(ee)
	if err != nil {
		return err
	}

	dst.Annotations[api.AnnotationDynatraceextEnsionExecutionController] = string(eec)

	return nil
}
