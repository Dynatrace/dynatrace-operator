//go:build e2e

package _default

import (
	"github.com/Dynatrace/dynatrace-operator/src/util/logger"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/environment"
	"github.com/Dynatrace/dynatrace-operator/test/scenarios/activegate"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/e2e-framework/pkg/env"
)

var testEnvironment env.Environment

func TestMain(m *testing.M) {
	log.SetLogger(logger.Factory.GetLogger("e2e-activegate-default"))
	testEnvironment = environment.GetStandardKubeClusterEnvironment()
	testEnvironment.Run(m)
}

func TestActiveGate(t *testing.T) {
	testEnvironment.Test(t, activegate.Install(t, nil))
}
