//go:build e2e

package proxy

import (
	"context"
	"fmt"
	"path"
	"path/filepath"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/src/installer/common"
	"github.com/Dynatrace/dynatrace-operator/src/webhook/mutation/pod_mutator/oneagent_mutation"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/deployment"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/manifests"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/namespace"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/pod"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/platform"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/sampleapps"
	sample "github.com/Dynatrace/dynatrace-operator/test/helpers/sampleapps/base"
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

	internetUrl = "dynatrace.com"
)

var (
	dynatraceNetworkPolicy = path.Join(project.TestDataDir(), "network/dynatrace-denial.yaml")

	proxyDeploymentPath = path.Join(project.TestDataDir(), "network/proxy.yaml")
	proxySCCPath        = path.Join(project.TestDataDir(), "network/proxy-scc.yaml")

	ProxySpec = &dynatracev1beta1.DynaKubeProxy{
		Value: "http://squid.proxy.svc.cluster.local:3128",
	}
)

func SetupProxyWithTeardown(t *testing.T, builder *features.FeatureBuilder, testDynakube dynatracev1beta1.DynaKube) {
	if testDynakube.Spec.Proxy != nil {
		installProxySCC(builder, t)
		builder.Assess("install proxy", manifests.InstallFromFile(proxyDeploymentPath))
		builder.Assess("proxy started", deployment.WaitFor(proxyDeploymentName, proxyNamespaceName))
		builder.WithTeardown("removing proxy", DeleteProxy())
	}
}

func installProxySCC(builder *features.FeatureBuilder, t *testing.T) {
	if platform.NewResolver().IsOpenshift(t) {
		builder.Assess("install proxy scc", manifests.InstallFromFile(proxySCCPath))
	}
}

func DeleteProxy() features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		return namespace.Delete(proxyNamespaceName)(ctx, t, envConfig)
	}
}

func CutOffDynatraceNamespace(builder *features.FeatureBuilder, proxySpec *dynatracev1beta1.DynaKubeProxy) {
	if proxySpec != nil {
		builder.Assess("cut off dynatrace namespace", manifests.InstallFromFile(dynatraceNetworkPolicy))
	}
}

func IsDynatraceNamespaceCutOff(builder *features.FeatureBuilder, testDynakube dynatracev1beta1.DynaKube) {
	if testDynakube.HasProxy() {
		isNetworkTrafficCutOff(builder, "ingress", curlPodNameDynatraceInboundTraffic, proxyNamespaceName, sampleapps.GetWebhookServiceUrl(testDynakube))
		isNetworkTrafficCutOff(builder, "egress", curlPodNameDynatraceOutboundTraffic, testDynakube.Namespace, internetUrl)
	}
}

func isNetworkTrafficCutOff(builder *features.FeatureBuilder, directionName, podName, podNamespaceName, targetUrl string) {
	builder.Assess(directionName+" - query namespace", sampleapps.InstallCutOffCurlPod(podName, podNamespaceName, targetUrl))
	builder.Assess(directionName+" - namespace is cutoff", sampleapps.WaitForCutOffCurlPod(podName, podNamespaceName))
}

func CheckRuxitAgentProcFileHasProxySetting(sampleApp sample.App, proxySpec *dynatracev1beta1.DynaKubeProxy) features.Func {
	return func(ctx context.Context, t *testing.T, e *envconf.Config) context.Context {
		resources := e.Client().Resources()

		err := deployment.NewQuery(ctx, resources, client.ObjectKey{
			Name:      sampleApp.Name(),
			Namespace: sampleApp.Namespace().Name,
		}).ForEachPod(func(podItem corev1.Pod) {
			dir := filepath.Join(oneagent_mutation.OneAgentConfMountPath, common.RuxitConfFileName)
			readFileCommand := shell.ReadFile(dir)
			result, err := pod.Exec(ctx, resources, podItem, "sample-dynakube", readFileCommand...)
			assert.Contains(t, result.StdOut.String(), fmt.Sprintf("proxy %s", proxySpec.Value))
			require.NoError(t, err)
		})

		require.NoError(t, err)
		return ctx
	}
}
