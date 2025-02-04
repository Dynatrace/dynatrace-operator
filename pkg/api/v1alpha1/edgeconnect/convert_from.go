package edgeconnect

import (
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha2/edgeconnect"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

// ConvertFrom converts v1alpha2 to v1alpha1.
func (dst *EdgeConnect) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*edgeconnect.EdgeConnect)
	dst.fromBase(src)
	dst.fromSpec(src)
	dst.fromStatus(src)

	return nil
}

func (dst *EdgeConnect) fromBase(src *edgeconnect.EdgeConnect) {
	dst.ObjectMeta = *src.ObjectMeta.DeepCopy()
}

func (dst *EdgeConnect) fromSpec(src *edgeconnect.EdgeConnect) {
	dst.Spec.Annotations = src.Spec.Annotations
	dst.Spec.Labels = src.Spec.Labels
	dst.Spec.Replicas = src.Spec.Replicas
	dst.Spec.NodeSelector = src.Spec.NodeSelector

	if src.Spec.KubernetesAutomation != nil {
		dst.Spec.KubernetesAutomation = &KubernetesAutomationSpec{
			Enabled: src.Spec.KubernetesAutomation.Enabled,
		}
	}

	if src.Spec.Proxy != nil {
		dst.Spec.Proxy = &ProxySpec{
			Host:    src.Spec.Proxy.Host,
			NoProxy: src.Spec.Proxy.NoProxy,
			AuthRef: src.Spec.Proxy.AuthRef,
			Port:    src.Spec.Proxy.Port,
		}
	}

	dst.Spec.ImageRef.Tag = src.Spec.ImageRef.Tag
	dst.Spec.ImageRef.Repository = src.Spec.ImageRef.Repository
	dst.Spec.ApiServer = src.Spec.ApiServer
	dst.Spec.HostRestrictions = strings.Join(src.Spec.HostRestrictions, ",")
	dst.Spec.CustomPullSecret = src.Spec.CustomPullSecret
	dst.Spec.CaCertsRef = src.Spec.CaCertsRef
	dst.Spec.ServiceAccountName = src.GetServiceAccountName()
	dst.Spec.OAuth.Resource = src.Spec.OAuth.Resource
	dst.Spec.OAuth.ClientSecret = src.Spec.OAuth.ClientSecret
	dst.Spec.OAuth.Endpoint = src.Spec.OAuth.Endpoint
	dst.Spec.OAuth.Provisioner = src.Spec.OAuth.Provisioner
	dst.Spec.Resources = src.Spec.Resources
	dst.Spec.Env = src.Spec.Env
	dst.Spec.Tolerations = src.Spec.Tolerations
	dst.Spec.TopologySpreadConstraints = src.Spec.TopologySpreadConstraints
	dst.Spec.HostPatterns = src.Spec.HostPatterns
	dst.Spec.AutoUpdate = src.IsAutoUpdateEnabled()
}

func (dst *EdgeConnect) fromStatus(src *edgeconnect.EdgeConnect) {
	dst.Status.Conditions = src.Status.Conditions
	dst.Status.KubeSystemUID = src.Status.KubeSystemUID
	dst.Status.DeploymentPhase = src.Status.DeploymentPhase
	dst.Status.UpdatedTimestamp = src.Status.UpdatedTimestamp
	dst.Status.Version = src.Status.Version
}
