//go:build e2e

package applicationmonitoring

import (
	"context"
	"encoding/json"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/address"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/deployment"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/namespace"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/pod"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/sampleapps"
	sample "github.com/Dynatrace/dynatrace-operator/test/helpers/sampleapps/base"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/setup"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/shell"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/tenant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	metadataFile = "/var/lib/dynatrace/enrichment/dt_metadata.json"
)

type metadata struct {
	WorkloadKind string `json:"dt.kubernetes.workload.kind,omitempty"`
	WorkloadName string `json:"dt.kubernetes.workload.name,omitempty"`
}

func dataIngest(t *testing.T) features.Feature {
	builder := features.New("data-ingest")
	secretConfig := tenant.GetSingleTenantSecret(t)
	testDynakube := dynakube.NewBuilder().
		WithDefaultObjectMeta().
		ApiUrl(secretConfig.ApiUrl).
		ApplicationMonitoring(&dynatracev1beta1.ApplicationMonitoringSpec{
			UseCSIDriver: address.Of(false),
		}).Build()

	sampleDeployment := sampleapps.NewSampleDeployment(t, testDynakube)
	sampleDeployment.WithAnnotations(map[string]string{
		webhook.AnnotationOneAgentInject:   "false",
		webhook.AnnotationDataIngestInject: "true",
	})
	samplePod := sampleapps.NewSamplePod(t, testDynakube)
	samplePod.WithAnnotations(map[string]string{
		webhook.AnnotationOneAgentInject:   "false",
		webhook.AnnotationDataIngestInject: "true",
	})

	// Register operator + dynakube install
	steps := setup.NewEnvironmentSetup(
		setup.CreateNamespaceWithoutTeardown(namespace.NewBuilder(testDynakube.Namespace).Build()),
		setup.DeployOperatorViaMake(testDynakube.NeedsCSIDriver()),
		setup.CreateDynakube(secretConfig, testDynakube),
	)
	steps.CreateSetupSteps(builder)

	// Register actual test (+sample cleanup)
	builder.Assess("install sample deployment and wait till ready", sampleDeployment.Install())
	builder.Assess("install sample pod  and wait till ready", samplePod.Install())
	builder.Assess("deployment pods only have data ingest", deploymentPodsHaveOnlyDataIngestInitContainer(sampleDeployment))
	builder.Assess("pod only has data ingest", podHasOnlyDataIngestInitContainer(samplePod))

	builder.WithTeardown("removing samples", sampleDeployment.UninstallNamespace())

	steps.CreateTeardownSteps(builder)

	return builder.Feature()
}

func podHasOnlyDataIngestInitContainer(samplePod sample.App) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		testPod := samplePod.Get(ctx, t, envConfig.Client().Resources()).(*corev1.Pod)

		assessOnlyDataIngestIsInjected(t)(*testPod)
		assessPodHasDataIngestFile(ctx, t, envConfig.Client().Resources(), *testPod)

		return ctx
	}
}

func assessPodHasDataIngestFile(ctx context.Context, t *testing.T, resource *resources.Resources, testPod corev1.Pod) {
	dataIngestMetadata := getDataIngestMetadataFromPod(ctx, t, resource, testPod)

	assert.Equal(t, dataIngestMetadata.WorkloadKind, "Pod")
	assert.Equal(t, dataIngestMetadata.WorkloadName, testPod.Name)
}

func deploymentPodsHaveOnlyDataIngestInitContainer(sampleApp sample.App) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		query := deployment.NewQuery(ctx, envConfig.Client().Resources(), client.ObjectKey{
			Name:      sampleApp.Name(),
			Namespace: sampleApp.Namespace().Name,
		})
		err := query.ForEachPod(assessOnlyDataIngestIsInjected(t))

		require.NoError(t, err)

		err = query.ForEachPod(assessDeploymentHasDataIngestFile(ctx, t, envConfig.Client().Resources(), sampleApp.Name()))

		require.NoError(t, err)

		return ctx
	}
}

func assessDeploymentHasDataIngestFile(ctx context.Context, t *testing.T, resource *resources.Resources, deploymentName string) deployment.PodConsumer {
	return func(pod corev1.Pod) {
		dataIngestMetadata := getDataIngestMetadataFromPod(ctx, t, resource, pod)

		assert.Equal(t, dataIngestMetadata.WorkloadKind, "Deployment")
		assert.Equal(t, dataIngestMetadata.WorkloadName, deploymentName)
	}
}

func getDataIngestMetadataFromPod(ctx context.Context, t *testing.T, resource *resources.Resources, dataIngestPod corev1.Pod) metadata {
	require.NotEmpty(t, dataIngestPod.Spec.Containers)
	dataIngestContainer := dataIngestPod.Spec.Containers[0].Name
	readMetadataCommand := shell.ReadFile(metadataFile)
	result, err := pod.Exec(ctx, resource, dataIngestPod, dataIngestContainer, readMetadataCommand...)

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

		require.Len(t, initContainers, 1)

		installOneAgentContainer := initContainers[0]
		envVars := installOneAgentContainer.Env

		assert.True(t, kubeobjects.EnvVarIsIn(envVars, consts.EnrichmentWorkloadKindEnv))
		assert.True(t, kubeobjects.EnvVarIsIn(envVars, consts.EnrichmentWorkloadNameEnv))
		assert.True(t, kubeobjects.EnvVarIsIn(envVars, consts.EnrichmentInjectedEnv))

		assert.False(t, kubeobjects.EnvVarIsIn(envVars, consts.AgentInjectedEnv))
	}
}
