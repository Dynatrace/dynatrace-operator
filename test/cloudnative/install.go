//go:build e2e

package cloudnative

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/test/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/istiosetup"
	"github.com/Dynatrace/dynatrace-operator/test/kubeobjects/manifests"
	"github.com/Dynatrace/dynatrace-operator/test/kubeobjects/pod"
	"github.com/Dynatrace/dynatrace-operator/test/logs"
	"github.com/Dynatrace/dynatrace-operator/test/oneagent"
	"github.com/Dynatrace/dynatrace-operator/test/sampleapps"
	"github.com/Dynatrace/dynatrace-operator/test/secrets"
	"github.com/Dynatrace/dynatrace-operator/test/setup"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func Install(t *testing.T, istioEnabled bool) features.Feature {
	secretConfig := getSecretConfig(t)

	defaultInstallation := features.New("default installation")

	if istioEnabled {
		defaultInstallation.Setup(manifests.InstallFromLocalFile("../testdata/cloudnativeistio/test-namespace.yaml"))
	} else {
		defaultInstallation.Setup(manifests.InstallFromLocalFile("../testdata/cloudnative/test-namespace.yaml"))
	}
	setup.InstallDynatraceFromSource(defaultInstallation, &secretConfig)
	setup.AssessOperatorDeployment(defaultInstallation)

	setup.DeploySampleApps(defaultInstallation, "../testdata/cloudnative/sample-deployment.yaml")

	dynakubeBuilder := dynakube.NewBuilder().
		WithDefaultObjectMeta().
		ApiUrl(secretConfig.ApiUrl).
		CloudNative(&v1beta1.CloudNativeFullStackSpec{})
	if istioEnabled {
		dynakubeBuilder = dynakubeBuilder.WithIstio()
	}
	defaultInstallation.Assess("dynakube applied", dynakube.Apply(dynakubeBuilder.Build()))

	setup.AssessDynakubeStartup(defaultInstallation)
	assessOneAgentsAreRunning(defaultInstallation)

	if istioEnabled {
		istiosetup.AssessIstio(defaultInstallation)
	}

	return defaultInstallation.Feature()
}

func assessOneAgentsAreRunning(builder *features.FeatureBuilder) {
	builder.Assess("restart sample apps", sampleapps.Restart)
	builder.Assess("sample apps have working init containers", checkInitContainers)
	builder.Assess("osAgent can connect", oneagent.OSAgentCanConnect())
}

func getSecretConfig(t *testing.T) secrets.Secret {
	secretConfig, err := secrets.DefaultSingleTenant(afero.NewOsFs())

	require.NoError(t, err)

	return secretConfig
}

func checkInitContainers(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
	resources := environmentConfig.Client().Resources()
	pods := pod.List(t, ctx, resources, sampleapps.Namespace)
	clientset, err := kubernetes.NewForConfig(resources.GetConfig())

	require.NoError(t, err)

	for _, podItem := range pods.Items {
		if podItem.DeletionTimestamp != nil {
			continue
		}

		require.NotNil(t, podItem)
		require.NotNil(t, podItem.Spec)
		require.NotEmpty(t, podItem.Spec.InitContainers)

		var oneAgentInstallContainer *corev1.Container

		for _, initContainer := range podItem.Spec.InitContainers {
			if initContainer.Name == oneAgentInstallContainerName {
				oneAgentInstallContainer = &initContainer
				break
			}
		}
		require.NotNil(t, oneAgentInstallContainer, "'%s' pod - '%s' container not found", podItem.Name, oneAgentInstallContainerName)

		assert.Equal(t, oneAgentInstallContainerName, oneAgentInstallContainer.Name)

		logStream, err := clientset.CoreV1().Pods(podItem.Namespace).GetLogs(podItem.Name, &corev1.PodLogOptions{
			Container: oneAgentInstallContainerName,
		}).Stream(ctx)

		require.NoError(t, err)
		logs.AssertContains(t, logStream, "standalone agent init completed")

		executionQuery := pod.NewExecutionQuery(podItem, sampleapps.Name, "cat /opt/dynatrace/oneagent-paas/log/nginx/ruxitagent_nginx_myapp-__bootstrap_1.0.log")
		executionResult, err := executionQuery.Execute(environmentConfig.Client().RESTConfig())

		require.NoError(t, err)

		stdOut := executionResult.StdOut.String()
		stdErr := executionResult.StdErr.String()

		assert.NotEmpty(t, stdOut)
		assert.Empty(t, stdErr)
		assert.Contains(t, stdOut, "[native] Dynatrace Bootstrap Agent")
	}

	return ctx
}
