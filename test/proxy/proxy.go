package proxy

import (
	"context"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/test/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/kubeobjects/deployment"
	"github.com/Dynatrace/dynatrace-operator/test/kubeobjects/manifests"
	"github.com/Dynatrace/dynatrace-operator/test/kubeobjects/namespace"
	"github.com/Dynatrace/dynatrace-operator/test/kubeobjects/pod"
	"github.com/Dynatrace/dynatrace-operator/test/logs"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	ProxyNamespace  = "proxy"
	ProxyDeployment = "squid"
	CurlPodProxy    = "curl-proxy"
)

var ProxySpec = &v1beta1.DynaKubeProxy{
	Value: "http://squid.proxy:3128",
}

func InstallProxy(builder *features.FeatureBuilder, proxySpec *v1beta1.DynaKubeProxy) {
	if proxySpec != nil {
		builder.Assess("install proxy", manifests.InstallFromLocalFile("../testdata/proxy/proxy.yaml"))
		builder.Assess("proxy started", deployment.WaitFor(ProxyDeployment, ProxyNamespace))

		builder.Assess("query webhook via proxy", manifests.InstallFromLocalFile("../testdata/activegate/curl-pod-webhook-via-proxy.yaml"))
		builder.Assess("query is completed", WaitForCurlProxyPod(CurlPodProxy, dynakube.Namespace))
		builder.Assess("proxy is running", CheckProxyService())
	}
}

func DeleteProxyIfExists() func(ctx context.Context, environmentConfig *envconf.Config, t *testing.T) (context.Context, error) {
	return func(ctx context.Context, environmentConfig *envconf.Config, t *testing.T) (context.Context, error) {
		return namespace.DeleteIfExists(ProxyNamespace)(ctx, environmentConfig, t)
	}
}

func CheckProxyService() features.Func {
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		resources := environmentConfig.Client().Resources()

		clientset, err := kubernetes.NewForConfig(resources.GetConfig())
		require.NoError(t, err)

		logStream, err := clientset.CoreV1().Pods(dynakube.Namespace).GetLogs(CurlPodProxy, &corev1.PodLogOptions{
			Container: CurlPodProxy,
		}).Stream(ctx)
		require.NoError(t, err)

		logs.AssertContains(t, logStream, "CONNECT dynatrace-webhook.dynatrace.svc.cluster.local")

		return ctx
	}
}

func WaitForCurlProxyPod(name string, namespace string) features.Func {
	return pod.WaitForCondition(name, namespace, func(object k8s.Object) bool {
		pod, isPod := object.(*corev1.Pod)
		return isPod && pod.Status.Phase == corev1.PodSucceeded
	}, 30*time.Second)
}
