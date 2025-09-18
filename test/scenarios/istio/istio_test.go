//go:build e2e

package istio

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/test/features/activegate"
	"github.com/Dynatrace/dynatrace-operator/test/features/cloudnative/codemodules"
	networkProblems "github.com/Dynatrace/dynatrace-operator/test/features/cloudnative/networkproblems"
	cloudnativeStandard "github.com/Dynatrace/dynatrace-operator/test/features/cloudnative/standard"
	"github.com/Dynatrace/dynatrace-operator/test/helpers"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/operator"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/events"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/istio"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/environment"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/namespace"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/proxy"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

var (
	testEnv env.Environment
	cfg     *envconf.Config
)

func TestMain(m *testing.M) {
	cfg = environment.GetStandardKubeClusterEnvConfig()
	testEnv = env.NewWithConfig(cfg)

	nsWithIstio := *namespace.New(operator.DefaultNamespace, namespace.WithIstio())
	nsWithoutIstio := *namespace.New(operator.DefaultNamespace)
	testEnv.BeforeEachTest(istio.AssertIstioNamespace())
	testEnv.BeforeEachTest(istio.AssertIstiodDeployment())

	testEnv.Setup(
		helpers.SetScheme,
		namespace.CreateForEnv(nsWithIstio),
		operator.InstallViaMake(true),
	)

	// If we cleaned up during a fail-fast (aka.: /debug) it wouldn't be possible to investigate the error.
	if !cfg.FailFast() {
		testEnv.Finish(operator.UninstallViaMake(true))
		testEnv.Finish(namespace.CreateForEnv(nsWithoutIstio))
	}

	testEnv.AfterEachTest(func(ctx context.Context, c *envconf.Config, t *testing.T) (context.Context, error) {
		if t.Failed() {
			events.LogEvents(ctx, c, t)
		}

		return ctx, nil
	})
	testEnv.Run(m)
}

func TestIstio_cloudnative_csi_resilience(t *testing.T) {
	testEnv.Test(t, networkProblems.ResilienceFeature(t))
}

func TestIstio_activegate(t *testing.T) {
	testEnv.Test(t, activegate.Feature(t, proxy.ProxySpec))
}

func TestIstio_cloudnative(t *testing.T) {
	testEnv.Test(t, cloudnativeStandard.Feature(t, true, true))
}

func TestIstio_codemodules_with_proxy_no_certs(t *testing.T) {
	testEnv.Test(t, codemodules.WithProxy(t, proxy.ProxySpec))
}

func TestIstio_codemodules_with_proxy_and_ag_cert(t *testing.T) {
	testEnv.Test(t, codemodules.WithProxyAndAGCert(t, proxy.ProxySpec))
}

func TestIstio_codemodules_with_proxy_and_auto_ag_cert(t *testing.T) {
	testEnv.Test(t, codemodules.WithProxyAndAutomaticAGCert(t, proxy.ProxySpec))
}

func TestIstio_codemodules_with_proxy_custom_ca_ag_cert(t *testing.T) {
	testEnv.Test(t, codemodules.WithProxyCAAndAGCert(t, proxy.HTTPSProxySpec))
}

func TestIstio_codemodules_with_proxy_custom_ca_auto_ag_cert(t *testing.T) {
	testEnv.Test(t, codemodules.WithProxyCAAndAutomaticAGCert(t, proxy.HTTPSProxySpec))
}
