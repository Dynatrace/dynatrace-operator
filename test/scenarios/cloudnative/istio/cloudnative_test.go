//go:build e2e

package cloudnativeistio

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/test/helpers/istio"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/environment"
	"github.com/Dynatrace/dynatrace-operator/test/scenarios/cloudnative"
	"sigs.k8s.io/e2e-framework/pkg/env"
)

var testEnvironment env.Environment

func TestMain(m *testing.M) {
	testEnvironment = environment.Get()
	testEnvironment.BeforeEachTest(istio.AssertIstioNamespace())
	testEnvironment.BeforeEachTest(istio.AssertIstiodDeployment())

	testEnvironment.Run(m)
}

func TestCloudNative(t *testing.T) {
	testEnvironment.Test(t, cloudnative.Install(t, true))
	testEnvironment.Test(t, cloudnative.Upgrade(t, true))
	testEnvironment.Test(t, cloudnative.CodeModules(t, true))
	testEnvironment.Test(t, cloudnative.SpecificAgentVersion(t, true))
}
