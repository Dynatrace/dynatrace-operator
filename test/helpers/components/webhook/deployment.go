//go:build e2e

package webhook

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubernetes/objects/k8sdeployment"
	"sigs.k8s.io/e2e-framework/pkg/env"
)

const (
	DeploymentName = webhook.DeploymentName
)

func WaitForDeployment(namespace string) env.Func {
	return k8sdeployment.WaitFor(DeploymentName, namespace)
}
