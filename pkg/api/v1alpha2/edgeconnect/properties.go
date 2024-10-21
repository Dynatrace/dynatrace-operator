package edgeconnect

import (
	"fmt"
	"strings"

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

func (ec *EdgeConnect) Conditions() *[]metav1.Condition { return &ec.Status.Conditions }

func (e *EdgeConnect) HostPatterns() []string {
	if !e.IsK8SAutomationEnabled() {
		return e.Spec.HostPatterns
	}

	var hostPatterns []string

	for _, hostPattern := range e.Spec.HostPatterns {
		if !strings.EqualFold(hostPattern, e.K8sAutomationHostPattern()) {
			hostPatterns = append(hostPatterns, hostPattern)
		}
	}

	hostPatterns = append(hostPatterns, e.K8sAutomationHostPattern())

	return hostPatterns
}

type HostMapping struct {
	From string `json:"from"`
	To   string `json:"to"`
}

func (e *EdgeConnect) HostMappings() []HostMapping {
	hostMappings := make([]HostMapping, 0)
	hostMappings = append(hostMappings, HostMapping{From: e.K8sAutomationHostPattern(), To: KubernetesDefaultDNS})

	return hostMappings
}

func (e *EdgeConnect) K8sAutomationHostPattern() string {
	return e.Name + "." + e.Namespace + "." + e.Status.KubeSystemUID + "." + kubernetesHostnameSuffix
}
