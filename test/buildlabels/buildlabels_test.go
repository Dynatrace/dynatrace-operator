//go:build e2e

package buildlabels

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/test/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/kubeobjects/environment"
	"github.com/Dynatrace/dynatrace-operator/test/kubeobjects/namespace"
	"github.com/Dynatrace/dynatrace-operator/test/oneagent"
	"sigs.k8s.io/e2e-framework/pkg/env"
)

var testEnvironment env.Environment

func TestMain(m *testing.M) {
	testEnvironment = environment.Get()
	for _, namespaceName := range namespaceNames {
		testEnvironment.BeforeEachTest(namespace.DeleteIfExists(namespaceName))
	}
	testEnvironment.BeforeEachTest(dynakube.DeleteIfExists(dynakube.NewBuilder().WithDefaultObjectMeta().Build()))
	testEnvironment.BeforeEachTest(dynakube.DeleteIfExists(dynakube.NewBuilder().Name(buildLabelsDynakube).Namespace(dynakube.Namespace).Build()))
	testEnvironment.BeforeEachTest(oneagent.WaitForDaemonSetPodsDeletion())
	testEnvironment.BeforeEachTest(namespace.Recreate(namespace.NewBuilder(dynakube.Namespace).Build()))

	for _, namespaceName := range namespaceNames {
		testEnvironment.AfterEachTest(namespace.DeleteIfExists(namespaceName))
	}
	testEnvironment.AfterEachTest(dynakube.DeleteIfExists(dynakube.NewBuilder().WithDefaultObjectMeta().Build()))
	testEnvironment.AfterEachTest(dynakube.DeleteIfExists(dynakube.NewBuilder().Name(buildLabelsDynakube).Namespace(dynakube.Namespace).Build()))
	testEnvironment.AfterEachTest(oneagent.WaitForDaemonSetPodsDeletion())
	testEnvironment.AfterEachTest(namespace.Delete(dynakube.Namespace))

	testEnvironment.Run(m)
}

func TestBuildLabels(t *testing.T) {
	testEnvironment.Test(t,
		install(t),
		installDynakube(t,
			dynakube.Name,
			map[string]string{}),
		installDynakube(t,
			buildLabelsDynakube,
			map[string]string{
				"feature.dynatrace.com/label-version-detection": "true",
			}),
		installSampleApplications(),
		checkBuildLabels())
}
