package oneagent

import (
	"github.com/Dynatrace/dynatrace-operator/test/kubeobjects/daemonset"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func WaitForDaemonset() features.Func {
	return daemonset.WaitFor("dynakube-oneagent", "dynatrace")
}

func WaitForDaemonSetPodsDeletion() env.Func {
	return daemonset.WaitForPodsDeletion("dynakube-oneagent", "dynatrace")
}
