//go:build e2e

package network_problems

import (
	"github.com/Dynatrace/dynatrace-operator/src/util/logger"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/test/helpers/istio"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/environment"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/e2e-framework/pkg/env"
)

var testEnvironment env.Environment

func TestMain(m *testing.M) {
	log.SetLogger(logger.Factory.GetLogger("e2e-cloudnative-network_problems"))

	testEnvironment = environment.GetStandardKubeClusterEnvironment()
	testEnvironment.BeforeEachTest(istio.AssertIstioNamespace())
	testEnvironment.BeforeEachTest(istio.AssertIstiodDeployment())
	testEnvironment.Run(m)
}

func TestNetworkProblems(t *testing.T) {
	testEnvironment.Test(t, networkProblems(t))
}
