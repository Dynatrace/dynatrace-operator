package kubeobjects

import (
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
)

// CheckIfOneAgentAPMExists checks if a OneAgentAPM object exists
func CheckIfOneAgentAPMExists(cfg *rest.Config) (bool, error) {
	client, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return false, err
	}
	_, resourceList, err := client.ServerGroupsAndResources()
	if err != nil {
		return false, err
	}

	for _, resource := range resourceList {
		for _, apiResource := range resource.APIResources {
			if apiResource.Kind == "OneAgentAPM" {
				return true, nil
			}
		}
	}
	return false, nil
}
