//go:build e2e

package operator

import (
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubeobjects/deployment"
	"sigs.k8s.io/e2e-framework/pkg/env"
)

const (
	DeploymentName   = "dynatrace-operator"
	ContainerName    = "operator"
	DefaultNamespace = "dynatrace"
)

func WaitForDeployment(namespace string) env.Func {
	return deployment.WaitFor(DeploymentName, namespace)
}
