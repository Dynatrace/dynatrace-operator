//go:build e2e

package activegateproxy

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/test/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/kubeobjects/environment"
	"github.com/Dynatrace/dynatrace-operator/test/kubeobjects/namespace"
	"github.com/Dynatrace/dynatrace-operator/test/proxy"
	"github.com/Dynatrace/dynatrace-operator/test/scenarios/activegate"
	"sigs.k8s.io/e2e-framework/pkg/env"
)

var testEnvironment env.Environment

func TestMain(m *testing.M) {
	testEnvironment = environment.Get()

	testEnvironment.BeforeEachTest(dynakube.DeleteIfExists(dynakube.NewBuilder().WithDefaultObjectMeta().Build()))
	testEnvironment.BeforeEachTest(namespace.Recreate(namespace.NewBuilder(dynakube.Namespace).Build()))
	testEnvironment.BeforeEachTest(proxy.DeleteProxyIfExists())

	testEnvironment.AfterEachTest(dynakube.DeleteIfExists(dynakube.NewBuilder().WithDefaultObjectMeta().Build()))
	testEnvironment.AfterEachTest(namespace.Delete(dynakube.Namespace))
	testEnvironment.AfterEachTest(proxy.DeleteProxyIfExists())

	testEnvironment.Run(m)
}

func TestActiveGateProxy(t *testing.T) {
	testEnvironment.Test(t, activegate.Install(t, proxy.ProxySpec))
}
