package dynakube

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

// ConvertTo converts v1beta2 to v1beta1.
func (src *DynaKube) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*dynakube.DynaKube)

	dst.ObjectMeta = src.ObjectMeta

	// DynakubeSpec
	dst.Spec.APIURL = src.Spec.APIURL
	dst.Spec.Tokens = src.Spec.Tokens
	dst.Spec.CustomPullSecret = src.Spec.CustomPullSecret
	dst.Spec.SkipCertCheck = src.Spec.SkipCertCheck
	dst.Spec.Proxy = (*dynakube.DynaKubeProxy)(src.Spec.Proxy)
	dst.Spec.TrustedCAs = src.Spec.TrustedCAs
	dst.Spec.NetworkZone = src.Spec.NetworkZone
	dst.Spec.EnableIstio = src.Spec.EnableIstio

	// Status
	dst.Status.Conditions = src.Status.Conditions

	dst.Status.OneAgent.Instances = map[string]dynakube.OneAgentInstance{}

	for key, value := range src.Status.OneAgent.Instances {
		tmp := dynakube.OneAgentInstance{
			PodName:   value.PodName,
			IPAddress: value.IPAddress,
		}
		dst.Status.OneAgent.Instances[key] = tmp
	}

	dst.Status.OneAgent.Version = src.Status.OneAgent.Version

	dst.Status.Phase = src.Status.Phase
	dst.Status.UpdatedTimestamp = src.Status.UpdatedTimestamp

	return nil
}

// ConvertFrom converts v1beta1 to v1beta2.
func (dst *DynaKube) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*dynakube.DynaKube)
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

	if !src.FeatureDisableMetadataEnrichment() {
		dst.Spec.MetaDataEnrichment = MetaDataEnrichment{
			Enabled: true,
		}
	}

	dst.Spec.DynatraceApiRequestThreshold = src.FeatureApiRequestThreshold()

	if src.FeatureOneAgentSecCompProfile() != "" {
		dst.Spec.OneAgent.HostMonitoring = &HostInjectSpec{
			SecCompProfile: src.FeatureOneAgentSecCompProfile(),
		}
	}

	if src.NamespaceSelector() != nil {
		matchExpressions := src.NamespaceSelector().MatchExpressions
		matchLabels := src.NamespaceSelector().MatchLabels

		if src.CloudNativeFullstackMode() {
			dst.Spec.OneAgent.CloudNativeFullStack = &CloudNativeFullStackSpec{}
			dst.Spec.OneAgent.CloudNativeFullStack.NamespaceSelector = v1.LabelSelector{
				MatchExpressions: matchExpressions,
				MatchLabels:      matchLabels,
			}
		} else if src.ApplicationMonitoringMode() {
			dst.Spec.OneAgent.ApplicationMonitoring = &ApplicationMonitoringSpec{}
			dst.Spec.OneAgent.ApplicationMonitoring.NamespaceSelector = v1.LabelSelector{
				MatchExpressions: matchExpressions,
				MatchLabels:      matchLabels,
			}
		}
	}

	// Status

	dst.Status.OneAgent.Instances = map[string]OneAgentInstance{}

	for key, value := range src.Status.OneAgent.Instances {
		instance := OneAgentInstance{
			PodName:   value.PodName,
			IPAddress: value.IPAddress,
		}
		dst.Status.OneAgent.Instances[key] = instance
	}

	dst.Status.Conditions = src.Status.Conditions
	dst.Status.OneAgent.Version = src.Status.OneAgent.Version

	dst.Status.Phase = src.Status.Phase
	dst.Status.UpdatedTimestamp = src.Status.UpdatedTimestamp

	return nil
}
