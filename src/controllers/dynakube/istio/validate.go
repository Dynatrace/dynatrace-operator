package istio

import (
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
)

// CheckIstioEnabled checks if Istio is installed
func CheckIstioEnabled(cfg *rest.Config) (bool, error) {
	discoveryclient, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return false, err
	}
	apiGroupList, err := discoveryclient.ServerGroups()
	if err != nil {
		return false, err
	}

	for _, apiGroup := range apiGroupList.Groups {
		if apiGroup.Name == istioGVRName {
			return true, nil
		}
	}
	return false, nil
}

func verifyIstioCrdAvailability(instance *dynatracev1beta1.DynaKube, config *rest.Config) kubeobjects.ProbeResult {
	var probe kubeobjects.ProbeResult

	probe, _ = kubeobjects.KubernetesObjectProbe(ServiceEntryGVK, instance.GetNamespace(), "", config)
	if probe == kubeobjects.ProbeTypeNotFound {
		return probe
	}

	probe, _ = kubeobjects.KubernetesObjectProbe(VirtualServiceGVK, instance.GetNamespace(), "", config)
	if probe == kubeobjects.ProbeTypeNotFound {
		return probe
	}

	return kubeobjects.ProbeTypeFound
}
