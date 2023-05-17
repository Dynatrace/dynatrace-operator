package v1

import (
	"encoding/json"

	"github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

// ConvertTo converts v1 to v1beta1
func (src *DynaKube) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*v1beta1.DynaKube)

	dst.ObjectMeta = src.ObjectMeta

	dst.Spec.APIURL = src.Spec.APIURL
	dst.Spec.Tokens = src.Spec.Tokens
	dst.Spec.CustomPullSecret = src.Spec.CustomPullSecret
	dst.Spec.SkipCertCheck = src.Spec.SkipCertCheck
	if src.Spec.Proxy != nil {
		dst.Spec.Proxy.Value = src.Spec.Proxy.Value
		dst.Spec.Proxy.ValueFrom = src.Spec.Proxy.ValueFrom
	}
	dst.Spec.TrustedCAs = src.Spec.TrustedCAs
	dst.Spec.NetworkZone = src.Spec.NetworkZone
	dst.Spec.EnableIstio = src.Spec.EnableIstio
	dst.Spec.NamespaceSelector = src.Spec.NamespaceSelector

	dst.Spec.OneAgent.ClassicFullStack = convertToClassicFullStack(src)
	dst.Spec.OneAgent.ApplicationMonitoring = convertToApplicationMonitoring(src)
	dst.Spec.OneAgent.HostMonitoring = convertToHostMonitoring(src)
	dst.Spec.OneAgent.CloudNativeFullStack = convertToCloudNativeFullStack(src)

	dst.Spec.ActiveGate = convertToActiveGate(src)
	convertToDroppedActiveGateCapabilities(dst, src)

	// Status
	dst.Status.Phase = v1beta1.DynaKubePhaseType(src.Status.Phase)
	dst.Status.UpdatedTimestamp = src.Status.UpdatedTimestamp
	dst.Status.LastTokenProbeTimestamp = src.Status.LastTokenProbeTimestamp
	dst.Status.KubeSystemUUID = src.Status.KubeSystemUUID
	dst.Status.Conditions = src.Status.Conditions

	dst.Status.ActiveGate = v1beta1.ActiveGateStatus{
		VersionStatus: v1beta1.VersionStatus{
			Source:             v1beta1.VersionSource(src.Status.ActiveGate.Source),
			ImageID:            src.Status.ActiveGate.ImageID,
			Version:            src.Status.ActiveGate.Version,
			LastProbeTimestamp: src.Status.ActiveGate.LastProbeTimestamp,
		},
		ConnectionInfoStatus: v1beta1.ActiveGateConnectionInfoStatus{
			ConnectionInfoStatus: v1beta1.ConnectionInfoStatus{
				TenantUUID:  src.Status.ActiveGate.ConnectionInfoStatus.TenantUUID,
				Endpoints:   src.Status.ActiveGate.ConnectionInfoStatus.Endpoints,
				LastRequest: src.Status.ActiveGate.ConnectionInfoStatus.LastRequest,
			},
		},
	}

	dst.Status.OneAgent = v1beta1.OneAgentStatus{
		VersionStatus: v1beta1.VersionStatus{
			Source:             v1beta1.VersionSource(src.Status.OneAgent.Source),
			ImageID:            src.Status.OneAgent.ImageID,
			Version:            src.Status.OneAgent.Version,
			LastProbeTimestamp: src.Status.OneAgent.LastProbeTimestamp,
		},
		Instances:                nil,
		LastInstanceStatusUpdate: src.Status.OneAgent.LastInstanceStatusUpdate,
		ConnectionInfoStatus: v1beta1.OneAgentConnectionInfoStatus{
			ConnectionInfoStatus: v1beta1.ConnectionInfoStatus{
				TenantUUID:  src.Status.ActiveGate.ConnectionInfoStatus.TenantUUID,
				Endpoints:   src.Status.ActiveGate.ConnectionInfoStatus.Endpoints,
				LastRequest: src.Status.ActiveGate.ConnectionInfoStatus.LastRequest,
			},
			CommunicationHosts: nil,
		},
	}

	dst.Status.OneAgent.Instances = make(map[string]v1beta1.OneAgentInstance)
	for key, instance := range src.Status.OneAgent.Instances {
		dst.Status.OneAgent.Instances[key] = v1beta1.OneAgentInstance{
			PodName:   instance.PodName,
			IPAddress: instance.IPAddress,
		}
	}
	dst.Status.OneAgent.ConnectionInfoStatus.CommunicationHosts = make([]v1beta1.CommunicationHostStatus, 0)
	for _, host := range src.Status.OneAgent.ConnectionInfoStatus.CommunicationHosts {
		dst.Status.OneAgent.ConnectionInfoStatus.CommunicationHosts = append(dst.Status.OneAgent.ConnectionInfoStatus.CommunicationHosts, v1beta1.CommunicationHostStatus{
			Protocol: host.Protocol,
			Host:     host.Host,
			Port:     host.Port,
		})
	}

	dst.Status.CodeModules = v1beta1.CodeModulesStatus{}
	dst.Status.Synthetic = v1beta1.SyntheticStatus{}
	dst.Status.DynatraceApi = v1beta1.DynatraceApiStatus{}
	return nil
}

