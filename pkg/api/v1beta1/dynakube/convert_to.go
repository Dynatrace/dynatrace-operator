package dynakube

import (
	"math"
	"strconv"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/metadataenrichment"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/communication"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/value"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

// ConvertTo converts v1beta1 to latest.
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
	configuredMountAttempts := src.FF().GetCSIMaxFailedMountAttempts()
	if configuredMountAttempts != exp.DefaultCSIMaxFailedMountAttempts {
		dst.Annotations[exp.CSIMaxMountTimeoutKey] = exp.MountAttemptsToTimeout(configuredMountAttempts)
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
		dst.Spec.OneAgent.CloudNativeFullStack = &oneagent.CloudNativeFullStackSpec{}
		dst.Spec.OneAgent.CloudNativeFullStack.HostInjectSpec = *toHostInjectSpec(src.Spec.OneAgent.CloudNativeFullStack.HostInjectSpec)
		dst.Spec.OneAgent.CloudNativeFullStack.AppInjectionSpec = *toAppInjectSpec(src.Spec.OneAgent.CloudNativeFullStack.AppInjectionSpec)
	case src.ApplicationMonitoringMode():
		dst.Spec.OneAgent.ApplicationMonitoring = &oneagent.ApplicationMonitoringSpec{}
		dst.Spec.OneAgent.ApplicationMonitoring.AppInjectionSpec = *toAppInjectSpec(src.Spec.OneAgent.ApplicationMonitoring.AppInjectionSpec)
		dst.Spec.OneAgent.ApplicationMonitoring.Version = src.Spec.OneAgent.ApplicationMonitoring.Version
	}
}

func (src *DynaKube) toActiveGateSpec(dst *dynakube.DynaKube) {
	dst.Spec.ActiveGate.Image = src.Spec.ActiveGate.Image
	dst.Spec.ActiveGate.PriorityClassName = src.Spec.ActiveGate.PriorityClassName
	dst.Spec.ActiveGate.TLSSecretName = src.Spec.ActiveGate.TLSSecretName
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

	dst.Spec.ActiveGate.Replicas = src.Spec.ActiveGate.Replicas
}

func (src *DynaKube) toMovedFields(dst *dynakube.DynaKube) error {
	if src.Annotations[exp.InjectionMetadataEnrichmentKey] == "false" ||
		!src.NeedAppInjection() {
		dst.Spec.MetadataEnrichment = metadataenrichment.Spec{Enabled: ptr.To(false)}
		delete(dst.Annotations, exp.InjectionMetadataEnrichmentKey)
	} else {
		dst.Spec.MetadataEnrichment = metadataenrichment.Spec{Enabled: ptr.To(true)}
		delete(dst.Annotations, exp.InjectionMetadataEnrichmentKey)
	}

	src.convertMaxMountAttempts(dst)

	src.convertDynatraceAPIRequestThreshold(dst)

	if src.Annotations[exp.OASecCompProfileKey] != "" {
		secCompProfile := src.Annotations[exp.OASecCompProfileKey]
		delete(dst.Annotations, exp.OASecCompProfileKey)

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

func (src *DynaKube) convertDynatraceAPIRequestThreshold(dst *dynakube.DynaKube) error {
	if src.Annotations[exp.APIRequestThresholdKey] != "" {
		duration, err := strconv.ParseInt(src.Annotations[exp.APIRequestThresholdKey], 10, 32)
		if err != nil {
			return err
		}

		if duration >= 0 {
			if math.MaxUint16 < duration {
				dst.Spec.DynatraceAPIRequestThreshold = ptr.To(uint16(math.MaxUint16))
			} else {
				// linting disabled, handled in if
				dst.Spec.DynatraceAPIRequestThreshold = ptr.To(uint16(duration)) //nolint:gosec
			}
		}

		delete(dst.Annotations, exp.APIRequestThresholdKey)
	}

	return nil
}

func (src *DynaKube) toStatus(dst *dynakube.DynaKube) {
	src.toOneAgentStatus(dst)
	src.toActiveGateStatus(dst)
	dst.Status.CodeModules = oneagent.CodeModulesStatus{
		VersionStatus: src.Status.CodeModules.VersionStatus,
	}

	dst.Status.DynatraceAPI = dynakube.DynatraceAPIStatus{
		LastTokenScopeRequest: src.Status.DynatraceAPI.LastTokenScopeRequest,
	}

	dst.Status.Conditions = src.Status.Conditions
	dst.Status.Phase = src.Status.Phase
	dst.Status.UpdatedTimestamp = src.Status.UpdatedTimestamp
	dst.Status.KubeSystemUUID = src.Status.KubeSystemUUID
}

func (src *DynaKube) toOneAgentStatus(dst *dynakube.DynaKube) {
	dst.Status.OneAgent.Instances = map[string]oneagent.Instance{}

	// Instance
	for key, instance := range src.Status.OneAgent.Instances {
		tmp := oneagent.Instance{
			PodName:   instance.PodName,
			IPAddress: instance.IPAddress,
		}
		dst.Status.OneAgent.Instances[key] = tmp
	}

	dst.Status.OneAgent.LastInstanceStatusUpdate = src.Status.OneAgent.LastInstanceStatusUpdate

	// Connection-Info
	dst.Status.OneAgent.ConnectionInfoStatus.ConnectionInfo = (communication.ConnectionInfo)(src.Status.OneAgent.ConnectionInfoStatus.ConnectionInfoStatus)

	for _, host := range src.Status.OneAgent.ConnectionInfoStatus.CommunicationHosts {
		tmp := oneagent.CommunicationHostStatus{
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

func toHostInjectSpec(src HostInjectSpec) *oneagent.HostInjectSpec {
	dst := &oneagent.HostInjectSpec{}
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

func toAppInjectSpec(src AppInjectionSpec) *oneagent.AppInjectionSpec {
	dst := &oneagent.AppInjectionSpec{}

	dst.CodeModulesImage = src.CodeModulesImage
	dst.InitResources = src.InitResources

	return dst
}
