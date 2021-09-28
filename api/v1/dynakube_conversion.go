package v1

import (
	"github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

// var log = logf.Log.WithName("dynakube-conversion")

func (src *DynaKube) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*v1alpha1.DynaKube)

	dst.ObjectMeta = src.ObjectMeta

	// DynakubeSpec
	dst.Spec.APIURL = src.Spec.APIURL
	dst.Spec.Tokens = src.Spec.Tokens
	dst.Spec.CustomPullSecret = src.Spec.CustomPullSecret
	dst.Spec.SkipCertCheck = src.Spec.SkipCertCheck
	dst.Spec.Proxy = (*v1alpha1.DynaKubeProxy)(src.Spec.Proxy)
	dst.Spec.TrustedCAs = src.Spec.TrustedCAs
	dst.Spec.NetworkZone = src.Spec.NetworkZone
	dst.Spec.EnableIstio = src.Spec.EnableIstio

	// ClassicFullStack
	if src.ClassicFullStackMode() {
		dst.Spec.OneAgent.Image = src.Spec.OneAgent.ClassicFullStack.Image
		dst.Spec.OneAgent.Version = src.Spec.OneAgent.ClassicFullStack.Version
		dst.Spec.OneAgent.AutoUpdate = src.Spec.OneAgent.ClassicFullStack.AutoUpdate

		dst.Spec.ClassicFullStack.Enabled = true
		dst.Spec.ClassicFullStack.NodeSelector = src.Spec.OneAgent.ClassicFullStack.NodeSelector
		dst.Spec.ClassicFullStack.PriorityClassName = src.Spec.OneAgent.ClassicFullStack.PriorityClassName
		dst.Spec.ClassicFullStack.Tolerations = src.Spec.OneAgent.ClassicFullStack.Tolerations
		dst.Spec.ClassicFullStack.Resources = src.Spec.OneAgent.ClassicFullStack.OneAgentResources
		dst.Spec.ClassicFullStack.Args = src.Spec.OneAgent.ClassicFullStack.Args
		dst.Spec.ClassicFullStack.DNSPolicy = src.Spec.OneAgent.ClassicFullStack.DNSPolicy
		dst.Spec.ClassicFullStack.Labels = src.Spec.OneAgent.ClassicFullStack.Labels
	}

	// Status
	dst.Status.ActiveGate.ImageHash = src.Status.ActiveGate.ImageHash
	dst.Status.ActiveGate.LastImageProbeTimestamp = src.Status.ActiveGate.LastUpdateProbeTimestamp
	dst.Status.ActiveGate.ImageVersion = src.Status.ActiveGate.Version

	dst.Status.Conditions = src.Status.Conditions

	dst.Status.LastAPITokenProbeTimestamp = src.Status.LastAPITokenProbeTimestamp
	dst.Status.LastClusterVersionProbeTimestamp = src.Status.LastClusterVersionProbeTimestamp
	dst.Status.LastPaaSTokenProbeTimestamp = src.Status.LastPaaSTokenProbeTimestamp

	dst.Status.OneAgent.UseImmutableImage = true
	dst.Status.OneAgent.ImageHash = src.Status.OneAgent.ImageHash
	dst.Status.OneAgent.Instances = map[string]v1alpha1.OneAgentInstance{}
	for key, value := range src.Status.OneAgent.Instances {
		tmp := v1alpha1.OneAgentInstance{
			Version:   src.Status.OneAgent.VersionStatus.Version,
			PodName:   value.PodName,
			IPAddress: value.IPAddress,
		}
		dst.Status.OneAgent.Instances[key] = tmp
	}
	dst.Status.OneAgent.LastUpdateProbeTimestamp = src.Status.OneAgent.LastUpdateProbeTimestamp
	dst.Status.OneAgent.Version = src.Status.OneAgent.Version
	dst.Status.OneAgent.ImageVersion = src.Status.OneAgent.Version

	dst.Status.Phase = v1alpha1.DynaKubePhaseType(src.Status.Phase)
	dst.Status.Tokens = src.Status.Tokens
	dst.Status.UpdatedTimestamp = src.Status.UpdatedTimestamp

	// ActiveGates
	// TODO

	return nil
}