func convertToClassicFullStack(src *DynaKube) *v1beta1.HostInjectSpec {
	if src.Spec.OneAgent.ClassicFullStack != nil {
		return &v1beta1.HostInjectSpec{
			NodeSelector:      src.Spec.OneAgent.ClassicFullStack.NodeSelector,
			PriorityClassName: src.Spec.OneAgent.ClassicFullStack.PriorityClassName,
			Tolerations:       src.Spec.OneAgent.ClassicFullStack.Tolerations,
			OneAgentResources: src.Spec.OneAgent.ClassicFullStack.OneAgentResources,
			Args:              src.Spec.OneAgent.ClassicFullStack.Args,
			Env:               src.Spec.OneAgent.ClassicFullStack.Env,
			AutoUpdate:        src.Spec.OneAgent.ClassicFullStack.AutoUpdate,
			DNSPolicy:         src.Spec.OneAgent.ClassicFullStack.DNSPolicy,
			Annotations:       src.Spec.OneAgent.ClassicFullStack.Annotations,
			Labels:            src.Spec.OneAgent.ClassicFullStack.Labels,
			Image:             src.Spec.OneAgent.ClassicFullStack.Image,
			Version:           src.Spec.OneAgent.ClassicFullStack.Version,
		}
	}
	return nil
}

func convertToApplicationMonitoring(src *DynaKube) *v1beta1.ApplicationMonitoringSpec {
	if src.Spec.OneAgent.ApplicationMonitoring != nil {
		return &v1beta1.ApplicationMonitoringSpec{
			AppInjectionSpec: v1beta1.AppInjectionSpec{
				InitResources:    src.Spec.OneAgent.ApplicationMonitoring.InitResources,
				CodeModulesImage: src.Spec.OneAgent.ApplicationMonitoring.CodeModulesImage,
			},
			Version:      src.Spec.OneAgent.ApplicationMonitoring.Version,
			UseCSIDriver: src.Spec.OneAgent.ApplicationMonitoring.UseCSIDriver,
		}
	}
	return nil
}

func convertToHostMonitoring(src *DynaKube) *v1beta1.HostInjectSpec {
	if src.Spec.OneAgent.HostMonitoring != nil {
		return &v1beta1.HostInjectSpec{
			NodeSelector:      src.Spec.OneAgent.HostMonitoring.NodeSelector,
			PriorityClassName: src.Spec.OneAgent.HostMonitoring.PriorityClassName,
			Tolerations:       src.Spec.OneAgent.HostMonitoring.Tolerations,
			OneAgentResources: src.Spec.OneAgent.HostMonitoring.OneAgentResources,
			Args:              src.Spec.OneAgent.HostMonitoring.Args,
			Env:               src.Spec.OneAgent.HostMonitoring.Env,
			AutoUpdate:        src.Spec.OneAgent.HostMonitoring.AutoUpdate,
			DNSPolicy:         src.Spec.OneAgent.HostMonitoring.DNSPolicy,
			Annotations:       src.Spec.OneAgent.HostMonitoring.Annotations,
			Labels:            src.Spec.OneAgent.HostMonitoring.Labels,
			Image:             src.Spec.OneAgent.HostMonitoring.Image,
			Version:           src.Spec.OneAgent.HostMonitoring.Version,
		}
	}
	return nil
}

