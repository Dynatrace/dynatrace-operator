package edgeconnect

import (
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/pkg/api"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// MaxNameLength is the maximum length of a EdgeConnect's name, we tend to add suffixes to the name to avoid name collisions for resources related to the EdgeConnect.
	// The limit is necessary because kubernetes uses the name of some resources for the label value, which has a limit of 63 characters. (see https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#syntax-and-character-set)
	MaxNameLength = 40

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

func (edgeConnect *EdgeConnect) EmptyPullSecret() corev1.Secret {
	return corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      edgeConnect.Spec.CustomPullSecret,
			Namespace: edgeConnect.Namespace,
		},
	}
}
