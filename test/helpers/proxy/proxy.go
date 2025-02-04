//go:build e2e

package proxy

import (
	"context"
	"fmt"
	"path"
	"path/filepath"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/value"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer/common"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	oamutation "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/oneagent"
	"github.com/Dynatrace/dynatrace-operator/test/helpers"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/curl"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/deployment"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/manifests"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/namespace"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/pod"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/platform"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/sample"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/shell"
	"github.com/Dynatrace/dynatrace-operator/test/project"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	proxyNamespaceName  = "proxy"
	proxyDeploymentName = "squid"

	curlPodNameDynatraceInboundTraffic  = "dynatrace-inbound-traffic"
	curlPodNameDynatraceOutboundTraffic = "dynatrace-outbound-traffic"

	internetUrl = "https://www.dynatrace.com"
)

var (
	dynatraceNetworkPolicy = path.Join(project.TestDataDir(), "network/dynatrace-denial.yaml")

	proxyDeploymentPath             = path.Join(project.TestDataDir(), "network/proxy.yaml")
	proxyWithCustomCADeploymentPath = path.Join(project.TestDataDir(), "network/proxy-ssl.yaml")
	proxySCCPath                    = path.Join(project.TestDataDir(), "network/proxy-scc.yaml")

	ProxySpec = &value.Source{
		Value: "http://squid.proxy.svc.cluster.local:3128",
	}
	HttpsProxySpec = &value.Source{
		Value: "https://squid.proxy.svc.cluster.local:3128",
	}
)

func SetupProxyWithTeardown(t *testing.T, builder *features.FeatureBuilder, testDynakube dynakube.DynaKube) {
	if testDynakube.Spec.Proxy != nil {
		installProxySCC(builder, t)
		builder.Assess("install proxy", helpers.ToFeatureFunc(manifests.InstallFromFile(proxyDeploymentPath), true))
		builder.Assess("proxy started", helpers.ToFeatureFunc(deployment.WaitFor(proxyDeploymentName, proxyNamespaceName), true))
		builder.Assess("proxy ready", checkProxyReady())
		builder.WithTeardown("removing proxy", DeleteProxy())
	}
}

func SetupProxyWithCustomCAandTeardown(t *testing.T, builder *features.FeatureBuilder, testDynakube dynakube.DynaKube) {
	if testDynakube.Spec.Proxy != nil {
		installProxySCC(builder, t)
		builder.Assess("install proxy", helpers.ToFeatureFunc(manifests.InstallFromFile(proxyWithCustomCADeploymentPath), true))
		builder.Assess("proxy started", helpers.ToFeatureFunc(deployment.WaitFor(proxyDeploymentName, proxyNamespaceName), true))
		builder.Assess("proxy ready", checkProxyReady())
		builder.WithTeardown("removing proxy", DeleteProxy())
	}
}

func installProxySCC(builder *features.FeatureBuilder, t *testing.T) {
	isOpenshift, err := platform.NewResolver().IsOpenshift()
	require.NoError(t, err)
	if isOpenshift {
		builder.Assess("install proxy scc", helpers.ToFeatureFunc(manifests.InstallFromFile(proxySCCPath), true))
	}
}

func DeleteProxy() features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		return namespace.Delete(proxyNamespaceName)(ctx, t, envConfig)
	}
}

func checkProxyReady() features.Func {
	return func(ctx context.Context, t *testing.T, envc *envconf.Config) context.Context {
		return helpers.ToFeatureFunc(deployment.WaitFor(proxyDeploymentName, proxyNamespaceName), false)(ctx, t, envc)
	}
}

func CutOffDynatraceNamespace(builder *features.FeatureBuilder, proxySpec *value.Source) {
	if proxySpec != nil {
		builder.Assess("cut off dynatrace namespace", helpers.ToFeatureFunc(manifests.InstallFromFile(dynatraceNetworkPolicy), true))
		builder.Teardown(helpers.ToFeatureFunc(manifests.UninstallFromFile(dynatraceNetworkPolicy), false))
	}
}

func IsDynatraceNamespaceCutOff(builder *features.FeatureBuilder, testDynakube dynakube.DynaKube) {
	if testDynakube.HasProxy() {
		isNetworkTrafficCutOff(builder, "ingress", curlPodNameDynatraceInboundTraffic, proxyNamespaceName, getWebhookServiceUrl(testDynakube))
		isNetworkTrafficCutOff(builder, "egress", curlPodNameDynatraceOutboundTraffic, testDynakube.Namespace, internetUrl)
	}
}

func isNetworkTrafficCutOff(builder *features.FeatureBuilder, directionName, podName, podNamespaceName, targetUrl string) {
	builder.Assess(directionName+" - query namespace", curl.InstallCutOffCurlPod(podName, podNamespaceName, targetUrl))
	builder.Assess(directionName+" - namespace is cutoff", curl.WaitForCutOffCurlPod(podName, podNamespaceName))
	builder.Teardown(curl.DeleteCutOffCurlPod(podName, podNamespaceName, targetUrl))
}

func CheckRuxitAgentProcFileHasProxySetting(sampleApp sample.App, proxySpec *value.Source) features.Func {
	return func(ctx context.Context, t *testing.T, e *envconf.Config) context.Context {
		resources := e.Client().Resources()

		err := deployment.NewQuery(ctx, resources, client.ObjectKey{
			Name:      sampleApp.Name(),
			Namespace: sampleApp.Namespace(),
		}).ForEachPod(func(podItem corev1.Pod) {
			dir := filepath.Join(oamutation.OneAgentConfMountPath, common.RuxitConfFileName)
			readFileCommand := shell.ReadFile(dir)
			result, err := pod.Exec(ctx, resources, podItem, sampleApp.ContainerName(), readFileCommand...)
			assert.Contains(t, result.StdOut.String(), fmt.Sprintf("proxy %s", proxySpec.Value))
			require.NoError(t, err)
		})

		require.NoError(t, err)

		return ctx
	}
}

func getWebhookServiceUrl(dk dynakube.DynaKube) string {
	return fmt.Sprintf("%s.%s.svc.cluster.local", webhook.DeploymentName, dk.Namespace)
}