func convertToCloudNativeFullStack(src *DynaKube) *v1beta1.CloudNativeFullStackSpec {
	if src.Spec.OneAgent.CloudNativeFullStack != nil {
		return &v1beta1.CloudNativeFullStackSpec{
			HostInjectSpec: v1beta1.HostInjectSpec{
				NodeSelector:      src.Spec.OneAgent.ClassicFullStack.NodeSelector,
				PriorityClassName: src.Spec.OneAgent.ClassicFullStack.PriorityClassName,
				Tolerations:       src.Spec.OneAgent.ClassicFullStack.Tolerations,
				OneAgentResources: src.Spec.OneAgent.ClassicFullStack.OneAgentResources,
				Args:              src.Spec.OneAgent.ClassicFullStack.Args,
				Env:               src.Spec.OneAgent.ClassicFullStack.Env,
				AutoUpdate:        src.Spec.OneAgent.ClassicFullStack.AutoUpdate,
				DNSPolicy:         src.Spec.OneAgent.ClassicFullStack.DNSPolicy,
				Annotations:       src.Spec.OneAgent.ClassicFullStack.Annotations,
				Labels:            src.Spec.OneAgent.ClassicFullStack.Labels,
				Image:             src.Spec.OneAgent.ClassicFullStack.Image,
				Version:           src.Spec.OneAgent.ClassicFullStack.Version,
			},
			AppInjectionSpec: v1beta1.AppInjectionSpec{
				InitResources:    src.Spec.OneAgent.ApplicationMonitoring.InitResources,
				CodeModulesImage: src.Spec.OneAgent.ApplicationMonitoring.CodeModulesImage,
			},
		}
	}
	return nil
}

func convertToActiveGate(src *DynaKube) v1beta1.ActiveGateSpec {
	capabilities := make([]v1beta1.CapabilityDisplayName, 0)
	for _, capability := range src.Spec.ActiveGate.Capabilities {
		capabilities = append(capabilities, v1beta1.CapabilityDisplayName(capability))
	}

	var customProperties *v1beta1.DynaKubeValueSource
	if src.Spec.ActiveGate.CapabilityProperties.CustomProperties != nil {
		customProperties = &v1beta1.DynaKubeValueSource{
			Value:     src.Spec.ActiveGate.CapabilityProperties.CustomProperties.Value,
			ValueFrom: src.Spec.ActiveGate.CapabilityProperties.CustomProperties.ValueFrom,
		}
	}

	return v1beta1.ActiveGateSpec{
		Capabilities: capabilities,
		CapabilityProperties: v1beta1.CapabilityProperties{
			Replicas:                  src.Spec.ActiveGate.CapabilityProperties.Replicas,
			Image:                     src.Spec.ActiveGate.CapabilityProperties.Image,
			Group:                     src.Spec.ActiveGate.CapabilityProperties.Group,
			CustomProperties:          customProperties,
			Resources:                 src.Spec.ActiveGate.CapabilityProperties.Resources,
			NodeSelector:              src.Spec.ActiveGate.CapabilityProperties.NodeSelector,
			Tolerations:               src.Spec.ActiveGate.CapabilityProperties.Tolerations,
			Labels:                    src.Spec.ActiveGate.CapabilityProperties.Labels,
			Env:                       src.Spec.ActiveGate.CapabilityProperties.Env,
			TopologySpreadConstraints: src.Spec.ActiveGate.CapabilityProperties.TopologySpreadConstraints,
		},
		TlsSecretName:     src.Spec.ActiveGate.TlsSecretName,
		DNSPolicy:         src.Spec.ActiveGate.DNSPolicy,
		PriorityClassName: src.Spec.ActiveGate.PriorityClassName,
		Annotations:       src.Spec.ActiveGate.Annotations,
	}
}

func convertToDroppedActiveGateCapabilities(dst *v1beta1.DynaKube, src *DynaKube) error {
	if dst.ObjectMeta.Annotations == nil {
		dst.ObjectMeta.Annotations = map[string]string{}
	}

	if value, ok := src.ObjectMeta.Annotations["routing"]; ok {
		if err := json.Unmarshal([]byte(value), &dst.Spec.Routing); err != nil {
			return err
		}
	} else {
		dst.Spec.Routing.Enabled = false
	}
	if value, ok := src.ObjectMeta.Annotations["kubernetes"]; ok {
		if err := json.Unmarshal([]byte(value), &dst.Spec.KubernetesMonitoring); err != nil {
			return err
		}
	} else {
		dst.Spec.KubernetesMonitoring.Enabled = false
	}
	return nil
}

