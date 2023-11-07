//go:build e2e

package istio

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/test/features/activegate"
	"github.com/Dynatrace/dynatrace-operator/test/features/cloudnative/codemodules"
	cloudnativeDefault "github.com/Dynatrace/dynatrace-operator/test/features/cloudnative/default"
	networkProblems "github.com/Dynatrace/dynatrace-operator/test/features/cloudnative/network_problems"
	cloudnativeProxy "github.com/Dynatrace/dynatrace-operator/test/features/cloudnative/proxy"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/operator"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/istio"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/environment"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/namespace"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/proxy"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

var testEnv env.Environment

func TestMain(m *testing.M) {
	cfg := environment.GetStandardKubeClusterEnvConfig()
	testEnv = env.NewWithConfig(cfg)

	nsWithIstio := *namespace.New(operator.DefaultNamespace, namespace.WithIstio())
	nsWithoutIstio := *namespace.New(operator.DefaultNamespace)
	testEnv.BeforeEachTest(istio.AssertIstioNamespace())
	testEnv.BeforeEachTest(istio.AssertIstiodDeployment())
	testEnv.Setup(
		namespace.CreateForEnv(nsWithIstio),
		operator.InstallViaMake(true),
	)
	// If we cleaned up during a fail-fast (aka.: /debug) it wouldn't be possible to investigate the error.
	if !cfg.FailFast() {
		testEnv.Finish(operator.UninstallViaMake(true))
		testEnv.Finish(namespace.CreateForEnv(nsWithoutIstio))
	}
	testEnv.Run(m)
}

func TestIstio(t *testing.T) {
	feats := []features.Feature{
		networkProblems.ResilienceFeature(t),
		activegate.Feature(t, proxy.ProxySpec),
		cloudnativeDefault.Feature(t, true),
		codemodules.WithProxy(t, proxy.ProxySpec),
		codemodules.WithProxyCA(t, proxy.ProxySpec),
		cloudnativeProxy.Feature(t, proxy.ProxySpec),
	}
	testEnv.Test(t, feats...)
}
