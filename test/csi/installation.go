package csi

import (
	"github.com/Dynatrace/dynatrace-operator/test/kubeobjects/daemonset"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func WaitForDaemonset() features.Func {
	return daemonset.WaitFor("dynatrace-oneagent-csi-driver", "dynatrace")
}
