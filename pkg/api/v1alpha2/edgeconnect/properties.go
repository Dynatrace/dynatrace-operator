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

func (ec *EdgeConnect) Image() string {
	repository := defaultEdgeConnectRepository
	tag := api.LatestTag

	if ec.Spec.ImageRef.Repository != "" {
		repository = ec.Spec.ImageRef.Repository
	}

	if ec.Spec.ImageRef.Tag != "" {
		tag = ec.Spec.ImageRef.Tag
	}

	return fmt.Sprintf("%s:%s", repository, tag)
}

func (ec *EdgeConnect) IsCustomImage() bool {
	return ec.Spec.ImageRef.Repository != ""
}

func (ec *EdgeConnect) IsAutoUpdateEnabled() bool {
	return ec.Spec.AutoUpdate == nil || *ec.Spec.AutoUpdate
}

func (ec *EdgeConnect) GetServiceAccountName() string {
	defaultServiceAccount := "dynatrace-edgeconnect"
	if ec.Spec.ServiceAccountName == nil {
		return defaultServiceAccount
	}

	return *ec.Spec.ServiceAccountName
}

func (ec *EdgeConnect) IsProvisionerModeEnabled() bool {
	return ec.Spec.OAuth.Provisioner
}

func (ec *EdgeConnect) IsK8SAutomationEnabled() bool {
	return ec.Spec.KubernetesAutomation != nil && ec.Spec.KubernetesAutomation.Enabled
}

func (ec *EdgeConnect) EmptyPullSecret() corev1.Secret {
	return corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ec.Spec.CustomPullSecret,
			Namespace: ec.Namespace,
		},
	}
}

func (ec *EdgeConnect) Conditions() *[]metav1.Condition { return &ec.Status.Conditions }

func (ec *EdgeConnect) HostPatterns() []string {
	if !ec.IsK8SAutomationEnabled() {
		return ec.Spec.HostPatterns
	}

	var hostPatterns []string

	for _, hostPattern := range ec.Spec.HostPatterns {
		if !strings.EqualFold(hostPattern, ec.K8sAutomationHostPattern()) {
			hostPatterns = append(hostPatterns, hostPattern)
		}
	}

	hostPatterns = append(hostPatterns, ec.K8sAutomationHostPattern())

	return hostPatterns
}

type HostMapping struct {
	From string `json:"from"`
	To   string `json:"to"`
}

func (ec *EdgeConnect) HostMappings() []HostMapping {
	return []HostMapping{{From: ec.K8sAutomationHostPattern(), To: KubernetesDefaultDNS}}
}

func (ec *EdgeConnect) K8sAutomationHostPattern() string {
	return ec.Name + "." + ec.Namespace + "." + ec.Status.KubeSystemUID + "." + kubernetesHostnameSuffix
}
