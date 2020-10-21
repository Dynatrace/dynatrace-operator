package builder

import (
	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/pkg/apis/dynatrace/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/pkg/dtclient"
	corev1 "k8s.io/api/core/v1"
)

func BuildActiveGateQuery(instance *dynatracev1alpha1.DynaKube, pod *corev1.Pod) *dtclient.ActiveGateQuery {
	networkZone := DefaultNetworkZone
	if instance.Spec.NetworkZone != "" {
		networkZone = instance.Spec.NetworkZone
	}

	return &dtclient.ActiveGateQuery{
		Hostname:       pod.Spec.Hostname,
		NetworkAddress: pod.Status.HostIP,
		NetworkZone:    networkZone,
	}
}

const (
	DefaultNetworkZone = "default"
)
