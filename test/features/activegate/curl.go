//go:build e2e

package activegate

import (
	"context"
	"fmt"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/curl"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/pod"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/logs"
	"github.com/stretchr/testify/require"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	activeGateEndpoint = "rest/state"
	proxyNamespaceName = "proxy"
)

func curlActiveGateHttps(builder *features.FeatureBuilder, dk dynakube.DynaKube) {
	podname := "curl-activegate-https"
	serviceUrl := getActiveGateHttpsServiceUrl(dk)
	builder.Assess("creating https curl pod for activeGate", installActiveGateCurlPod(podname, serviceUrl, dk))
	builder.Assess("waiting for https curl pod for activeGate", waitForActiveGateCurlPod(podname, dk))
	builder.Assess("checking https curl pod for activeGate", checkActiveGateCurlResult(podname, dk))
	builder.Teardown(removeActiveGateCurlPod(podname, serviceUrl, dk))
}

func curlActiveGateHttp(builder *features.FeatureBuilder, dk dynakube.DynaKube) {
	podname := "curl-activegate-http"
	serviceUrl := getActiveGateHttpServiceUrl(dk)
	builder.Assess("creating http curl pod for activeGate", installActiveGateCurlPod(podname, serviceUrl, dk))
	builder.Assess("waiting for http curl pod for activeGate", waitForActiveGateCurlPod(podname, dk))
	builder.Assess("checking http curl pod for activeGate", checkActiveGateCurlResult(podname, dk))
	builder.Teardown(removeActiveGateCurlPod(podname, serviceUrl, dk))
}

func installActiveGateCurlPod(podName, serviceUrl string, dk dynakube.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		curlTarget := fmt.Sprintf("%s/%s", serviceUrl, activeGateEndpoint)

		curlPod := curl.NewPod(podName, curlNamespace(dk), curlTarget, curl.WithProxy(dk))
		require.NoError(t, envConfig.Client().Resources().Create(ctx, curlPod))

		return ctx
	}
}

func removeActiveGateCurlPod(podName, serviceUrl string, dk dynakube.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		curlTarget := fmt.Sprintf("%s/%s", serviceUrl, activeGateEndpoint)

		curlPod := curl.NewPod(podName, curlNamespace(dk), curlTarget, curl.WithProxy(dk))
		err := envConfig.Client().Resources().Delete(ctx, curlPod)
		if !k8sErrors.IsNotFound(err) {
			require.NoError(t, err)
		}

		return ctx
	}
}

func waitForActiveGateCurlPod(podName string, dk dynakube.DynaKube) features.Func {
	return pod.WaitFor(podName, curlNamespace(dk))
}

func checkActiveGateCurlResult(podName string, dk dynakube.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resources := envConfig.Client().Resources()

		logStream := curl.GetCurlPodLogStream(ctx, t, resources, podName, curlNamespace(dk))
		logs.AssertContains(t, logStream, "RUNNING")

		return ctx
	}
}

func curlNamespace(dk dynakube.DynaKube) string {
	if dk.HasProxy() {
		return proxyNamespaceName
	}

	return dk.Namespace
}

func getActiveGateHttpsServiceUrl(dk dynakube.DynaKube) string {
	serviceName := capability.BuildServiceName(dk.Name, consts.MultiActiveGateName)

	return fmt.Sprintf("https://%s.%s.svc.cluster.local", serviceName, dk.Namespace)
}

func getActiveGateHttpServiceUrl(dk dynakube.DynaKube) string {
	serviceName := capability.BuildServiceName(dk.Name, consts.MultiActiveGateName)

	return fmt.Sprintf("http://%s.%s.svc.cluster.local", serviceName, dk.Namespace)
}
