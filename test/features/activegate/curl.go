//go:build e2e

package activegate

import (
	"context"
	"fmt"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/curl"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/pod"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/logs"
	"github.com/stretchr/testify/require"

	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	activeGateEndpoint = "rest/state"
	proxyNamespaceName = "proxy"
)

func curlActiveGateHttps(builder *features.FeatureBuilder, dynakube dynatracev1beta1.DynaKube) {
	podname := "curl-activegate-https"
	serviceUrl := getActiveGateHttpsServiceUrl(dynakube)
	builder.Assess("creating https curl pod for activeGate", installActiveGateCurlPod(podname, serviceUrl, dynakube))
	builder.Assess("waiting for https curl pod for activeGate", waitForActiveGateCurlPod(podname, dynakube))
	builder.Assess("checking https curl pod for activeGate", checkActiveGateCurlResult(podname, dynakube))
	builder.Teardown(removeActiveGateCurlPod(podname, serviceUrl, dynakube))
}

func curlActiveGateHttp(builder *features.FeatureBuilder, dynakube dynatracev1beta1.DynaKube) {
	podname := "curl-activegate-http"
	serviceUrl := getActiveGateHttpServiceUrl(dynakube)
	builder.Assess("creating http curl pod for activeGate", installActiveGateCurlPod(podname, serviceUrl, dynakube))
	builder.Assess("waiting for http curl pod for activeGate", waitForActiveGateCurlPod(podname, dynakube))
	builder.Assess("checking http curl pod for activeGate", checkActiveGateCurlResult(podname, dynakube))
	builder.Teardown(removeActiveGateCurlPod(podname, serviceUrl, dynakube))
}

func installActiveGateCurlPod(podName, serviceUrl string, dynakube dynatracev1beta1.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		curlTarget := fmt.Sprintf("%s/%s", serviceUrl, activeGateEndpoint)

		curlPod := curl.NewPod(podName, curlNamespace(dynakube), curlTarget, curl.WithProxy(dynakube))
		require.NoError(t, envConfig.Client().Resources().Create(ctx, curlPod))
		return ctx
	}
}

func removeActiveGateCurlPod(podName, serviceUrl string, dynakube dynatracev1beta1.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		curlTarget := fmt.Sprintf("%s/%s", serviceUrl, activeGateEndpoint)

		curlPod := curl.NewPod(podName, curlNamespace(dynakube), curlTarget, curl.WithProxy(dynakube))
		err := envConfig.Client().Resources().Delete(ctx, curlPod)
		if !k8sErrors.IsNotFound(err) {
			require.NoError(t, err)
		}
		return ctx
	}
}

func waitForActiveGateCurlPod(podName string, dynakube dynatracev1beta1.DynaKube) features.Func {
	return pod.WaitFor(podName, curlNamespace(dynakube))
}

func checkActiveGateCurlResult(podName string, dynakube dynatracev1beta1.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resources := envConfig.Client().Resources()

		logStream := curl.GetCurlPodLogStream(ctx, t, resources, podName, curlNamespace(dynakube))
		logs.AssertContains(t, logStream, "RUNNING")

		return ctx
	}
}

func curlNamespace(dynakube dynatracev1beta1.DynaKube) string {
	if dynakube.HasProxy() {
		return proxyNamespaceName
	}
	return dynakube.Namespace
}

func getActiveGateHttpsServiceUrl(dynakube dynatracev1beta1.DynaKube) string {
	serviceName := capability.BuildServiceName(dynakube.Name, consts.MultiActiveGateName)
	return fmt.Sprintf("https://%s.%s.svc.cluster.local", serviceName, dynakube.Namespace)
}

func getActiveGateHttpServiceUrl(dynakube dynatracev1beta1.DynaKube) string {
	serviceName := capability.BuildServiceName(dynakube.Name, consts.MultiActiveGateName)
	return fmt.Sprintf("http://%s.%s.svc.cluster.local", serviceName, dynakube.Namespace)
}
