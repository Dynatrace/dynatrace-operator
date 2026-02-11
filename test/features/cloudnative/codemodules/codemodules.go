//go:build e2e

package codemodules

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/value"
	dtcsi "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8senv"
	oacommon "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator/oneagent"
	"github.com/Dynatrace/dynatrace-operator/test/features/cloudnative"
	"github.com/Dynatrace/dynatrace-operator/test/helpers"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/csi"
	dynakubeComponents "github.com/Dynatrace/dynatrace-operator/test/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/istio"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubernetes/objects/k8sconfigmap"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubernetes/objects/k8sdaemonset"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubernetes/objects/k8sdeployment"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubernetes/objects/k8snamespace"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubernetes/objects/k8spod"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubernetes/objects/k8ssecret"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/proxy"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/registry"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/sample"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/shell"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/tenant"
	"github.com/Dynatrace/dynatrace-operator/test/project"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	diskUsageKiBDelta = 1000000

	dataPath                 = "/data/"
	provisionerContainerName = "provisioner"

	agSecretName                    = "ag-ca"
	configMapName                   = "proxy-ca"
	agCertificate                   = "custom-cas/agcrt.pem"
	agCertificateAndPrivateKey      = "custom-cas/agcrtkey.p12"
	agCertificateAndPrivateKeyField = "server.p12"
)

// Verification that the storage in the CSI driver directory does not increase when
// there are multiple tenants and pods which are monitored.
func InstallFromImage(t *testing.T) features.Feature {
	builder := features.New("cloudnative-codemodules-image")
	storageMap := make(map[string]int)
	secretConfigs := tenant.GetMultiTenantSecret(t)
	require.Len(t, secretConfigs, 2)

	cloudNativeDynakube := *dynakubeComponents.New(
		dynakubeComponents.WithName("cloudnative-codemodules"),
		dynakubeComponents.WithCloudNativeSpec(codeModulesCloudNativeSpec(t)),
		dynakubeComponents.WithNameBasedOneAgentNamespaceSelector(),
		dynakubeComponents.WithNameBasedMetadataEnrichmentNamespaceSelector(),
		dynakubeComponents.WithAPIURL(secretConfigs[0].APIURL),
	)

	appDynakube := *dynakubeComponents.New(
		dynakubeComponents.WithName("app-codemodules"),
		dynakubeComponents.WithApplicationMonitoringSpec(&oneagent.ApplicationMonitoringSpec{AppInjectionSpec: *codeModulesAppInjectSpec(t)}),
		dynakubeComponents.WithNameBasedOneAgentNamespaceSelector(),
		dynakubeComponents.WithNameBasedMetadataEnrichmentNamespaceSelector(),
		dynakubeComponents.WithAPIURL(secretConfigs[1].APIURL),
	)

	labels := cloudNativeDynakube.OneAgent().GetNamespaceSelector().MatchLabels
	sampleNamespace := *k8snamespace.New("codemodules-sample", k8snamespace.WithLabels(labels))

	sampleApp := sample.NewApp(t, &cloudNativeDynakube,
		sample.AsDeployment(),
		sample.WithNamespace(sampleNamespace),
	)

	builder.Assess("create sample namespace", sampleApp.InstallNamespace())

	// Register dynakubeComponents install
	dynakubeComponents.Install(builder, helpers.LevelAssess, &secretConfigs[0], cloudNativeDynakube)

	// Register sample app install
	builder.Assess("install sample app", sampleApp.Install())

	// Register actual test
	cloudnative.AssessSampleInitContainers(builder, sampleApp)

	builder.Assess("codemodules have been downloaded", ImageHasBeenDownloaded(cloudNativeDynakube))
	builder.Assess("checking storage used", measureDiskUsage(appDynakube.Namespace, storageMap))
	dynakubeComponents.Install(builder, helpers.LevelAssess, &secretConfigs[1], appDynakube)
	builder.Assess("storage size has not increased", diskUsageDoesNotIncrease(appDynakube.Namespace, storageMap))
	builder.Assess("volumes are mounted correctly", VolumesAreMountedCorrectly(*sampleApp))

	// Register sample, dynakubeComponents and operator uninstall
	builder.Teardown(sampleApp.Uninstall())
	dynakubeComponents.Delete(builder, helpers.LevelTeardown, cloudNativeDynakube)
	dynakubeComponents.Delete(builder, helpers.LevelTeardown, appDynakube)

	return builder.Feature()
}

