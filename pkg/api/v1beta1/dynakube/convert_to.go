package dynakube

import (
	"fmt"
	"strconv"

	v1beta2 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta2/dynakube"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

// ConvertTo converts v1beta1 to v1beta2.
func (src *DynaKube) ConvertTo(dstRaw conversion.Hub) error {

	log.Info("wepudt autoupdate:ConvertTo: conversion v1beta1 to v1beta2 of DK " + src.Name + " started") // TODO: remove this logline
	dst := dstRaw.(*v1beta2.DynaKube)
	src.toBase(dst)
	src.toOneAgentSpec(dst)
	src.toActiveGateSpec(dst)
	src.toStatus(dst)

	err := src.toMovedFields(dst)
	if err != nil {
		log.Info("wepudt autoupdate:ConvertTo: conversion v1beta1 to v1beta2 of DK " + src.Name + " failed") // TODO: remove this logline

		return err
	}

	log.Info("wepudt autoupdate:ConvertTo: conversion v1beta1 to v1beta2 of DK " + src.Name + " succeeded")                                                                // TODO: remove this logline
	log.Info("wepudt autoupdate:ConvertTo: autoUpdate value of v1beta2 DK " + fmt.Sprintf("%v", dstRaw.(*v1beta2.DynaKube).Spec.OneAgent.CloudNativeFullStack.AutoUpdate)) // TODO: remove this logline

	return nil
}

func (src *DynaKube) toBase(dst *v1beta2.DynaKube) {
	dst.ObjectMeta = *src.ObjectMeta.DeepCopy() // DeepCopy mainly relevant for testing

	dst.Spec.APIURL = src.Spec.APIURL
	dst.Spec.Tokens = src.Spec.Tokens
	dst.Spec.CustomPullSecret = src.Spec.CustomPullSecret
	dst.Spec.SkipCertCheck = src.Spec.SkipCertCheck
	dst.Spec.Proxy = (*v1beta2.DynaKubeProxy)(src.Spec.Proxy)
	dst.Spec.TrustedCAs = src.Spec.TrustedCAs
	dst.Spec.NetworkZone = src.Spec.NetworkZone
	dst.Spec.EnableIstio = src.Spec.EnableIstio
}

func (src *DynaKube) toOneAgentSpec(dst *v1beta2.DynaKube) {
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

		if src.Spec.OneAgent.ApplicationMonitoring.UseCSIDriver != nil {
			dst.Spec.OneAgent.ApplicationMonitoring.UseCSIDriver = *src.Spec.OneAgent.ApplicationMonitoring.UseCSIDriver
		} else {
			dst.Spec.OneAgent.ApplicationMonitoring.UseCSIDriver = false
		}
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

func (src *DynaKube) toMovedFields(dst *v1beta2.DynaKube) error {
	if src.Annotations[AnnotationFeatureMetadataEnrichment] == "false" ||
		!src.NeedAppInjection() {
		dst.Spec.MetadataEnrichment = v1beta2.MetadataEnrichment{Enabled: false}
		delete(dst.Annotations, AnnotationFeatureMetadataEnrichment)
	} else {
		dst.Spec.MetadataEnrichment = v1beta2.MetadataEnrichment{Enabled: true}
		delete(dst.Annotations, AnnotationFeatureMetadataEnrichment)
	}

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
	log.Info("wepudt autoupdate:ConvertTo: HostInjectSpec conversion started")                                  // TODO: remove this logline
	log.Info("wepudt autoupdate:ConvertTo: HostInjectSpec.AutoUpdate is " + fmt.Sprintf("%v", *src.AutoUpdate)) // TODO: remove this logline
	dst := &v1beta2.HostInjectSpec{}
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
	log.Info("wepudt autoupdate: HostInjectSpec conversion succeeded") // TODO: remove this logline
	return dst
}

func toAppInjectSpec(src AppInjectionSpec) *v1beta2.AppInjectionSpec {
	dst := &v1beta2.AppInjectionSpec{}

	dst.CodeModulesImage = src.CodeModulesImage
	dst.InitResources = src.InitResources

	return dst
}
