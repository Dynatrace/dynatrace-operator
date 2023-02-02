package kubeobjects

import (
	"context"

	apiv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const crdName = "oneagentapms.dynatrace.com"

// CheckIfOneAgentAPMExists checks if a OneAgentAPM object exists
func CheckIfOneAgentAPMExists(clt client.Client) (bool, error) {
	var crd apiv1.CustomResourceDefinition

	err := clt.Get(context.TODO(), client.ObjectKey{Name: crdName}, &crd)

	if client.IgnoreNotFound(err) != nil {
		return false, err
	}

	if crd.Kind == "OneAgentAPM" {
		return true, nil
	}

	return false, nil
}
