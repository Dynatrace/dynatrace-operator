//go:build e2e

package webhook

import (
	"sigs.k8s.io/e2e-framework/pkg/env"

	"github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/deployment"
)

const (
	DeploymentName = webhook.DeploymentName
)

func WaitForDeployment(namespace string) env.Func {
	return deployment.WaitFor(DeploymentName, namespace)
}
