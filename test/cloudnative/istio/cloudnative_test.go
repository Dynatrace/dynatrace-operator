//go:build e2e

package cloudnativeistio

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/test/cloudnative"
	"github.com/Dynatrace/dynatrace-operator/test/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/istiosetup"
	"github.com/Dynatrace/dynatrace-operator/test/kubeobjects/environment"
	"github.com/Dynatrace/dynatrace-operator/test/kubeobjects/namespace"
	"github.com/Dynatrace/dynatrace-operator/test/oneagent"
	"github.com/Dynatrace/dynatrace-operator/test/sampleapps"
	"sigs.k8s.io/e2e-framework/pkg/env"
)

var testEnvironment env.Environment

func TestMain(m *testing.M) {
	testEnvironment = environment.Get()
	testEnvironment.BeforeEachTest(istiosetup.AssertIstioNamespace())
	testEnvironment.BeforeEachTest(istiosetup.AssertIstiodDeployment())
	testEnvironment.BeforeEachTest(namespace.DeleteIfExists(sampleapps.Namespace))
	testEnvironment.BeforeEachTest(dynakube.DeleteIfExists(dynakube.NewBuilder().WithDefaultObjectMeta().Build()))
	testEnvironment.BeforeEachTest(oneagent.WaitForDaemonSetPodsDeletion())
	testEnvironment.BeforeEachTest(namespace.Recreate(namespace.NewBuilder(dynakube.Namespace).WithLabels(istiosetup.IstioLabel).Build()))

	testEnvironment.AfterEachTest(namespace.DeleteIfExists(sampleapps.Namespace))
	testEnvironment.AfterEachTest(dynakube.DeleteIfExists(dynakube.NewBuilder().WithDefaultObjectMeta().Build()))
	testEnvironment.AfterEachTest(oneagent.WaitForDaemonSetPodsDeletion())
	testEnvironment.AfterEachTest(namespace.Delete(dynakube.Namespace))

	testEnvironment.Run(m)
}

func TestCloudNative(t *testing.T) {
	testEnvironment.Test(t, cloudnative.Install(t, true))
	testEnvironment.Test(t, cloudnative.Upgrade(t, true))
	testEnvironment.Test(t, cloudnative.CodeModules(t, true))
	testEnvironment.Test(t, cloudnative.SpecificAgentVersion(t, true))
}
