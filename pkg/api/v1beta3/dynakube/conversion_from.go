package dynakube

import (
	"encoding/json"

	"github.com/Dynatrace/dynatrace-operator/pkg/api"
	v1beta2 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta2/dynakube"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

// ConvertFrom converts from the Hub version (v1beta2) to this version (v1beta3).
func (dst *DynaKube) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*v1beta2.DynaKube)
	if src.Annotations == nil {
		src.Annotations = map[string]string{}
	}

	dst.ObjectMeta = *src.ObjectMeta.DeepCopy() // DeepCopy mainly relevant for testing
	dst.Spec.APIURL = src.Spec.APIURL
	dst.Spec.Tokens = src.Spec.Tokens
	dst.Spec.CustomPullSecret = src.Spec.CustomPullSecret
	dst.Spec.SkipCertCheck = src.Spec.SkipCertCheck
	dst.Spec.Proxy = (*DynaKubeProxy)(src.Spec.Proxy)
	dst.Spec.TrustedCAs = src.Spec.TrustedCAs
	dst.Spec.NetworkZone = src.Spec.NetworkZone
	dst.Spec.EnableIstio = src.Spec.EnableIstio
	dst.fromOneAgentSpec(src)
	dst.fromActiveGateSpec(src)

	e := src.Annotations[api.AnnotationDynatraceExtensions]
	es := ExtensionsSpec{}
	json.Unmarshal([]byte(e), &es)
	dst.Spec.Extensions = es

	o := src.Annotations[api.AnnotationDynatraceOpenTelemetryCollector]
	otel := OpenTelemetryCollectorSpec{}
	json.Unmarshal([]byte(o), &otel)
	dst.Spec.OpenTelemetryCollector = otel

	ee := src.Annotations[api.AnnotationDynatraceextEnsionExecutionController]
	eec := ExtensionExecutionControllerSpec{}
	json.Unmarshal([]byte(ee), &eec)
	dst.Spec.ExtensionExecutionController = eec

	return nil
}

func (dst *DynaKube) fromOneAgentSpec(src *v1beta2.DynaKube) {
	dst.Spec.OneAgent.HostGroup = src.Spec.OneAgent.HostGroup

	switch {
	case src.HostMonitoringMode():
		dst.Spec.OneAgent.HostMonitoring = fromHostInjectSpec(*src.Spec.OneAgent.HostMonitoring)
	case src.ClassicFullStackMode():
		dst.Spec.OneAgent.ClassicFullStack = fromHostInjectSpec(*src.Spec.OneAgent.ClassicFullStack)
	case src.CloudNativeFullstackMode():
		dst.Spec.OneAgent.CloudNativeFullStack = &CloudNativeFullStackSpec{}
		dst.Spec.OneAgent.CloudNativeFullStack.HostInjectSpec = *fromHostInjectSpec(src.Spec.OneAgent.CloudNativeFullStack.HostInjectSpec)
		dst.Spec.OneAgent.CloudNativeFullStack.AppInjectionSpec = *fromAppInjectSpec(src.Spec.OneAgent.CloudNativeFullStack.AppInjectionSpec)
	case src.ApplicationMonitoringMode():
		dst.Spec.OneAgent.ApplicationMonitoring = &ApplicationMonitoringSpec{}
		dst.Spec.OneAgent.ApplicationMonitoring.AppInjectionSpec = *fromAppInjectSpec(src.Spec.OneAgent.ApplicationMonitoring.AppInjectionSpec)
		dst.Spec.OneAgent.ApplicationMonitoring.Version = src.Spec.OneAgent.ApplicationMonitoring.Version
		dst.Spec.OneAgent.ApplicationMonitoring.UseCSIDriver = src.Spec.OneAgent.ApplicationMonitoring.UseCSIDriver
	}
}

func (dst *DynaKube) fromActiveGateSpec(src *v1beta2.DynaKube) {
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

	for _, capability := range src.Spec.ActiveGate.Capabilities {
		dst.Spec.ActiveGate.Capabilities = append(dst.Spec.ActiveGate.Capabilities, CapabilityDisplayName(capability))
	}

	if src.Spec.ActiveGate.CustomProperties != nil {
		dst.Spec.ActiveGate.CustomProperties = &DynaKubeValueSource{
			Value:     src.Spec.ActiveGate.CustomProperties.Value,
			ValueFrom: src.Spec.ActiveGate.CustomProperties.ValueFrom,
		}
	}
}

func fromHostInjectSpec(src v1beta2.HostInjectSpec) *HostInjectSpec {
	dst := &HostInjectSpec{}
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

	return dst
}

func fromAppInjectSpec(src v1beta2.AppInjectionSpec) *AppInjectionSpec {
	dst := &AppInjectionSpec{}

	dst.CodeModulesImage = src.CodeModulesImage
	dst.InitResources = src.InitResources

	return dst
}