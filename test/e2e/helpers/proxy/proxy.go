//go:build e2e

package proxy

import (
	"context"
	"crypto/x509"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/Dynatrace/dynatrace-bootstrapper/pkg/configure/oneagent/pmc"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/proxy"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/value"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/certificates"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/volumes"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/curl"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubeobjects/deployment"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubeobjects/manifests"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubeobjects/namespace"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubeobjects/pod"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubeobjects/secret"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/platform"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/sample"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/shell"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/project"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	proxyNamespaceName  = "proxy"
	proxyDeploymentName = "squid"

	curlPodNameDynatraceInboundTraffic  = "dynatrace-inbound-traffic"
	curlPodNameDynatraceOutboundTraffic = "dynatrace-outbound-traffic"

	internetURL = "https://www.dynatrace.com"
)

var (
	dynatraceNetworkPolicy = filepath.Join(project.TestDataDir(), "network/dynatrace-denial.yaml")

	proxyDeploymentPath                      = filepath.Join(project.TestDataDir(), "network/proxy.yaml")
	proxyWithCustomCADeploymentPath          = filepath.Join(project.TestDataDir(), "network/proxy-ssl.yaml")
	proxyNamespaceWithCustomCADeploymentPath = filepath.Join(project.TestDataDir(), "network/proxy-ssl-namespace.yaml")
	proxySCCPath                             = filepath.Join(project.TestDataDir(), "network/proxy-scc.yaml")

	ProxySpec = &value.Source{
		Value: "http://squid.proxy.svc.cluster.local:3128",
	}
	HTTPSProxySpec = &value.Source{
		Value: "https://squid.proxy.svc.cluster.local:3128",
	}
	EdgeConnectProxySpec = &proxy.Spec{
		Host: "squid.proxy",
		Port: 3128,
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

func SetupProxyWithCustomCAandTeardown(t *testing.T, builder *features.FeatureBuilder, testDynakube dynakube.DynaKube, pemCert []byte, pemPk []byte) {
	if testDynakube.HasProxy() {
		builder.Assess("create proxy namespace", helpers.ToFeatureFunc(manifests.InstallFromFile(proxyNamespaceWithCustomCADeploymentPath), true))
		proxySecret := createProxyTLSSecret(pemCert, pemPk)
		builder.Assess("create proxy TLS secret", secret.Create(proxySecret))
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
		isNetworkTrafficCutOff(builder, "ingress", curlPodNameDynatraceInboundTraffic, proxyNamespaceName, getWebhookServiceURL(testDynakube))
		isNetworkTrafficCutOff(builder, "egress", curlPodNameDynatraceOutboundTraffic, testDynakube.Namespace, internetURL)
	}
}

func isNetworkTrafficCutOff(builder *features.FeatureBuilder, directionName, podName, podNamespaceName, targetURL string) {
	// Pod might take a while to delete, so instead of waiting for them to be deleted just create a new one for each check.
	podName += "-" + rand.String(6) //nolint:mnd // One-off, don't need a constant for this
	builder.Assess(directionName+" - query namespace", curl.InstallCutOffCurlPod(podName, podNamespaceName, targetURL))
	builder.Assess(directionName+" - namespace is cutoff", curl.WaitForCutOffCurlPod(podName, podNamespaceName))
	builder.Teardown(curl.DeleteCutOffCurlPod(podName, podNamespaceName, targetURL))
}

func CheckRuxitAgentProcFileHasProxySetting(sampleApp sample.App, proxySpec *value.Source) features.Func {
	return func(ctx context.Context, t *testing.T, e *envconf.Config) context.Context {
		resources := e.Client().Resources()

		err := deployment.NewQuery(ctx, resources, client.ObjectKey{
			Name:      sampleApp.Name(),
			Namespace: sampleApp.Namespace(),
		}).ForEachPod(func(podItem corev1.Pod) {
			dir := filepath.Join(volumes.ConfigMountPath, pmc.DestinationRuxitAgentProcPath)
			readFileCommand := shell.ReadFile(dir)
			result, err := pod.Exec(ctx, resources, podItem, sampleApp.ContainerName(), readFileCommand...)
			assert.Contains(t, result.StdOut.String(), fmt.Sprintf("proxy %s", proxySpec.Value))
			require.NoError(t, err)
		})

		require.NoError(t, err)

		return ctx
	}
}

func getWebhookServiceURL(dk dynakube.DynaKube) string {
	return fmt.Sprintf("%s.%s.svc.cluster.local", webhook.DeploymentName, dk.Namespace)
}

func createProxyTLSSecret(pemCert []byte, pemPK []byte) corev1.Secret {
	pem := pemCert
	pem = append(pem, byte('\n'))
	pem = append(pem, pemPK...)

	secretData := map[string][]byte{
		"squid-ca-cert.pem": pem,
	}

	proxySecret := secret.New("proxy-ca", "proxy", secretData)
	proxySecret.Type = corev1.SecretTypeOpaque

	return proxySecret
}

func CreateProxyTLSCertAndKey() (pemCert []byte, pemKey []byte, err error) {
	cert, err := certificates.New(timeprovider.New())
	if err != nil {
		return nil, nil, err
	}

	cert.Cert.DNSNames = []string{
		"squid.proxy",
		"squid.proxy.svc",
		"squid.proxy.svc.cluster.local",
	}
	cert.Cert.KeyUsage = x509.KeyUsageKeyEncipherment | x509.KeyUsageDataEncipherment
	cert.Cert.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
	cert.Cert.Subject.CommonName = "squid.proxy"
	cert.Cert.IsCA = true
	cert.Cert.BasicConstraintsValid = true

	err = cert.SelfSign()
	if err != nil {
		return nil, nil, err
	}

	return cert.ToPEM()
}
