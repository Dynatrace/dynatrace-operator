package operator

import (
	"github.com/Dynatrace/dynatrace-operator/test/kubeobjects/deployment"
	"github.com/Dynatrace/dynatrace-operator/test/kubeobjects/manifests"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func InstallForKubernetes() features.Func {
	return manifests.InstallFromFile("../../config/deploy/kubernetes/kubernetes-all.yaml")
}

func WaitForDeployment() features.Func {
	return deployment.WaitFor("dynatrace-operator", "dynatrace")
}