const (
	httpsProxy = "https_proxy"
)

// Prerequisites: istio service mesh
//
// Setup: CloudNative deployment with CSI driver
//
// Verification that the operator and all deployed OneAgents are able to communicate
// over a http proxy.
//
// Connectivity in the dynatrace namespace and sample application namespace is restricted to
// the local cluster. Sample application is installed. The test checks if DT_PROXY environment
// variable is defined in the *dynakubeComponents-oneagent* container and the *application container*.
func WithProxy(t *testing.T, proxySpec *value.Source) features.Feature {
	builder := features.New("codemodules-with-proxy-no-certs")
	secretConfigs := tenant.GetMultiTenantSecret(t)
	require.Len(t, secretConfigs, 2)

	cloudNativeDynakube := *dynakubeComponents.New(
		dynakubeComponents.WithName("codemodules-with-proxy"),
		dynakubeComponents.WithAPIURL(secretConfigs[0].APIURL),
		dynakubeComponents.WithCloudNativeSpec(codeModulesCloudNativeSpec(t)),
		dynakubeComponents.WithActiveGate(),
		dynakubeComponents.WithIstioIntegration(),
		dynakubeComponents.WithProxy(proxySpec),
		dynakubeComponents.WithAnnotations(map[string]string{
			exp.AGAutomaticTLSCertificateKey: "false",
		}),
	)

	sampleNamespace := *k8snamespace.New("codemodules-sample-with-proxy", k8snamespace.WithIstio())
	sampleApp := sample.NewApp(t, &cloudNativeDynakube,
		sample.AsDeployment(),
		sample.WithNamespace(sampleNamespace),
	)

	builder.Assess("create sample namespace", sampleApp.InstallNamespace())

	// Register proxy create and delete
	proxy.SetupProxyWithTeardown(t, builder, cloudNativeDynakube)
	proxy.CutOffDynatraceNamespace(builder, proxySpec)
	proxy.IsDynatraceNamespaceCutOff(builder, cloudNativeDynakube)

	// Register dynakubeComponents install
	dynakubeComponents.Install(builder, helpers.LevelAssess, &secretConfigs[0], cloudNativeDynakube)

	// Register sample app install
	builder.Assess("install sample app", sampleApp.Install())

	// Register actual test
	cloudnative.AssessSampleInitContainers(builder, sampleApp)
	istio.AssessIstio(builder, cloudNativeDynakube, *sampleApp)
	builder.Assess("codemodules have been downloaded", ImageHasBeenDownloaded(cloudNativeDynakube))

	builder.Assess("check env variables of oneagent pods", checkOneAgentEnvVars(cloudNativeDynakube))
	builder.Assess("check proxy settings in ruxitagentproc.conf", proxy.CheckRuxitAgentProcFileHasProxySetting(*sampleApp, proxySpec))

	cloudnative.AssessSampleContainer(builder, sampleApp, func() []byte { return nil }, nil)
	cloudnative.AssessOneAgentContainer(builder, func() []byte { return nil }, nil)
	cloudnative.AssessActiveGateContainer(builder, &cloudNativeDynakube, nil)

	// Register sample, dynakubeComponents and operator uninstall
	builder.Teardown(sampleApp.Uninstall())
	dynakubeComponents.Delete(builder, helpers.LevelTeardown, cloudNativeDynakube)

	builder.WithTeardown("deleted tenant secret", tenant.DeleteTenantSecret(cloudNativeDynakube.Name, cloudNativeDynakube.Namespace))

	return builder.Feature()
}

func getAgTLSSecret(secret *corev1.Secret) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		err := envConfig.Client().Resources().Get(ctx, secret.Name, secret.Namespace, secret)
		require.NoError(t, err)

		_, ok := secret.Data[dynakube.ServerCertKey]
		require.True(t, ok)

		return ctx
	}
}

