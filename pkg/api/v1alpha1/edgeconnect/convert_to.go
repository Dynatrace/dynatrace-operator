package edgeconnect

import (
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/proxy"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha2/edgeconnect"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

// ConvertTo converts v1alpha1 to v1alpha2.
func (src *EdgeConnect) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*edgeconnect.EdgeConnect)
	src.toBase(dst)
	src.toSpec(dst)
	src.toStatus(dst)

	return nil
}

func (src *EdgeConnect) toBase(dst *edgeconnect.EdgeConnect) {
	dst.ObjectMeta = *src.ObjectMeta.DeepCopy()
}

func (src *EdgeConnect) toSpec(dst *edgeconnect.EdgeConnect) {
	dst.Spec.Annotations = src.Spec.Annotations
	dst.Spec.Labels = src.Spec.Labels
	dst.Spec.Replicas = src.Spec.Replicas
	dst.Spec.NodeSelector = src.Spec.NodeSelector

	if src.Spec.KubernetesAutomation != nil {
		dst.Spec.KubernetesAutomation = &edgeconnect.KubernetesAutomationSpec{
			Enabled: src.Spec.KubernetesAutomation.Enabled,
		}
	}

	if src.Spec.Proxy != nil {
		dst.Spec.Proxy = &proxy.Spec{
			Host:    src.Spec.Proxy.Host,
			NoProxy: src.Spec.Proxy.NoProxy,
			AuthRef: src.Spec.Proxy.AuthRef,
			Port:    src.Spec.Proxy.Port,
		}
	}

	dst.Spec.ImageRef.Tag = src.Spec.ImageRef.Tag
	dst.Spec.ImageRef.Repository = src.Spec.ImageRef.Repository
	dst.Spec.ApiServer = src.Spec.ApiServer

	// Note: strings.Split returns [""] if we apply it to empty "" string
	if src.Spec.HostRestrictions != "" {
		dst.Spec.HostRestrictions = strings.Split(src.Spec.HostRestrictions, ",")
	}

	dst.Spec.CustomPullSecret = src.Spec.CustomPullSecret
	dst.Spec.CaCertsRef = src.Spec.CaCertsRef
	dst.Spec.ServiceAccountName = src.Spec.ServiceAccountName
	dst.Spec.OAuth.Resource = src.Spec.OAuth.Resource
	dst.Spec.OAuth.ClientSecret = src.Spec.OAuth.ClientSecret
	dst.Spec.OAuth.Endpoint = src.Spec.OAuth.Endpoint
	dst.Spec.OAuth.Provisioner = src.Spec.OAuth.Provisioner
	dst.Spec.Resources = src.Spec.Resources
	dst.Spec.Env = src.Spec.Env
	dst.Spec.Tolerations = src.Spec.Tolerations
	dst.Spec.TopologySpreadConstraints = src.Spec.TopologySpreadConstraints
	dst.Spec.HostPatterns = src.Spec.HostPatterns
	dst.Spec.AutoUpdate = src.Spec.AutoUpdate
}

func (src *EdgeConnect) toStatus(dst *edgeconnect.EdgeConnect) {
	dst.Status.Conditions = src.Status.Conditions
	dst.Status.KubeSystemUID = src.Status.KubeSystemUID
	dst.Status.DeploymentPhase = src.Status.DeploymentPhase
	dst.Status.UpdatedTimestamp = src.Status.UpdatedTimestamp
	dst.Status.Version = src.Status.Version
}