// ConvertFrom converts v1beta1 to v1
func (dst *DynaKube) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*v1beta1.DynaKube)
	dst.ObjectMeta = src.ObjectMeta

	dst.Spec.APIURL = src.Spec.APIURL
	dst.Spec.Tokens = src.Spec.Tokens
	dst.Spec.CustomPullSecret = src.Spec.CustomPullSecret
	dst.Spec.SkipCertCheck = src.Spec.SkipCertCheck
	if src.Spec.Proxy != nil {
		dst.Spec.Proxy.Value = src.Spec.Proxy.Value
		dst.Spec.Proxy.ValueFrom = src.Spec.Proxy.ValueFrom
	}
	dst.Spec.TrustedCAs = src.Spec.TrustedCAs
	dst.Spec.NetworkZone = src.Spec.NetworkZone
	dst.Spec.EnableIstio = src.Spec.EnableIstio
	dst.Spec.NamespaceSelector = src.Spec.NamespaceSelector

	dst.Spec.OneAgent.ClassicFullStack = convertFromClassicFullStack(src)
	dst.Spec.OneAgent.ApplicationMonitoring = convertFromApplicationMonitoring(src)
	dst.Spec.OneAgent.HostMonitoring = convertFromHostMonitoring(src)
	dst.Spec.OneAgent.CloudNativeFullStack = convertFromCloudNativeFullStack(src)

	dst.Spec.ActiveGate = convertFromActiveGate(src)
	convertFromDroppedActiveGateCapabilities(dst, src)

	// Status
	dst.Status.Phase = DynaKubePhaseType(src.Status.Phase)
	dst.Status.UpdatedTimestamp = src.Status.UpdatedTimestamp
	dst.Status.LastTokenProbeTimestamp = src.Status.LastTokenProbeTimestamp
	dst.Status.KubeSystemUUID = src.Status.KubeSystemUUID
	dst.Status.Conditions = src.Status.Conditions

	dst.Status.ActiveGate = ActiveGateStatus{
		VersionStatus: VersionStatus{
			Source:             VersionSource(src.Status.ActiveGate.Source),
			ImageID:            src.Status.ActiveGate.ImageID,
			Version:            src.Status.ActiveGate.Version,
			LastProbeTimestamp: src.Status.ActiveGate.LastProbeTimestamp,
		},
		ConnectionInfoStatus: ActiveGateConnectionInfoStatus{
			ConnectionInfoStatus: ConnectionInfoStatus{
				TenantUUID:  src.Status.ActiveGate.ConnectionInfoStatus.TenantUUID,
				Endpoints:   src.Status.ActiveGate.ConnectionInfoStatus.Endpoints,
				LastRequest: src.Status.ActiveGate.ConnectionInfoStatus.LastRequest,
			},
		},
	}

	dst.Status.OneAgent = OneAgentStatus{
		VersionStatus: VersionStatus{
			Source:             VersionSource(src.Status.OneAgent.Source),
			ImageID:            src.Status.OneAgent.ImageID,
			Version:            src.Status.OneAgent.Version,
			LastProbeTimestamp: src.Status.OneAgent.LastProbeTimestamp,
		},
		Instances:                nil,
		LastInstanceStatusUpdate: src.Status.OneAgent.LastInstanceStatusUpdate,
		ConnectionInfoStatus: OneAgentConnectionInfoStatus{
			ConnectionInfoStatus: ConnectionInfoStatus{
				TenantUUID:  src.Status.ActiveGate.ConnectionInfoStatus.TenantUUID,
				Endpoints:   src.Status.ActiveGate.ConnectionInfoStatus.Endpoints,
				LastRequest: src.Status.ActiveGate.ConnectionInfoStatus.LastRequest,
			},
			CommunicationHosts: nil,
		},
	}

	dst.Status.OneAgent.Instances = make(map[string]OneAgentInstance)
	for key, instance := range src.Status.OneAgent.Instances {
		dst.Status.OneAgent.Instances[key] = OneAgentInstance{
			PodName:   instance.PodName,
			IPAddress: instance.IPAddress,
		}
	}
	dst.Status.OneAgent.ConnectionInfoStatus.CommunicationHosts = make([]CommunicationHostStatus, 0)
	for _, host := range src.Status.OneAgent.ConnectionInfoStatus.CommunicationHosts {
		dst.Status.OneAgent.ConnectionInfoStatus.CommunicationHosts = append(dst.Status.OneAgent.ConnectionInfoStatus.CommunicationHosts, CommunicationHostStatus{
			Protocol: host.Protocol,
			Host:     host.Host,
			Port:     host.Port,
		})
	}

	dst.Status.CodeModules = CodeModulesStatus{}
	dst.Status.Synthetic = SyntheticStatus{}
	dst.Status.DynatraceApi = DynatraceApiStatus{}

	return nil
}

