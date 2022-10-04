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
	proxyNamespace  = "proxy"
	proxyDeployment = "squid"
	curlPodProxy    = "curl-proxy"
)

var ProxySpec = &v1beta1.DynaKubeProxy{
	Value: "http://squid.proxy:3128",
}

func InstallProxy(builder *features.FeatureBuilder, proxySpec *v1beta1.DynaKubeProxy) {
	if proxySpec != nil {
		builder.Assess("install proxy", manifests.InstallFromFile("../testdata/proxy/proxy.yaml"))
		builder.Assess("proxy started", deployment.WaitFor(proxyDeployment, proxyNamespace))

		builder.Assess("query webhook via proxy", manifests.InstallFromFile("../testdata/activegate/curl-pod-webhook-via-proxy.yaml"))
		builder.Assess("query is completed", waitForCurlProxyPod(curlPodProxy, dynakube.DynatraceNamespace))
		builder.Assess("proxy is running", checkProxyService())
	}
}

func DeleteProxyIfExists() func(ctx context.Context, environmentConfig *envconf.Config, t *testing.T) (context.Context, error) {
	return func(ctx context.Context, environmentConfig *envconf.Config, t *testing.T) (context.Context, error) {
		return namespace.DeleteIfExists(proxyNamespace)(ctx, environmentConfig, t)
	}
}

func checkProxyService() features.Func {
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		resources := environmentConfig.Client().Resources()

		clientset, err := kubernetes.NewForConfig(resources.GetConfig())
		require.NoError(t, err)

		logStream, err := clientset.CoreV1().Pods(dynakube.DynatraceNamespace).GetLogs(curlPodProxy, &corev1.PodLogOptions{
			Container: curlPodProxy,
		}).Stream(ctx)
		require.NoError(t, err)

		logs.AssertLogContains(t, logStream, "CONNECT dynatrace-webhook.dynatrace.svc.cluster.local")

		return ctx
	}
}

func waitForCurlProxyPod(name string, namespace string) features.Func {
	return pod.WaitForCondition(name, namespace, func(object k8s.Object) bool {
		pod, isPod := object.(*corev1.Pod)
		return isPod && pod.Status.Phase == corev1.PodSucceeded
	}, 30*time.Second)
}
