package dynakube

import (
	"strconv"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/communication"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/value"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube/activegate"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

// ConvertTo converts v1beta1 to v1beta3.
func (src *DynaKube) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*dynakube.DynaKube)
	src.toBase(dst)
	src.toOneAgentSpec(dst)
	src.toActiveGateSpec(dst)
	src.toStatus(dst)

	err := src.toMovedFields(dst)
	if err != nil {
		return err
	}

	return nil
}

func (src *DynaKube) toBase(dst *dynakube.DynaKube) {
	if src.Annotations == nil {
		src.Annotations = map[string]string{}
	}

	dst.ObjectMeta = *src.ObjectMeta.DeepCopy() // DeepCopy mainly relevant for testing

	dst.Spec.APIURL = src.Spec.APIURL
	dst.Spec.Tokens = src.Spec.Tokens
	dst.Spec.CustomPullSecret = src.Spec.CustomPullSecret
	dst.Spec.SkipCertCheck = src.Spec.SkipCertCheck
	dst.Spec.Proxy = (*value.Source)(src.Spec.Proxy)
	dst.Spec.TrustedCAs = src.Spec.TrustedCAs
	dst.Spec.NetworkZone = src.Spec.NetworkZone
	dst.Spec.EnableIstio = src.Spec.EnableIstio
}

func (src *DynaKube) convertMaxMountAttempts(dst *dynakube.DynaKube) {
	configuredMountAttempts := src.FeatureMaxFailedCsiMountAttempts()
	if configuredMountAttempts != DefaultMaxFailedCsiMountAttempts {
		dst.Annotations[dynakube.AnnotationFeatureMaxCsiMountTimeout] = dynakube.MountAttemptsToTimeout(configuredMountAttempts)
	}
}

func (src *DynaKube) toOneAgentSpec(dst *dynakube.DynaKube) {
	dst.Spec.OneAgent.HostGroup = src.Spec.OneAgent.HostGroup

	switch {
	case src.HostMonitoringMode():
		dst.Spec.OneAgent.HostMonitoring = toHostInjectSpec(*src.Spec.OneAgent.HostMonitoring)
	case src.ClassicFullStackMode():
		dst.Spec.OneAgent.ClassicFullStack = toHostInjectSpec(*src.Spec.OneAgent.ClassicFullStack)
	case src.CloudNativeFullstackMode():
		dst.Spec.OneAgent.CloudNativeFullStack = &dynakube.CloudNativeFullStackSpec{}
		dst.Spec.OneAgent.CloudNativeFullStack.HostInjectSpec = *toHostInjectSpec(src.Spec.OneAgent.CloudNativeFullStack.HostInjectSpec)
		dst.Spec.OneAgent.CloudNativeFullStack.AppInjectionSpec = *toAppInjectSpec(src.Spec.OneAgent.CloudNativeFullStack.AppInjectionSpec)
	case src.ApplicationMonitoringMode():
		dst.Spec.OneAgent.ApplicationMonitoring = &dynakube.ApplicationMonitoringSpec{}
		dst.Spec.OneAgent.ApplicationMonitoring.AppInjectionSpec = *toAppInjectSpec(src.Spec.OneAgent.ApplicationMonitoring.AppInjectionSpec)
		dst.Spec.OneAgent.ApplicationMonitoring.Version = src.Spec.OneAgent.ApplicationMonitoring.Version

		if src.Spec.OneAgent.ApplicationMonitoring.UseCSIDriver != nil {
			dst.Spec.OneAgent.ApplicationMonitoring.UseCSIDriver = *src.Spec.OneAgent.ApplicationMonitoring.UseCSIDriver
		} else {
			dst.Spec.OneAgent.ApplicationMonitoring.UseCSIDriver = false
		}
	}
}

func (src *DynaKube) toActiveGateSpec(dst *dynakube.DynaKube) {
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
		dst.Spec.ActiveGate.Capabilities = append(dst.Spec.ActiveGate.Capabilities, activegate.CapabilityDisplayName(capability))
	}

	if src.Spec.ActiveGate.CustomProperties != nil {
		dst.Spec.ActiveGate.CustomProperties = &value.Source{
			Value:     src.Spec.ActiveGate.CustomProperties.Value,
			ValueFrom: src.Spec.ActiveGate.CustomProperties.ValueFrom,
		}
	}

	if src.Spec.ActiveGate.Replicas != nil {
		dst.Spec.ActiveGate.Replicas = *src.Spec.ActiveGate.Replicas
	}
}

