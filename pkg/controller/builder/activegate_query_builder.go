package builder

import (
	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-activegate-operator/pkg/apis/dynatrace/v1alpha1"
	"github.com/Dynatrace/dynatrace-activegate-operator/pkg/dtclient"
	corev1 "k8s.io/api/core/v1"
)

func BuildActiveGateQuery(instance *dynatracev1alpha1.ActiveGate, pod *corev1.Pod) *dtclient.ActiveGateQuery {
	networkZone := DefaultNetworkZone
	if instance.Spec.NetworkZone != "" {
		networkZone = instance.Spec.NetworkZone
	}

	hostname := ""
	networkAddress := ""
	if pod != nil {
		hostname = pod.Spec.Hostname
		networkAddress = pod.Status.HostIP
	}

	return &dtclient.ActiveGateQuery{
		Hostname:       hostname,
		NetworkAddress: networkAddress,
		NetworkZone:    networkZone,
	}
}

const (
	DefaultNetworkZone = "default"
)
