//go:build e2e

package cloudnativeistio

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/logger"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/istio"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/environment"
	"github.com/Dynatrace/dynatrace-operator/test/scenarios/cloudnative/basic"
	"github.com/Dynatrace/dynatrace-operator/test/scenarios/cloudnative/codemodules"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/e2e-framework/pkg/env"
)

var testEnvironment env.Environment

func TestMain(m *testing.M) {
	log.SetLogger(logger.Factory.GetLogger("e2e-cloudnative-istio"))

	testEnvironment = environment.Get()
	testEnvironment.BeforeEachTest(istio.AssertIstioNamespace())
	testEnvironment.BeforeEachTest(istio.AssertIstiodDeployment())

	testEnvironment.Run(m)
}

func TestCloudNative(t *testing.T) {
	testEnvironment.Test(t, basic.Install(t, true))
	testEnvironment.Test(t, codemodules.CodeModules(t, true))
}