func WithProxyAndAGCert(t *testing.T, proxySpec *value.Source) features.Feature {
	builder := features.New("codemodules-with-proxy-and-ag-cert")
	secretConfigs := tenant.GetMultiTenantSecret(t)
	require.Len(t, secretConfigs, 2)

	cloudNativeDynakube := *dynakubeComponents.New(
		dynakubeComponents.WithName("codemodules-with-proxy-and-ag-cert"),
		dynakubeComponents.WithAPIURL(secretConfigs[0].APIURL),
		dynakubeComponents.WithCloudNativeSpec(codeModulesCloudNativeSpec(t)),
		dynakubeComponents.WithActiveGate(),
		dynakubeComponents.WithActiveGateTLSSecret(agSecretName),
		dynakubeComponents.WithIstioIntegration(),
		dynakubeComponents.WithProxy(proxySpec),
	)

	sampleNamespace := *k8snamespace.New("codemodules-sample-with-proxy-custom-ca", k8snamespace.WithIstio())
	sampleApp := sample.NewApp(t, &cloudNativeDynakube,
		sample.AsDeployment(),
		sample.WithNamespace(sampleNamespace),
	)

	builder.Assess("create sample namespace", sampleApp.InstallNamespace())

	// Add ActiveGate TLS secret
	// public certificate for OneAgents
	agCrt, _ := os.ReadFile(filepath.Join(project.TestDataDir(), agCertificate))
	// public certificate and private key for ActiveGate server
	agP12, _ := os.ReadFile(filepath.Join(project.TestDataDir(), agCertificateAndPrivateKey))
	agSecret := k8ssecret.New(agSecretName, cloudNativeDynakube.Namespace,
		map[string][]byte{
			dynakube.ServerCertKey:          agCrt,
			agCertificateAndPrivateKeyField: agP12,
		})
	builder.Assess("create AG TLS secret", k8ssecret.Create(agSecret))

	// Register proxy create and delete
	proxy.SetupProxyWithTeardown(t, builder, cloudNativeDynakube)
	proxy.CutOffDynatraceNamespace(builder, proxySpec)
	proxy.IsDynatraceNamespaceCutOff(builder, cloudNativeDynakube)

	// Register dynakubeComponents install
	dynakubeComponents.Install(builder, helpers.LevelAssess, &secretConfigs[0], cloudNativeDynakube)

	// Register sample app install
	builder.Assess("install sample app", sampleApp.Install())

	// Register actual test
	cloudnative.AssessSampleInitContainers(builder, sampleApp)
	istio.AssessIstio(builder, cloudNativeDynakube, *sampleApp)

	builder.Assess("codemodules have been downloaded", ImageHasBeenDownloaded(cloudNativeDynakube))

	cloudnative.AssessSampleContainer(builder, sampleApp, func() []byte { return agCrt }, nil)
	cloudnative.AssessOneAgentContainer(builder, func() []byte { return agCrt }, nil)
	cloudnative.AssessActiveGateContainer(builder, &cloudNativeDynakube, nil)

	// Register sample, dynakubeComponents and operator uninstall
	builder.Teardown(sampleApp.Uninstall())
	dynakubeComponents.Delete(builder, helpers.LevelTeardown, cloudNativeDynakube)

	builder.WithTeardown("deleted tenant secret", tenant.DeleteTenantSecret(cloudNativeDynakube.Name, cloudNativeDynakube.Namespace))
	builder.WithTeardown("custom tls secret exists", k8ssecret.Exists(agSecretName, cloudNativeDynakube.Namespace))

	return builder.Feature()
}

