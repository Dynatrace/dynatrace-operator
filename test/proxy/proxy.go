package proxy

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/test/kubeobjects/deployment"
	"github.com/Dynatrace/dynatrace-operator/test/kubeobjects/manifests"
	"github.com/Dynatrace/dynatrace-operator/test/kubeobjects/namespace"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	proxyNamespace  = "proxy"
	proxyDeployment = "squid"
)

var ProxySpec = &v1beta1.DynaKubeProxy{
	Value: "http://squid.proxy:3128",
}

func InstallProxy(builder *features.FeatureBuilder, proxySpec *v1beta1.DynaKubeProxy) {
	if proxySpec != nil {
		builder.Assess("install proxy", manifests.InstallFromFile("../testdata/proxy/proxy.yaml"))
		builder.Assess("proxy started", deployment.WaitFor(proxyDeployment, proxyNamespace))
	}
}

func DeleteProxyIfExists() func(ctx context.Context, environmentConfig *envconf.Config, t *testing.T) (context.Context, error) {
	return func(ctx context.Context, environmentConfig *envconf.Config, t *testing.T) (context.Context, error) {
		return namespace.DeleteIfExists(proxyNamespace)(ctx, environmentConfig, t)
	}
}
