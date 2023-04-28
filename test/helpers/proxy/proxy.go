//go:build e2e

package proxy

import (
	"context"
	"path"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/deployment"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/manifests"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/namespace"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/sampleapps"
	"github.com/Dynatrace/dynatrace-operator/test/project"
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

func SetupProxyWithTeardown(builder *features.FeatureBuilder, testDynakube dynatracev1beta1.DynaKube) {
	if testDynakube.Spec.Proxy != nil {
		installProxySCC(builder)
		builder.Assess("create proxy namespace", namespace.Create(namespace.NewBuilder(proxyNamespaceName).Build()))
		builder.Assess("install proxy", manifests.InstallFromFile(proxyDeploymentPath))
		builder.Assess("proxy started", deployment.WaitFor(proxyDeploymentName, proxyNamespaceName))

		builder.Assess("query webhook via proxy", sampleapps.InstallWebhookCurlProxyPod(testDynakube))
		builder.Assess("query is completed", sampleapps.WaitForWebhookCurlProxyPod(testDynakube))
		builder.Assess("proxy is running", sampleapps.CheckWebhookCurlProxyResult(testDynakube))

		builder.WithTeardown("removing proxy", DeleteProxy())
	}
}

func installProxySCC(builder *features.FeatureBuilder) {
	if kubeobjects.ResolvePlatformFromEnv() == kubeobjects.Openshift {
		builder.Assess("install proxy scc", manifests.InstallFromFile(proxySCCPath))
	}
}

func DeleteProxy() features.Func {
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		return namespace.Delete(proxyNamespaceName)(ctx, t, environmentConfig)
	}
}

func CutOffDynatraceNamespace(builder *features.FeatureBuilder, proxySpec *dynatracev1beta1.DynaKubeProxy) {
	if proxySpec != nil {
		builder.Assess("cut off dynatrace namespace", manifests.InstallFromFile(dynatraceNetworkPolicy))
	}
}

func ApproveConnectionsWithK8SAndProxy(builder *features.FeatureBuilder, proxySpec *dynatracev1beta1.DynaKubeProxy) {
	if proxySpec != nil {
		if kubeobjects.ResolvePlatformFromEnv() == kubeobjects.Openshift {
			builder.Assess("approve dynatrace-openshift network traffic", manifests.InstallFromFile(path.Join(project.TestDataDir(), "network/dynatrace-openshift-approval.yaml")))
		} else {
			builder.Assess("approve dynatrace-kube-system network traffic", manifests.InstallFromFile(path.Join(project.TestDataDir(), "network/dynatrace-kube-system-approval.yaml")))
		}
		builder.Assess("approve dynatrace-proxy network traffic", manifests.InstallFromFile(path.Join(project.TestDataDir(), "network/proxy-approval.yaml")))
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
