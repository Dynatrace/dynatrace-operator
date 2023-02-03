package kubeobjects

import (
	"context"

	apiv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const namespace = "dynatrace"

// CheckIfOneAgentAPMExists checks if a OneAgentAPM object exists
func CheckIfOneAgentAPMExists(clt client.Reader) (bool, error) {
	var crds apiv1.CustomResourceDefinitionList

	err := clt.List(context.TODO(), &crds, &client.ListOptions{
		Namespace: namespace,
	})

	if client.IgnoreNotFound(err) != nil {
		return false, err
	}

	for _, crd := range crds.Items {
		if crd.Kind == "OneAgentAPM" {
			return true, nil
		}
	}

	return false, nil
}