func WithProxyAndAutomaticAGCert(t *testing.T, proxySpec *value.Source) features.Feature {
	builder := features.New("codemodules-with-proxy-and-auto-ag-cert")
	secretConfigs := tenant.GetMultiTenantSecret(t)
	require.Len(t, secretConfigs, 2)

	cloudNativeDynakube := *dynakubeComponents.New(
		dynakubeComponents.WithName("codemodules-with-proxy"),
		dynakubeComponents.WithAPIURL(secretConfigs[0].APIURL),
		dynakubeComponents.WithCloudNativeSpec(codeModulesCloudNativeSpec(t)),
		dynakubeComponents.WithActiveGate(),
		dynakubeComponents.WithIstioIntegration(),
		dynakubeComponents.WithProxy(proxySpec),
	)

	sampleNamespace := *k8snamespace.New("codemodules-sample-with-proxy", k8snamespace.WithIstio())
	sampleApp := sample.NewApp(t, &cloudNativeDynakube,
		sample.AsDeployment(),
		sample.WithNamespace(sampleNamespace),
	)

	builder.Assess("create sample namespace", sampleApp.InstallNamespace())

	// Register proxy create and delete
	proxy.SetupProxyWithTeardown(t, builder, cloudNativeDynakube)
	proxy.CutOffDynatraceNamespace(builder, proxySpec)
	proxy.IsDynatraceNamespaceCutOff(builder, cloudNativeDynakube)

	// Register dynakubeComponents install
	dynakubeComponents.Install(builder, helpers.LevelAssess, &secretConfigs[0], cloudNativeDynakube)

	// Register sample app install
	builder.Assess("install sample app", sampleApp.Install())

	// Register actual test
	cloudnative.AssessSampleInitContainers(builder, sampleApp)
	istio.AssessIstio(builder, cloudNativeDynakube, *sampleApp)
	builder.Assess("codemodules have been downloaded", ImageHasBeenDownloaded(cloudNativeDynakube))

	builder.Assess("check env variables of oneagent pods", checkOneAgentEnvVars(cloudNativeDynakube))
	builder.Assess("check proxy settings in ruxitagentproc.conf", proxy.CheckRuxitAgentProcFileHasProxySetting(*sampleApp, proxySpec))

	agTLSSecret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cloudNativeDynakube.ActiveGate().GetTLSSecretName(),
			Namespace: cloudNativeDynakube.Namespace,
		},
	}
	builder.Assess("read AG TLS secret", getAgTLSSecret(&agTLSSecret))

	cloudnative.AssessSampleContainer(builder, sampleApp, func() []byte { return agTLSSecret.Data[dynakube.ServerCertKey] }, nil)
	cloudnative.AssessOneAgentContainer(builder, func() []byte { return agTLSSecret.Data[dynakube.ServerCertKey] }, nil)
	cloudnative.AssessActiveGateContainer(builder, &cloudNativeDynakube, nil)

	// Register sample, dynakubeComponents and operator uninstall
	builder.Teardown(sampleApp.Uninstall())
	dynakubeComponents.Delete(builder, helpers.LevelTeardown, cloudNativeDynakube)

	builder.WithTeardown("deleted tenant secret", tenant.DeleteTenantSecret(cloudNativeDynakube.Name, cloudNativeDynakube.Namespace))

	return builder.Feature()
}

func WithProxyCAAndAGCert(t *testing.T, proxySpec *value.Source) features.Feature {
	builder := features.New("codemodules-with-proxy-custom-ca-ag-cert")
	secretConfigs := tenant.GetMultiTenantSecret(t)
	require.Len(t, secretConfigs, 2)

	cloudNativeDynakube := *dynakubeComponents.New(
		dynakubeComponents.WithName("codemodules-with-proxy-custom-ca-ag-cert"),
		dynakubeComponents.WithAPIURL(secretConfigs[0].APIURL),
		dynakubeComponents.WithCloudNativeSpec(codeModulesCloudNativeSpec(t)),
		dynakubeComponents.WithCustomCAs(configMapName),
		dynakubeComponents.WithActiveGate(),
		dynakubeComponents.WithActiveGateTLSSecret(agSecretName),
		dynakubeComponents.WithIstioIntegration(),
		dynakubeComponents.WithProxy(proxySpec),
	)

	sampleNamespace := *k8snamespace.New("codemodules-sample-with-proxy-custom-ca", k8snamespace.WithIstio())
	sampleApp := sample.NewApp(t, &cloudNativeDynakube,
		sample.AsDeployment(),
		sample.WithNamespace(sampleNamespace),
	)

	builder.Assess("create sample namespace", sampleApp.InstallNamespace())

	// Add ActiveGate TLS secret
	// public certificate for OneAgents
	agCrt, _ := os.ReadFile(filepath.Join(project.TestDataDir(), agCertificate))
	// public certificate and private key for ActiveGate server
	agP12, _ := os.ReadFile(filepath.Join(project.TestDataDir(), agCertificateAndPrivateKey))
	agSecret := k8ssecret.New(agSecretName, cloudNativeDynakube.Namespace,
		map[string][]byte{
			dynakube.ServerCertKey:          agCrt,
			agCertificateAndPrivateKeyField: agP12,
		})
	builder.Assess("create AG TLS secret", k8ssecret.Create(agSecret))

	proxyCert, proxyPk, err := proxy.CreateProxyTLSCertAndKey()
	require.NoError(t, err, "failed to create proxy TLS secret")

	// Add customCA config map
	trustedCa := proxyCert
	caConfigMap := k8sconfigmap.New(configMapName, cloudNativeDynakube.Namespace,
		map[string]string{dynakube.TrustedCAKey: string(trustedCa)})
	builder.Assess("create trusted CAs config map", k8sconfigmap.Create(caConfigMap))

	// Register proxy create and delete
	proxy.SetupProxyWithCustomCAandTeardown(t, builder, cloudNativeDynakube, proxyCert, proxyPk)
	proxy.CutOffDynatraceNamespace(builder, proxySpec)
	proxy.IsDynatraceNamespaceCutOff(builder, cloudNativeDynakube)

	// Register dynakubeComponents install
	dynakubeComponents.Install(builder, helpers.LevelAssess, &secretConfigs[0], cloudNativeDynakube)

	// Register sample app install
	builder.Assess("install sample app", sampleApp.Install())

	// Register actual test
	cloudnative.AssessSampleInitContainers(builder, sampleApp)
	istio.AssessIstio(builder, cloudNativeDynakube, *sampleApp)

	builder.Assess("codemodules have been downloaded", ImageHasBeenDownloaded(cloudNativeDynakube))

	cloudnative.AssessSampleContainer(builder, sampleApp, func() []byte { return agCrt }, trustedCa)
	cloudnative.AssessOneAgentContainer(builder, func() []byte { return agCrt }, trustedCa)
	cloudnative.AssessActiveGateContainer(builder, &cloudNativeDynakube, trustedCa)

	// Register sample, dynakubeComponents and operator uninstall
	builder.Teardown(sampleApp.Uninstall())
	dynakubeComponents.Delete(builder, helpers.LevelTeardown, cloudNativeDynakube)

	builder.WithTeardown("deleted tenant secret", tenant.DeleteTenantSecret(cloudNativeDynakube.Name, cloudNativeDynakube.Namespace))
	builder.WithTeardown("deleted trusted CAs config map", k8sconfigmap.Delete(caConfigMap))
	builder.WithTeardown("custom tls secret exists", k8ssecret.Exists(agSecretName, cloudNativeDynakube.Namespace))

	return builder.Feature()
}

