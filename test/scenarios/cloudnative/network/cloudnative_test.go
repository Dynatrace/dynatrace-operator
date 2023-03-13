//go:build e2e

package cloudnativenetwork

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
	testEnvironment.Test(t, cloudnative.NetworkProblems(t))
}
