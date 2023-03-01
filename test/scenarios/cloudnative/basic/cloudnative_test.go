//go:build e2e

package cloudnative

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/environment"
	"github.com/Dynatrace/dynatrace-operator/test/scenarios/cloudnative"
	"sigs.k8s.io/e2e-framework/pkg/env"
)

var testEnvironment env.Environment

func TestMain(m *testing.M) {
	testEnvironment = environment.Get()
	testEnvironment.Run(m)
}

func TestCloudNative(t *testing.T) {
	testEnvironment.Test(t, cloudnative.Install(t, false))
	testEnvironment.Test(t, cloudnative.Upgrade(t, false))
	testEnvironment.Test(t, cloudnative.CodeModules(t, false))
	testEnvironment.Test(t, cloudnative.SpecificAgentVersion(t, false))
}