func WithProxyCAAndAutomaticAGCert(t *testing.T, proxySpec *value.Source) features.Feature {
	builder := features.New("codemodules-with-proxy-custom-ca-auto-ag-cert")
	secretConfigs := tenant.GetMultiTenantSecret(t)
	require.Len(t, secretConfigs, 2)

	cloudNativeDynakube := *dynakubeComponents.New(
		dynakubeComponents.WithName("codemodules-with-proxy-custom-ca-ag-cert"),
		dynakubeComponents.WithAPIURL(secretConfigs[0].APIURL),
		dynakubeComponents.WithCloudNativeSpec(codeModulesCloudNativeSpec(t)),
		dynakubeComponents.WithCustomCAs(configMapName),
		dynakubeComponents.WithActiveGate(),
		dynakubeComponents.WithActiveGateTLSSecret(agSecretName),
		dynakubeComponents.WithIstioIntegration(),
		dynakubeComponents.WithProxy(proxySpec),
	)

	sampleNamespace := *k8snamespace.New("codemodules-sample-with-proxy-custom-ca", k8snamespace.WithIstio())
	sampleApp := sample.NewApp(t, &cloudNativeDynakube,
		sample.AsDeployment(),
		sample.WithNamespace(sampleNamespace),
	)

	builder.Assess("create sample namespace", sampleApp.InstallNamespace())

	proxyCert, proxyPk, err := proxy.CreateProxyTLSCertAndKey()
	require.NoError(t, err, "failed to create proxy TLS secret")

	// Add customCA config map
	trustedCa := proxyCert
	caConfigMap := k8sconfigmap.New(configMapName, cloudNativeDynakube.Namespace,
		map[string]string{dynakube.TrustedCAKey: string(trustedCa)})
	builder.Assess("create trusted CAs config map", k8sconfigmap.Create(caConfigMap))

	// Register proxy create and delete
	proxy.SetupProxyWithCustomCAandTeardown(t, builder, cloudNativeDynakube, proxyCert, proxyPk)
	proxy.CutOffDynatraceNamespace(builder, proxySpec)
	proxy.IsDynatraceNamespaceCutOff(builder, cloudNativeDynakube)

	// Register dynakubeComponents install
	dynakubeComponents.Install(builder, helpers.LevelAssess, &secretConfigs[0], cloudNativeDynakube)

	// Register sample app install
	builder.Assess("install sample app", sampleApp.Install())

	// Register actual test
	cloudnative.AssessSampleInitContainers(builder, sampleApp)
	istio.AssessIstio(builder, cloudNativeDynakube, *sampleApp)

	builder.Assess("codemodules have been downloaded", ImageHasBeenDownloaded(cloudNativeDynakube))

	agTLSSecret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cloudNativeDynakube.ActiveGate().GetTLSSecretName(),
			Namespace: cloudNativeDynakube.Namespace,
		},
	}
	builder.Assess("read AG TLS secret", getAgTLSSecret(&agTLSSecret))

	cloudnative.AssessSampleContainer(builder, sampleApp, func() []byte { return agTLSSecret.Data[dynakube.ServerCertKey] }, trustedCa)
	cloudnative.AssessOneAgentContainer(builder, func() []byte { return agTLSSecret.Data[dynakube.ServerCertKey] }, trustedCa)
	cloudnative.AssessActiveGateContainer(builder, &cloudNativeDynakube, trustedCa)

	// Register sample, dynakubeComponents and operator uninstall
	builder.Teardown(sampleApp.Uninstall())
	dynakubeComponents.Delete(builder, helpers.LevelTeardown, cloudNativeDynakube)

	builder.WithTeardown("deleted tenant secret", tenant.DeleteTenantSecret(cloudNativeDynakube.Name, cloudNativeDynakube.Namespace))
	builder.WithTeardown("deleted trusted CAs config map", k8sconfigmap.Delete(caConfigMap))

	return builder.Feature()
}

