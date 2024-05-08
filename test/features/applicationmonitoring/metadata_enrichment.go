//go:build e2e

package applicationmonitoring

import (
	"context"
	"encoding/json"
	"testing"

	dynatracev1beta2 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta2/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/address"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/env"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	"github.com/Dynatrace/dynatrace-operator/test/helpers"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/deployment"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/pod"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/sample"
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

// Verification of the metadata enrichment part of the operator. The test checks that
// enrichment variables are added to the initContainer and dt_metadata.json
// file contains required fields.
func MetadataEnrichment(t *testing.T) features.Feature {
	builder := features.New("metadata-enrichment")
	builder.WithLabel("name", "app-metadata-enrichment")
	secretConfig := tenant.GetSingleTenantSecret(t)
	testDynakube := *dynakube.New(
		dynakube.WithApiUrl(secretConfig.ApiUrl),
		dynakube.WithApplicationMonitoringSpec(&dynatracev1beta2.ApplicationMonitoringSpec{
			UseCSIDriver: address.Of(false),
		}),
	)

	sampleDeployment := sample.NewApp(t, &testDynakube,
		sample.WithName("deploy-app"),
		sample.AsDeployment(),
		sample.WithAnnotations(map[string]string{
			webhook.AnnotationOneAgentInject:           "false",
			webhook.AnnotationMetadataEnrichmentInject: "true",
		}))

	samplePod := sample.NewApp(t, &testDynakube,
		sample.WithName("pod-app"),
		sample.WithAnnotations(map[string]string{
			webhook.AnnotationOneAgentInject:           "false",
			webhook.AnnotationMetadataEnrichmentInject: "true",
		}))

	// dynakube install
	dynakube.Install(builder, helpers.LevelAssess, &secretConfig, testDynakube)

	// Register actual test
	builder.Assess("install sample deployment and wait till ready", sampleDeployment.Install())
	builder.Assess("install sample pod  and wait till ready", samplePod.Install())
	builder.Assess("deployment pods only have metadata enrichment", deploymentPodsHaveOnlyMetadataEnrichmentInitContainer(sampleDeployment))
	builder.Assess("pod only has metadata enrichment", podHasOnlyMetadataEnrichmentInitContainer(samplePod))

	builder.WithTeardown("removing samples", sampleDeployment.Uninstall())
	builder.WithTeardown("removing samples", samplePod.Uninstall())
	dynakube.Delete(builder, helpers.LevelTeardown, testDynakube)

	return builder.Feature()
}

func podHasOnlyMetadataEnrichmentInitContainer(samplePod *sample.App) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		testPod := samplePod.GetPods(ctx, t, envConfig.Client().Resources()).Items[0]

		assessOnlyMetadataEnrichmentIsInjected(t)(testPod)
		assessPodHasMetadataEnrichmentFile(ctx, t, envConfig.Client().Resources(), testPod)

		return ctx
	}
}

func assessPodHasMetadataEnrichmentFile(ctx context.Context, t *testing.T, resource *resources.Resources, testPod corev1.Pod) {
	enrichmentMetadata := getMetadataEnrichmentMetadataFromPod(ctx, t, resource, testPod)

	assert.Equal(t, "Pod", enrichmentMetadata.WorkloadKind)
	assert.Equal(t, testPod.Name, enrichmentMetadata.WorkloadName)
}

func deploymentPodsHaveOnlyMetadataEnrichmentInitContainer(sampleApp *sample.App) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		query := deployment.NewQuery(ctx, envConfig.Client().Resources(), client.ObjectKey{
			Name:      sampleApp.Name(),
			Namespace: sampleApp.Namespace(),
		})
		err := query.ForEachPod(assessOnlyMetadataEnrichmentIsInjected(t))

		require.NoError(t, err)

		err = query.ForEachPod(assessDeploymentHasMetadataEnrichmentFile(ctx, t, envConfig.Client().Resources(), sampleApp.Name()))

		require.NoError(t, err)

		return ctx
	}
}

func assessDeploymentHasMetadataEnrichmentFile(ctx context.Context, t *testing.T, resource *resources.Resources, deploymentName string) deployment.PodConsumer {
	return func(pod corev1.Pod) {
		enrichmentMetadata := getMetadataEnrichmentMetadataFromPod(ctx, t, resource, pod)

		assert.Equal(t, "Deployment", enrichmentMetadata.WorkloadKind)
		assert.Equal(t, deploymentName, enrichmentMetadata.WorkloadName)
	}
}

func getMetadataEnrichmentMetadataFromPod(ctx context.Context, t *testing.T, resource *resources.Resources, enrichedPod corev1.Pod) metadata {
	require.NotEmpty(t, enrichedPod.Spec.Containers)
	enrichedContainer := enrichedPod.Spec.Containers[0].Name
	readMetadataCommand := shell.ReadFile(metadataFile)
	result, err := pod.Exec(ctx, resource, enrichedPod, enrichedContainer, readMetadataCommand...)

	require.NoError(t, err)

	assert.Zero(t, result.StdErr.Len())
	assert.NotEmpty(t, result.StdOut)

	var enrichmentMetadata metadata
	err = json.Unmarshal(result.StdOut.Bytes(), &enrichmentMetadata)

	require.NoError(t, err)

	return enrichmentMetadata
}

func assessOnlyMetadataEnrichmentIsInjected(t *testing.T) deployment.PodConsumer {
	return func(pod corev1.Pod) {
		initContainers := pod.Spec.InitContainers

		require.Len(t, initContainers, 1)

		installOneAgentContainer := initContainers[0]
		envVars := installOneAgentContainer.Env

		assert.True(t, env.IsIn(envVars, consts.EnrichmentWorkloadKindEnv))
		assert.True(t, env.IsIn(envVars, consts.EnrichmentWorkloadNameEnv))
		assert.True(t, env.IsIn(envVars, consts.EnrichmentInjectedEnv))

		assert.False(t, env.IsIn(envVars, consts.AgentInjectedEnv))

		assert.Contains(t, pod.Annotations, webhook.AnnotationWorkloadKind)
		assert.Contains(t, pod.Annotations, webhook.AnnotationWorkloadName)
	}
}
