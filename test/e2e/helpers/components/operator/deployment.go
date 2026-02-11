//go:build e2e

package operator

import (
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubernetes/objects/k8sdeployment"
	"sigs.k8s.io/e2e-framework/pkg/env"
)

const (
	DeploymentName   = "dynatrace-operator"
	ContainerName    = "operator"
	DefaultNamespace = "dynatrace"
)

func WaitForDeployment(namespace string) env.Func {
	return k8sdeployment.WaitFor(DeploymentName, namespace)
}
