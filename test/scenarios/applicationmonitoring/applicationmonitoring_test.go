//go:build e2e

package applicationmonitoring

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/test/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/kubeobjects/environment"
	"github.com/Dynatrace/dynatrace-operator/test/kubeobjects/namespace"
	"github.com/Dynatrace/dynatrace-operator/test/sampleapps"
)

var testEnvironment *environment.Environment

func TestMain(m *testing.M) {
	testEnvironment = environment.Get()
	for _, namespaceName := range namespaceNames {
		testEnvironment.BeforeEachTest(namespace.DeleteIfExists(namespaceName))
	}
	testEnvironment.BeforeEachTest(namespace.Recreate(namespace.NewBuilder(sampleapps.Namespace).Build()))
	testEnvironment.BeforeEachTest(dynakube.DeleteIfExists(dynakube.NewBuilder().WithDefaultObjectMeta().Build()))
	testEnvironment.BeforeEachTest(dynakube.DeleteIfExists(dynakube.NewBuilder().Name(buildLabelsDynakube).Namespace(dynakube.Namespace).Build()))
	testEnvironment.BeforeEachTest(namespace.Recreate(namespace.NewBuilder(dynakube.Namespace).Build()))

	for _, namespaceName := range namespaceNames {
		testEnvironment.AfterEachTest(namespace.DeleteIfExists(namespaceName))
	}
	testEnvironment.AfterEachTest(dynakube.DeleteIfExists(dynakube.NewBuilder().WithDefaultObjectMeta().Build()))
	testEnvironment.AfterEachTest(dynakube.DeleteIfExists(dynakube.NewBuilder().Name(buildLabelsDynakube).Namespace(dynakube.Namespace).Build()))
	testEnvironment.AfterEachTest(namespace.Delete(sampleapps.Namespace))
	testEnvironment.AfterEachTest(namespace.Delete(dynakube.Namespace))

	testEnvironment.Run(m)
}

func TestApplicationMonitoring(t *testing.T) {
	testEnvironment.Test(t, dataIngest(t))
}

func TestLabelVersionDetection(t *testing.T) {
	testEnvironment.Test(t,
		installOperator(t),
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