func convertFromClassicFullStack(src *v1beta1.DynaKube) *HostInjectSpec {
	if src.Spec.OneAgent.ClassicFullStack != nil {
		return &HostInjectSpec{
			NodeSelector:      src.Spec.OneAgent.ClassicFullStack.NodeSelector,
			PriorityClassName: src.Spec.OneAgent.ClassicFullStack.PriorityClassName,
			Tolerations:       src.Spec.OneAgent.ClassicFullStack.Tolerations,
			OneAgentResources: src.Spec.OneAgent.ClassicFullStack.OneAgentResources,
			Args:              src.Spec.OneAgent.ClassicFullStack.Args,
			Env:               src.Spec.OneAgent.ClassicFullStack.Env,
			AutoUpdate:        src.Spec.OneAgent.ClassicFullStack.AutoUpdate,
			DNSPolicy:         src.Spec.OneAgent.ClassicFullStack.DNSPolicy,
			Annotations:       src.Spec.OneAgent.ClassicFullStack.Annotations,
			Labels:            src.Spec.OneAgent.ClassicFullStack.Labels,
			Image:             src.Spec.OneAgent.ClassicFullStack.Image,
			Version:           src.Spec.OneAgent.ClassicFullStack.Version,
		}
	}
	return nil
}

func convertFromApplicationMonitoring(src *v1beta1.DynaKube) *ApplicationMonitoringSpec {
	if src.Spec.OneAgent.ApplicationMonitoring != nil {
		return &ApplicationMonitoringSpec{
			AppInjectionSpec: AppInjectionSpec{
				InitResources:    src.Spec.OneAgent.ApplicationMonitoring.InitResources,
				CodeModulesImage: src.Spec.OneAgent.ApplicationMonitoring.CodeModulesImage,
			},
			Version:      src.Spec.OneAgent.ApplicationMonitoring.Version,
			UseCSIDriver: src.Spec.OneAgent.ApplicationMonitoring.UseCSIDriver,
		}
	}
	return nil
}

func convertFromHostMonitoring(src *v1beta1.DynaKube) *HostInjectSpec {
	if src.Spec.OneAgent.HostMonitoring != nil {
		return &HostInjectSpec{
			NodeSelector:      src.Spec.OneAgent.HostMonitoring.NodeSelector,
			PriorityClassName: src.Spec.OneAgent.HostMonitoring.PriorityClassName,
			Tolerations:       src.Spec.OneAgent.HostMonitoring.Tolerations,
			OneAgentResources: src.Spec.OneAgent.HostMonitoring.OneAgentResources,
			Args:              src.Spec.OneAgent.HostMonitoring.Args,
			Env:               src.Spec.OneAgent.HostMonitoring.Env,
			AutoUpdate:        src.Spec.OneAgent.HostMonitoring.AutoUpdate,
			DNSPolicy:         src.Spec.OneAgent.HostMonitoring.DNSPolicy,
			Annotations:       src.Spec.OneAgent.HostMonitoring.Annotations,
			Labels:            src.Spec.OneAgent.HostMonitoring.Labels,
			Image:             src.Spec.OneAgent.HostMonitoring.Image,
			Version:           src.Spec.OneAgent.HostMonitoring.Version,
		}
	}
	return nil
}

