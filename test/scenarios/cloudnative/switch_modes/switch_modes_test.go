//go:build e2e

package switch_modes

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/util/logger"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/environment"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/e2e-framework/pkg/env"
)

var testEnvironment env.Environment

func TestMain(m *testing.M) {
	log.SetLogger(logger.Factory.GetLogger("e2e-cloudnative-switch-modes"))

	testEnvironment = environment.GetStandardKubeClusterEnvironment()
	testEnvironment.Run(m)
}

func TestSwitchModes(t *testing.T) {
	testEnvironment.Test(t, switchModes(t, "switch mode from cloud native to classic"))
}
