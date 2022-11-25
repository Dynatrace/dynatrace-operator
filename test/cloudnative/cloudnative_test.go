//go:build e2e

package cloudnative

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/test/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/kubeobjects/environment"
	"github.com/Dynatrace/dynatrace-operator/test/kubeobjects/namespace"
	"github.com/Dynatrace/dynatrace-operator/test/oneagent"
	"github.com/Dynatrace/dynatrace-operator/test/sampleapps"
	"sigs.k8s.io/e2e-framework/pkg/env"
)

var testEnvironment env.Environment

func TestMain(m *testing.M) {
	testEnvironment = environment.Get()
	testEnvironment.BeforeEachTest(namespace.DeleteIfExists(dynakube.Namespace))
	testEnvironment.BeforeEachTest(namespace.DeleteIfExists(sampleapps.Namespace))
	testEnvironment.BeforeEachTest(oneagent.WaitForDaemonSetPodsDeletion())
	testEnvironment.BeforeEachTest(namespace.Recreate(namespace.NewBuilder(dynakube.Namespace).Build()))

	testEnvironment.AfterEachTest(namespace.DeleteIfExists(sampleapps.Namespace))
	testEnvironment.AfterEachTest(dynakube.DeleteIfExists(dynakube.NewBuilder().WithDefaultObjectMeta().Build()))
	testEnvironment.AfterEachTest(oneagent.WaitForDaemonSetPodsDeletion())
	testEnvironment.AfterEachTest(namespace.Delete(dynakube.Namespace))

	testEnvironment.Run(m)
}

func TestCloudNative(t *testing.T) {
	testEnvironment.Test(t, Install(t, false))
	testEnvironment.Test(t, Upgrade(t))
	testEnvironment.Test(t, CodeModules(t, false))
	testEnvironment.Test(t, NetworkProblems(t))
}
