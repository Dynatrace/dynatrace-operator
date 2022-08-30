package webhook

import (
	"github.com/Dynatrace/dynatrace-operator/test/kubeobjects/deployment"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func WaitForDeployment() features.Func {
	return deployment.WaitFor("dynatrace-webhook", "dynatrace")
}