func (dst *DynaKube) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*v1alpha1.DynaKube)
	dst.ObjectMeta = src.ObjectMeta

	// DynakubeSpec
	dst.Spec.APIURL = src.Spec.APIURL
	dst.Spec.Tokens = src.Spec.Tokens
	dst.Spec.CustomPullSecret = src.Spec.CustomPullSecret
	dst.Spec.SkipCertCheck = src.Spec.SkipCertCheck
	dst.Spec.Proxy = (*DynaKubeProxy)(src.Spec.Proxy)
	dst.Spec.TrustedCAs = src.Spec.TrustedCAs
	dst.Spec.NetworkZone = src.Spec.NetworkZone
	dst.Spec.EnableIstio = src.Spec.EnableIstio

	// ClassicFullStack
	if src.Spec.ClassicFullStack.Enabled {
		dst.Spec.OneAgent.ClassicFullStack.AutoUpdate = src.Spec.OneAgent.AutoUpdate
		dst.Spec.OneAgent.ClassicFullStack.Image = src.Spec.OneAgent.Image
		dst.Spec.OneAgent.ClassicFullStack.Version = src.Spec.OneAgent.Version

		dst.Spec.OneAgent.ClassicFullStack.NodeSelector = src.Spec.ClassicFullStack.NodeSelector
		dst.Spec.OneAgent.ClassicFullStack.PriorityClassName = src.Spec.ClassicFullStack.PriorityClassName
		dst.Spec.OneAgent.ClassicFullStack.Tolerations = src.Spec.ClassicFullStack.Tolerations
		dst.Spec.OneAgent.ClassicFullStack.OneAgentResources = src.Spec.ClassicFullStack.Resources
		dst.Spec.OneAgent.ClassicFullStack.Args = src.Spec.ClassicFullStack.Args
		dst.Spec.OneAgent.ClassicFullStack.DNSPolicy = src.Spec.ClassicFullStack.DNSPolicy
		dst.Spec.OneAgent.ClassicFullStack.Labels = src.Spec.ClassicFullStack.Labels
	}
	// Status
	dst.Status.ActiveGate.ImageHash = src.Status.ActiveGate.ImageHash
	dst.Status.ActiveGate.LastUpdateProbeTimestamp = src.Status.ActiveGate.LastImageProbeTimestamp
	dst.Status.ActiveGate.Version = src.Status.ActiveGate.ImageVersion

	dst.Status.Conditions = src.Status.Conditions

	dst.Status.LastAPITokenProbeTimestamp = src.Status.LastAPITokenProbeTimestamp
	dst.Status.LastClusterVersionProbeTimestamp = src.Status.LastClusterVersionProbeTimestamp
	dst.Status.LastPaaSTokenProbeTimestamp = src.Status.LastPaaSTokenProbeTimestamp

	dst.Status.OneAgent.ImageHash = src.Status.OneAgent.ImageHash
	dst.Status.OneAgent.Instances = map[string]OneAgentInstance{}
	for key, value := range src.Status.OneAgent.Instances {
		tmp := OneAgentInstance{
			PodName:   value.PodName,
			IPAddress: value.IPAddress,
		}
		dst.Status.OneAgent.Instances[key] = tmp
	}
	dst.Status.OneAgent.LastUpdateProbeTimestamp = src.Status.OneAgent.LastUpdateProbeTimestamp
	dst.Status.OneAgent.Version = src.Status.OneAgent.Version

	dst.Status.Phase = DynaKubePhaseType(src.Status.Phase)
	dst.Status.Tokens = src.Status.Tokens
	dst.Status.UpdatedTimestamp = src.Status.UpdatedTimestamp

	// ActiveGates
	// TODO

	return nil
}
