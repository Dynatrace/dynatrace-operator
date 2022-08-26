//go:build e2e
// +build e2e

package test

import (
	"github.com/Dynatrace/dynatrace-operator/test/csi"
	"github.com/Dynatrace/dynatrace-operator/test/environment"
	"github.com/Dynatrace/dynatrace-operator/test/namespace"
	"github.com/Dynatrace/dynatrace-operator/test/operator"
	"github.com/Dynatrace/dynatrace-operator/test/webhook"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/features"
	"testing"
)

const (
	dynatraceNamespace = "dynatrace"
)

var testEnvironment env.Environment

func TestMain(m *testing.M) {
	testEnvironment = environment.Get()
	testEnvironment.Setup(namespace.Recreate(dynatraceNamespace))

	//testEnvironment.Finish(namespace.Delete(dynatraceNamespace))
	testEnvironment.Run(m)
}

func TestCloudNative(t *testing.T) {
	testEnvironment.Test(t, install())
}

func install() features.Feature {
	defaultInstallation := features.New("default installation")
	defaultInstallation.Setup(operator.InstallForKubernetes)
	defaultInstallation.Assess("operator started", operator.WaitForDeployment())
	defaultInstallation.Assess("webhook started", webhook.WaitForDeployment())
	defaultInstallation.Assess("csi driver started", csi.WaitForDaemonset())

	return defaultInstallation.Feature()
}
