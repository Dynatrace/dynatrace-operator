//go:build e2e

package classic

import (
	"github.com/Dynatrace/dynatrace-operator/src/util/logger"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/environment"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/e2e-framework/pkg/env"
)

var testEnvironment env.Environment

func TestMain(m *testing.M) {
	log.SetLogger(logger.Factory.GetLogger("e2e-classic-fullstack"))

	testEnvironment = environment.GetStandardKubeClusterEnvironment()
	testEnvironment.Run(m)
}

func TestClassicFullStack(t *testing.T) {
	testEnvironment.Test(t, install(t))
}
