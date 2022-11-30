//go:build e2e

package appmon

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/config"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects/address"
	"github.com/Dynatrace/dynatrace-operator/test/bash"
	"github.com/Dynatrace/dynatrace-operator/test/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/kubeobjects/deployment"
	"github.com/Dynatrace/dynatrace-operator/test/kubeobjects/manifests"
	"github.com/Dynatrace/dynatrace-operator/test/kubeobjects/pod"
	"github.com/Dynatrace/dynatrace-operator/test/operator"
	"github.com/Dynatrace/dynatrace-operator/test/sampleapps"
	"github.com/Dynatrace/dynatrace-operator/test/secrets"
	"github.com/Dynatrace/dynatrace-operator/test/webhook"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	sampleApps   = "../testdata/application-monitoring/sample-apps.yaml"
	metadataFile = "/var/lib/dynatrace/enrichment/dt_metadata.json"
)

type metadata struct {
	WorkloadKind string `json:"dt.kubernetes.workload.kind,omitempty"`
	WorkloadName string `json:"dt.kubernetes.workload.name,omitempty"`
}

func dataIngest(t *testing.T) features.Feature {
	dataIngestFeature := features.New("data-ingest")
	tenantSecret, err := secrets.DefaultSingleTenant(afero.NewOsFs())
	dataIngestDynakube := dynakube.NewBuilder().
		WithDefaultObjectMeta().
		ApiUrl(tenantSecret.ApiUrl).
		ApplicationMonitoring(&v1beta1.ApplicationMonitoringSpec{
			UseCSIDriver: address.Of(false),
		}).Build()

	require.NoError(t, err)

	dataIngestFeature.Setup(operator.InstallFromSource(false))
	dataIngestFeature.Setup(operator.WaitForDeployment())
	dataIngestFeature.Setup(webhook.WaitForDeployment())
	dataIngestFeature.Setup(secrets.ApplyDefault(tenantSecret))
	dataIngestFeature.Setup(dynakube.Apply(dataIngestDynakube))
	dataIngestFeature.Setup(dynakube.WaitForDynakubePhase(dataIngestDynakube))
	dataIngestFeature.Setup(manifests.InstallFromFile(sampleApps))
	dataIngestFeature.Setup(deployment.WaitFor("test-deployment", sampleapps.Namespace))
	dataIngestFeature.Setup(pod.WaitFor("test-pod", sampleapps.Namespace))

	dataIngestFeature.Assess("deployment pods only have data ingest", deploymentPodsHaveOnlyDataIngestInitContainer())
	dataIngestFeature.Assess("pod only has data ingest", podHasOnlyDataIngestInitContainer())

	return dataIngestFeature.Feature()
}

func podHasOnlyDataIngestInitContainer() features.Func {
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		var testPod corev1.Pod
		err := environmentConfig.Client().Resources().Get(ctx, "test-pod", sampleapps.Namespace, &testPod)

		require.NoError(t, err)

		assessOnlyDataIngestIsInjected(t)(testPod)
		assessPodHasDataIngestFile(t, environmentConfig.Client().RESTConfig(), testPod)

		return ctx
	}
}

func assessPodHasDataIngestFile(t *testing.T, restConfig *rest.Config, testPod corev1.Pod) {
	dataIngestMetadata := getDataIngestMetadataFromPod(t, restConfig, testPod)

	assert.Equal(t, dataIngestMetadata.WorkloadKind, "Pod")
	assert.Equal(t, dataIngestMetadata.WorkloadName, "test-pod")
}

func deploymentPodsHaveOnlyDataIngestInitContainer() features.Func {
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		query := deployment.NewQuery(ctx, environmentConfig.Client().Resources(), client.ObjectKey{
			Name:      "test-deployment",
			Namespace: sampleapps.Namespace,
		})
		err := query.ForEachPod(assessOnlyDataIngestIsInjected(t))

		require.NoError(t, err)

		err = query.ForEachPod(assessDeploymentHasDataIngestFile(t, environmentConfig.Client().RESTConfig()))

		require.NoError(t, err)

		return ctx
	}
}

func assessDeploymentHasDataIngestFile(t *testing.T, restConfig *rest.Config) deployment.PodConsumer {
	return func(pod corev1.Pod) {
		dataIngestMetadata := getDataIngestMetadataFromPod(t, restConfig, pod)

		assert.Equal(t, dataIngestMetadata.WorkloadKind, "Deployment")
		assert.Equal(t, dataIngestMetadata.WorkloadName, "test-deployment")
	}
}

func getDataIngestMetadataFromPod(t *testing.T, restConfig *rest.Config, dataIngestPod corev1.Pod) metadata {
	query := pod.NewExecutionQuery(dataIngestPod, dataIngestPod.Spec.Containers[0].Name, bash.ReadFile(metadataFile)...)
	result, err := query.Execute(restConfig)

	require.NoError(t, err)

	assert.Zero(t, result.StdErr.Len())
	assert.NotEmpty(t, result.StdOut)

	var dataIngestMetadata metadata
	err = json.Unmarshal(result.StdOut.Bytes(), &dataIngestMetadata)

	require.NoError(t, err)

	return dataIngestMetadata
}

func assessOnlyDataIngestIsInjected(t *testing.T) deployment.PodConsumer {
	return func(pod corev1.Pod) {
		initContainers := pod.Spec.InitContainers

		assert.Len(t, initContainers, 1)

		installOneAgentContainer := initContainers[0]
		envVars := installOneAgentContainer.Env

		assert.True(t, kubeobjects.EnvVarIsIn(envVars, config.EnrichmentWorkloadKindEnv))
		assert.True(t, kubeobjects.EnvVarIsIn(envVars, config.EnrichmentWorkloadNameEnv))
		assert.True(t, kubeobjects.EnvVarIsIn(envVars, config.EnrichmentInjectedEnv))

		assert.False(t, kubeobjects.EnvVarIsIn(envVars, config.AgentInjectedEnv))
	}
}
