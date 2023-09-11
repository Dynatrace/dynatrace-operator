//go:build e2e

package switch_modes

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/logger"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/environment"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/e2e-framework/pkg/env"
)

var testEnvironment env.Environment

func TestMain(m *testing.M) {
	log.SetLogger(logger.Factory.GetLogger("e2e-classic-fullstack-switch-modes"))

	testEnvironment = environment.Get()
	testEnvironment.Run(m)
}

func TestSwitchModes(t *testing.T) {
	testEnvironment.Test(t, SwitchModes(t, "classic to cloud native upgrade"))
}