func codeModulesCloudNativeSpec(t *testing.T) *oneagent.CloudNativeFullStackSpec {
	return &oneagent.CloudNativeFullStackSpec{
		AppInjectionSpec: *codeModulesAppInjectSpec(t),
	}
}

func codeModulesAppInjectSpec(t *testing.T) *oneagent.AppInjectionSpec {
	return &oneagent.AppInjectionSpec{
		CodeModulesImage: registry.GetLatestCodeModulesImageURI(t),
	}
}

func ImageHasBeenDownloaded(dk dynakube.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resource := envConfig.Client().Resources()
		clientset, err := kubernetes.NewForConfig(resource.GetConfig())
		require.NoError(t, err)

		err = k8sdaemonset.NewQuery(ctx, resource, client.ObjectKey{
			Name:      csi.DaemonSetName,
			Namespace: dk.Namespace,
		}).ForEachPod(func(podItem corev1.Pod) {
			err = wait.For(func(ctx context.Context) (done bool, err error) {
				logStream, err := clientset.CoreV1().Pods(podItem.Namespace).GetLogs(podItem.Name, &corev1.PodLogOptions{
					Container: provisionerContainerName,
				}).Stream(ctx)
				require.NoError(t, err)
				buffer := new(bytes.Buffer)
				_, err = io.Copy(buffer, logStream)
				isNew := strings.Contains(buffer.String(), "Installed agent version: "+dk.OneAgent().GetCustomCodeModulesImage())
				isOld := strings.Contains(buffer.String(), "agent already installed")
				t.Logf("wait for Installed agent version in %s", podItem.Name)

				return isNew || isOld, err
			}, wait.WithTimeout(time.Minute*5))
			require.NoError(t, err)

			listCommand := shell.ListDirectory(dataPath)
			result, err := k8spod.Exec(ctx, resource, podItem, provisionerContainerName, listCommand...)

			require.NoError(t, err)
			assert.Contains(t, result.StdOut.String(), dtcsi.SharedAgentBinDir)
		})

		require.NoError(t, err)

		return ctx
	}
}

func measureDiskUsage(namespace string, storageMap map[string]int) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resource := envConfig.Client().Resources()
		err := k8sdaemonset.NewQuery(ctx, resource, client.ObjectKey{
			Name:      csi.DaemonSetName,
			Namespace: namespace,
		}).ForEachPod(func(podItem corev1.Pod) {
			diskUsage := getDiskUsage(ctx, t, envConfig.Client().Resources(), podItem, provisionerContainerName, dataPath)
			storageMap[podItem.Name] = diskUsage
		})
		require.NoError(t, err)

		return ctx
	}
}

