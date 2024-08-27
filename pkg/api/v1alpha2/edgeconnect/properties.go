package edgeconnect

import (
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/pkg/api"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	defaultEdgeConnectRepository = "docker.io/dynatrace/edgeconnect"
)

func (edgeConnect *EdgeConnect) Image() string {
	repository := defaultEdgeConnectRepository
	tag := api.LatestTag

	if edgeConnect.Spec.ImageRef.Repository != "" {
		repository = edgeConnect.Spec.ImageRef.Repository
	}

	if edgeConnect.Spec.ImageRef.Tag != "" {
		tag = edgeConnect.Spec.ImageRef.Tag
	}

	return fmt.Sprintf("%s:%s", repository, tag)
}

func (edgeConnect *EdgeConnect) IsCustomImage() bool {
	return edgeConnect.Spec.ImageRef.Repository != ""
}

func (edgeConnect *EdgeConnect) IsProvisionerModeEnabled() bool {
	return edgeConnect.Spec.OAuth.Provisioner
}

func (edgeConnect *EdgeConnect) IsK8SAutomationEnabled() bool {
	return edgeConnect.Spec.KubernetesAutomation != nil && edgeConnect.Spec.KubernetesAutomation.Enabled
}

func (edgeConnect *EdgeConnect) EmptyPullSecret() corev1.Secret {
	return corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      edgeConnect.Spec.CustomPullSecret,
			Namespace: edgeConnect.Namespace,
		},
	}
}
