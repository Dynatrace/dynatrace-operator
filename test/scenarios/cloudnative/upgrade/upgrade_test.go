//go:build e2e

package upgrade

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/logger"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/environment"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/e2e-framework/pkg/env"
)

var testEnvironment env.Environment

func TestMain(m *testing.M) {
	log.SetLogger(logger.Factory.GetLogger("e2e-cloudnative-upgrade"))

	testEnvironment = environment.Get()
	testEnvironment.Run(m)
}

func TestUpgrade(t *testing.T) {
	testEnvironment.Test(t, upgrade(t))
}