func convertFromCloudNativeFullStack(src *v1beta1.DynaKube) *CloudNativeFullStackSpec {
	if src.Spec.OneAgent.CloudNativeFullStack != nil {
		return &CloudNativeFullStackSpec{
			HostInjectSpec: HostInjectSpec{
				NodeSelector:      src.Spec.OneAgent.ClassicFullStack.NodeSelector,
				PriorityClassName: src.Spec.OneAgent.ClassicFullStack.PriorityClassName,
				Tolerations:       src.Spec.OneAgent.ClassicFullStack.Tolerations,
				OneAgentResources: src.Spec.OneAgent.ClassicFullStack.OneAgentResources,
				Args:              src.Spec.OneAgent.ClassicFullStack.Args,
				Env:               src.Spec.OneAgent.ClassicFullStack.Env,
				AutoUpdate:        src.Spec.OneAgent.ClassicFullStack.AutoUpdate,
				DNSPolicy:         src.Spec.OneAgent.ClassicFullStack.DNSPolicy,
				Annotations:       src.Spec.OneAgent.ClassicFullStack.Annotations,
				Labels:            src.Spec.OneAgent.ClassicFullStack.Labels,
				Image:             src.Spec.OneAgent.ClassicFullStack.Image,
				Version:           src.Spec.OneAgent.ClassicFullStack.Version,
			},
			AppInjectionSpec: AppInjectionSpec{
				InitResources:    src.Spec.OneAgent.ApplicationMonitoring.InitResources,
				CodeModulesImage: src.Spec.OneAgent.ApplicationMonitoring.CodeModulesImage,
			},
		}
	}
	return nil
}

func convertFromActiveGate(src *v1beta1.DynaKube) ActiveGateSpec {
	capabilities := make([]CapabilityDisplayName, 0)
	for _, capability := range src.Spec.ActiveGate.Capabilities {
		capabilities = append(capabilities, CapabilityDisplayName(capability))
	}

	var customProperties *DynaKubeValueSource
	if src.Spec.ActiveGate.CapabilityProperties.CustomProperties != nil {
		customProperties = &DynaKubeValueSource{
			Value:     src.Spec.ActiveGate.CapabilityProperties.CustomProperties.Value,
			ValueFrom: src.Spec.ActiveGate.CapabilityProperties.CustomProperties.ValueFrom,
		}
	}

	return ActiveGateSpec{
		Capabilities: capabilities,
		CapabilityProperties: CapabilityProperties{
			Replicas:                  src.Spec.ActiveGate.CapabilityProperties.Replicas,
			Image:                     src.Spec.ActiveGate.CapabilityProperties.Image,
			Group:                     src.Spec.ActiveGate.CapabilityProperties.Group,
			CustomProperties:          customProperties,
			Resources:                 src.Spec.ActiveGate.CapabilityProperties.Resources,
			NodeSelector:              src.Spec.ActiveGate.CapabilityProperties.NodeSelector,
			Tolerations:               src.Spec.ActiveGate.CapabilityProperties.Tolerations,
			Labels:                    src.Spec.ActiveGate.CapabilityProperties.Labels,
			Env:                       src.Spec.ActiveGate.CapabilityProperties.Env,
			TopologySpreadConstraints: src.Spec.ActiveGate.CapabilityProperties.TopologySpreadConstraints,
		},
		TlsSecretName:     src.Spec.ActiveGate.TlsSecretName,
		DNSPolicy:         src.Spec.ActiveGate.DNSPolicy,
		PriorityClassName: src.Spec.ActiveGate.PriorityClassName,
		Annotations:       src.Spec.ActiveGate.Annotations,
	}
}

func convertFromDroppedActiveGateCapabilities(dst *DynaKube, src *v1beta1.DynaKube) error {
	if dst.ObjectMeta.Annotations == nil {
		dst.ObjectMeta.Annotations = map[string]string{}
	}

	if src.Spec.Routing.Enabled {
		routing, err := json.Marshal(&src.Spec.Routing)
		if err != nil {
			return err
		}
		dst.ObjectMeta.Annotations["routing"] = string(routing)
	}

	if src.Spec.KubernetesMonitoring.Enabled {
		kubernetes, err := json.Marshal(&src.Spec.KubernetesMonitoring)
		if err != nil {
			return err
		}
		dst.ObjectMeta.Annotations["kubernetes"] = string(kubernetes)
	}
	return nil
}