func diskUsageDoesNotIncrease(namespace string, storageMap map[string]int) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resource := envConfig.Client().Resources()
		err := k8sdaemonset.NewQuery(ctx, resource, client.ObjectKey{
			Name:      csi.DaemonSetName,
			Namespace: namespace,
		}).ForEachPod(func(podItem corev1.Pod) {
			diskUsage := getDiskUsage(ctx, t, envConfig.Client().Resources(), podItem, provisionerContainerName, dataPath)
			assert.InDelta(t, storageMap[podItem.Name], diskUsage, diskUsageKiBDelta)
		})
		require.NoError(t, err)

		return ctx
	}
}

func getDiskUsage(ctx context.Context, t *testing.T, resource *resources.Resources, podItem corev1.Pod, containerName, path string) int { //nolint:revive
	diskUsageCommand := shell.Shell(
		shell.Pipe(
			shell.DiskUsageWithTotal(path),
			shell.FilterLastLineOnly(),
		),
	)
	result, err := k8spod.Exec(ctx, resource, podItem, containerName, diskUsageCommand...)
	require.NoError(t, err)

	diskUsage, err := strconv.Atoi(strings.Split(result.StdOut.String(), "\t")[0])
	require.NoError(t, err)

	return diskUsage
}

func VolumesAreMountedCorrectly(sampleApp sample.App) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resource := envConfig.Client().Resources()
		err := k8sdeployment.NewQuery(ctx, resource, client.ObjectKey{
			Name:      sampleApp.Name(),
			Namespace: sampleApp.Namespace(),
		}).ForEachPod(func(podItem corev1.Pod) {
			volumes := podItem.Spec.Volumes
			volumeMounts := podItem.Spec.Containers[0].VolumeMounts

			assert.True(t, isVolumeAttached(t, volumes, oacommon.BinVolumeName))
			assert.True(t, isVolumeMounted(t, volumeMounts, oacommon.BinVolumeName))

			listCommand := shell.ListDirectory(oacommon.DefaultInstallPath)
			executionResult, err := k8spod.Exec(ctx, resource, podItem, sampleApp.ContainerName(), listCommand...)

			require.NoError(t, err)
			assert.NotEmpty(t, executionResult.StdOut.String())

			diskUsage := getDiskUsage(ctx, t, envConfig.Client().Resources(), podItem, sampleApp.ContainerName(), oacommon.DefaultInstallPath)
			assert.Positive(t, diskUsage)
		})

		require.NoError(t, err)

		return ctx
	}
}

func isVolumeMounted(t *testing.T, volumeMounts []corev1.VolumeMount, volumeMountName string) bool {
	result := false
	for _, volumeMount := range volumeMounts {
		if volumeMount.Name == volumeMountName {
			result = true

			assert.Equal(t, oacommon.DefaultInstallPath, volumeMount.MountPath)
			assert.True(t, volumeMount.ReadOnly)
		}
	}

	return result
}

func isVolumeAttached(t *testing.T, volumes []corev1.Volume, volumeName string) bool {
	result := false
	for _, volume := range volumes {
		if volume.Name == volumeName {
			result = true

			require.NotNil(t, volume.CSI)
			assert.Equal(t, dtcsi.DriverName, volume.CSI.Driver)

			if volume.CSI.ReadOnly != nil {
				assert.True(t, *volume.CSI.ReadOnly)
			}
		}
	}

	return result
}

func checkOneAgentEnvVars(dk dynakube.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resources := envConfig.Client().Resources()
		err := k8sdaemonset.NewQuery(ctx, resources, client.ObjectKey{
			Name:      dk.OneAgent().GetDaemonsetName(),
			Namespace: dk.Namespace,
		}).ForEachPod(func(podItem corev1.Pod) {
			require.NotNil(t, podItem)
			require.NotNil(t, podItem.Spec)

			checkEnvVarsInContainer(t, podItem, dk.OneAgent().GetDaemonsetName(), httpsProxy)
		})

		require.NoError(t, err)

		return ctx
	}
}

func checkEnvVarsInContainer(t *testing.T, podItem corev1.Pod, containerName string, envVar string) {
	for _, container := range podItem.Spec.Containers {
		if container.Name == containerName {
			require.NotNil(t, container.Env)
			require.True(t, k8senv.Contains(container.Env, envVar))
			for _, env := range container.Env {
				if env.Name == envVar {
					require.NotNil(t, env.Value)
				}
			}
		}
	}
}