func (src *DynaKube) toMovedFields(dst *dynakube.DynaKube) error {
	if src.Annotations[AnnotationFeatureMetadataEnrichment] == "false" ||
		!src.NeedAppInjection() {
		dst.Spec.MetadataEnrichment = dynakube.MetadataEnrichment{Enabled: false}
		delete(dst.Annotations, AnnotationFeatureMetadataEnrichment)
	} else {
		dst.Spec.MetadataEnrichment = dynakube.MetadataEnrichment{Enabled: true}
		delete(dst.Annotations, AnnotationFeatureMetadataEnrichment)
	}

	src.convertMaxMountAttempts(dst)

	if src.Annotations[AnnotationFeatureApiRequestThreshold] != "" {
		duration, err := strconv.ParseInt(src.Annotations[AnnotationFeatureApiRequestThreshold], 10, 32)
		if err != nil {
			return err
		}

		dst.Spec.DynatraceApiRequestThreshold = int(duration)
		delete(dst.Annotations, AnnotationFeatureApiRequestThreshold)
	} else {
		dst.Spec.DynatraceApiRequestThreshold = DefaultMinRequestThresholdMinutes
		delete(dst.Annotations, AnnotationFeatureApiRequestThreshold)
	}

	if src.Annotations[AnnotationFeatureOneAgentSecCompProfile] != "" {
		secCompProfile := src.Annotations[AnnotationFeatureOneAgentSecCompProfile]
		delete(dst.Annotations, AnnotationFeatureOneAgentSecCompProfile)

		switch {
		case src.CloudNativeFullstackMode():
			dst.Spec.OneAgent.CloudNativeFullStack.HostInjectSpec.SecCompProfile = secCompProfile
		case src.HostMonitoringMode():
			dst.Spec.OneAgent.HostMonitoring.SecCompProfile = secCompProfile
		case src.ClassicFullStackMode():
			dst.Spec.OneAgent.ClassicFullStack.SecCompProfile = secCompProfile
		}
	}

	if src.Spec.NamespaceSelector.Size() != 0 {
		if src.Spec.OneAgent.CloudNativeFullStack != nil {
			dst.Spec.OneAgent.CloudNativeFullStack.NamespaceSelector = src.Spec.NamespaceSelector
		} else if src.Spec.OneAgent.ApplicationMonitoring != nil {
			dst.Spec.OneAgent.ApplicationMonitoring.NamespaceSelector = src.Spec.NamespaceSelector
		}

		dst.Spec.MetadataEnrichment.NamespaceSelector = src.Spec.NamespaceSelector
	}

	return nil
}

func (src *DynaKube) toStatus(dst *dynakube.DynaKube) {
	src.toOneAgentStatus(dst)
	src.toActiveGateStatus(dst)
	dst.Status.CodeModules = dynakube.CodeModulesStatus{
		VersionStatus: src.Status.CodeModules.VersionStatus,
	}

	dst.Status.DynatraceApi = dynakube.DynatraceApiStatus{
		LastTokenScopeRequest: src.Status.DynatraceApi.LastTokenScopeRequest,
	}

	dst.Status.Conditions = src.Status.Conditions
	dst.Status.Phase = src.Status.Phase
	dst.Status.UpdatedTimestamp = src.Status.UpdatedTimestamp
	dst.Status.KubeSystemUUID = src.Status.KubeSystemUUID
}

func (src *DynaKube) toOneAgentStatus(dst *dynakube.DynaKube) {
	dst.Status.OneAgent.Instances = map[string]dynakube.OneAgentInstance{}

	// Instance
	for key, instance := range src.Status.OneAgent.Instances {
		tmp := dynakube.OneAgentInstance{
			PodName:   instance.PodName,
			IPAddress: instance.IPAddress,
		}
		dst.Status.OneAgent.Instances[key] = tmp
	}

	dst.Status.OneAgent.LastInstanceStatusUpdate = src.Status.OneAgent.LastInstanceStatusUpdate

	// Connection-Info
	dst.Status.OneAgent.ConnectionInfoStatus.ConnectionInfo = (communication.ConnectionInfo)(src.Status.OneAgent.ConnectionInfoStatus.ConnectionInfoStatus)

	for _, host := range src.Status.OneAgent.ConnectionInfoStatus.CommunicationHosts {
		tmp := dynakube.CommunicationHostStatus{
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

func (src *DynaKube) toActiveGateStatus(dst *dynakube.DynaKube) {
	dst.Status.ActiveGate.ConnectionInfo = (communication.ConnectionInfo)(src.Status.ActiveGate.ConnectionInfoStatus.ConnectionInfoStatus)
	dst.Status.ActiveGate.ServiceIPs = src.Status.ActiveGate.ServiceIPs
	dst.Status.ActiveGate.VersionStatus = src.Status.ActiveGate.VersionStatus
}

func toHostInjectSpec(src HostInjectSpec) *dynakube.HostInjectSpec {
	dst := &dynakube.HostInjectSpec{}
	if src.AutoUpdate != nil {
		dst.AutoUpdate = *src.AutoUpdate
	} else {
		dst.AutoUpdate = true
	}

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

func toAppInjectSpec(src AppInjectionSpec) *dynakube.AppInjectionSpec {
	dst := &dynakube.AppInjectionSpec{}

	dst.CodeModulesImage = src.CodeModulesImage
	dst.InitResources = src.InitResources

	return dst
}
