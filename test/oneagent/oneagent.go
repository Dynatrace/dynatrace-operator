package oneagent

import (
	"github.com/Dynatrace/dynatrace-operator/test/daemonset"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func WaitForDaemonset() features.Func {
	return daemonset.WaitFor("dynakube-oneagent", "dynatrace")
}

func DeleteDaemonsetIfExists() env.Func {
	return daemonset.DeleteIfExists("dynakube-oneagent", "dynatrace")
}
