package dynakube

import (
	"strconv"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/installconfig"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

// ConvertFrom converts latest to v1beta1.
func (dst *DynaKube) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*dynakube.DynaKube)
	dst.fromBase(src)
	dst.fromOneAgentSpec(src)
	dst.fromActiveGateSpec(src)
	dst.fromStatus(src)

	err := dst.fromMovedFields(src)
	if err != nil {
		return err
	}

	return nil
}

func (dst *DynaKube) fromBase(src *dynakube.DynaKube) {
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
}

func (dst *DynaKube) fromOneAgentSpec(src *dynakube.DynaKube) {
	dst.Spec.OneAgent.HostGroup = src.Spec.OneAgent.HostGroup

	switch {
	case src.OneAgent().IsHostMonitoringMode():
		dst.Spec.OneAgent.HostMonitoring = fromHostInjectSpec(*src.Spec.OneAgent.HostMonitoring)
	case src.OneAgent().IsClassicFullStackMode():
		dst.Spec.OneAgent.ClassicFullStack = fromHostInjectSpec(*src.Spec.OneAgent.ClassicFullStack)
	case src.OneAgent().IsCloudNativeFullstackMode():
		dst.Spec.OneAgent.CloudNativeFullStack = &CloudNativeFullStackSpec{}
		dst.Spec.OneAgent.CloudNativeFullStack.HostInjectSpec = *fromHostInjectSpec(src.Spec.OneAgent.CloudNativeFullStack.HostInjectSpec)
		dst.Spec.OneAgent.CloudNativeFullStack.AppInjectionSpec = *fromAppInjectSpec(src.Spec.OneAgent.CloudNativeFullStack.AppInjectionSpec)
	case src.OneAgent().IsApplicationMonitoringMode():
		dst.Spec.OneAgent.ApplicationMonitoring = &ApplicationMonitoringSpec{}
		dst.Spec.OneAgent.ApplicationMonitoring.AppInjectionSpec = *fromAppInjectSpec(src.Spec.OneAgent.ApplicationMonitoring.AppInjectionSpec)
		dst.Spec.OneAgent.ApplicationMonitoring.Version = src.Spec.OneAgent.ApplicationMonitoring.Version
		dst.Spec.OneAgent.ApplicationMonitoring.UseCSIDriver = ptr.To(installconfig.GetModules().CSIDriver)
	}
}

func (dst *DynaKube) fromActiveGateSpec(src *dynakube.DynaKube) {
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
	dst.Spec.ActiveGate.Replicas = ptr.To(src.Spec.ActiveGate.GetReplicas())

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

func (dst *DynaKube) fromMovedFields(src *dynakube.DynaKube) error {
	dst.Annotations[exp.InjectionMetadataEnrichmentKey] = strconv.FormatBool(src.MetadataEnrichment().IsEnabled())
	dst.Annotations[exp.APIRequestThresholdKey] = strconv.FormatInt(int64(src.GetDynatraceAPIRequestThreshold()), 10)
	dst.Annotations[exp.OASecCompProfileKey] = src.OneAgent().GetSecCompProfile()

	if selector := src.OneAgent().GetNamespaceSelector(); selector != nil {
		dst.Spec.NamespaceSelector = *selector
	} else {
		dst.Spec.NamespaceSelector = src.Spec.MetadataEnrichment.NamespaceSelector
	}

	return nil
}

func (dst *DynaKube) fromStatus(src *dynakube.DynaKube) {
	dst.fromOneAgentStatus(*src)
	dst.fromActiveGateStatus(*src)
	dst.Status.CodeModules = CodeModulesStatus{
		VersionStatus: src.Status.CodeModules.VersionStatus,
	}

	dst.Status.DynatraceAPI = DynatraceAPIStatus{
		LastTokenScopeRequest: src.Status.DynatraceAPI.LastTokenScopeRequest,
	}

	dst.Status.LastTokenProbeTimestamp = &src.Status.DynatraceAPI.LastTokenScopeRequest
	dst.Status.Conditions = src.Status.Conditions
	dst.Status.Phase = src.Status.Phase
	dst.Status.UpdatedTimestamp = src.Status.UpdatedTimestamp
	dst.Status.KubeSystemUUID = src.Status.KubeSystemUUID
}

func (dst *DynaKube) fromOneAgentStatus(src dynakube.DynaKube) {
	dst.Status.OneAgent.Instances = map[string]OneAgentInstance{}

	// Instance
	dst.Status.OneAgent.LastInstanceStatusUpdate = src.Status.OneAgent.LastInstanceStatusUpdate

	for key, instance := range src.Status.OneAgent.Instances {
		tmp := OneAgentInstance{
			PodName:   instance.PodName,
			IPAddress: instance.IPAddress,
		}
		dst.Status.OneAgent.Instances[key] = tmp
	}

	// Connection-Info
	dst.Status.OneAgent.ConnectionInfoStatus.ConnectionInfoStatus = (ConnectionInfoStatus)(src.Status.OneAgent.ConnectionInfoStatus.ConnectionInfo)

	for _, host := range src.Status.OneAgent.ConnectionInfoStatus.CommunicationHosts {
		tmp := CommunicationHostStatus{
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

func (dst *DynaKube) fromActiveGateStatus(src dynakube.DynaKube) {
	dst.Status.ActiveGate.ConnectionInfoStatus.ConnectionInfoStatus = (ConnectionInfoStatus)(src.Status.ActiveGate.ConnectionInfo)
	dst.Status.ActiveGate.ServiceIPs = src.Status.ActiveGate.ServiceIPs
	dst.Status.ActiveGate.VersionStatus = src.Status.ActiveGate.VersionStatus
}

func fromHostInjectSpec(src oneagent.HostInjectSpec) *HostInjectSpec {
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

func fromAppInjectSpec(src oneagent.AppInjectionSpec) *AppInjectionSpec {
	dst := &AppInjectionSpec{}

	dst.CodeModulesImage = src.CodeModulesImage
	dst.InitResources = src.InitResources

	return dst
}
