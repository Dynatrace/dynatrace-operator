//go:build e2e

package applicationmonitoring

import (
	"context"
	"fmt"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/cmd/bootstrapper"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	maputil "github.com/Dynatrace/dynatrace-operator/pkg/util/map"
	metacommon "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator/metadata"
	oacommon "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator/oneagent"
	"github.com/Dynatrace/dynatrace-operator/test/helpers"
	dynakubeComponents "github.com/Dynatrace/dynatrace-operator/test/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/metadataenrichment"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubernetes/objects/k8sdeployment"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/sample"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/tenant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

// Verification of the metadata enrichment part of the operator. The test checks that
// enrichment variables are added to the initContainer and dt_metadata.json
// file contains required fields.
func MetadataEnrichment(t *testing.T) features.Feature {
	builder := features.New("metadata-enrichment")
	secretConfig := tenant.GetSingleTenantSecret(t)
	testDynakube := *dynakubeComponents.New(
		dynakubeComponents.WithAPIURL(secretConfig.APIURL),
		dynakubeComponents.WithMetadataEnrichment(),
		dynakubeComponents.WithApplicationMonitoringSpec(&oneagent.ApplicationMonitoringSpec{}),
		dynakubeComponents.WithNameBasedMetadataEnrichmentNamespaceSelector(),
		dynakubeComponents.WithNameBasedOneAgentNamespaceSelector(),
	)

	type testCase struct {
		name   string
		app    *sample.App
		assess func(samplePod *sample.App) features.Func
	}

	injectEverythingLabels := maputil.MergeMap(
		testDynakube.OneAgent().GetNamespaceSelector().MatchLabels,
		testDynakube.MetadataEnrichment().GetNamespaceSelector().MatchLabels,
	)

	testCases := []testCase{
		{
			name: "control metadata-enrichment with annotations - deployment",
			app: sample.NewApp(t, &testDynakube,
				sample.WithName("deploy-metadata-annotation"),
				sample.AsDeployment(),
				sample.WithNamespaceLabels(injectEverythingLabels),
				sample.WithAnnotations(map[string]string{
					oacommon.AnnotationInject:   "false",
					metacommon.AnnotationInject: "true",
				})),
			assess: deploymentPodsHaveOnlyMetadataEnrichmentInitContainer,
		},
		{
			name: "control metadata-enrichment with annotations - pod",
			app: sample.NewApp(t, &testDynakube,
				sample.WithName("pod-metadata-annotation"),
				sample.WithNamespaceLabels(injectEverythingLabels),
				sample.WithAnnotations(map[string]string{
					oacommon.AnnotationInject:   "false",
					metacommon.AnnotationInject: "true",
				})),
			assess: podHasOnlyMetadataEnrichmentInitContainer,
		},
		{
			name: "control metadata-enrichment with namespace-selector - deployment",
			app: sample.NewApp(t, &testDynakube,
				sample.WithName("deploy-metadata-label"),
				sample.AsDeployment(),
				sample.WithNamespaceLabels(testDynakube.MetadataEnrichment().GetNamespaceSelector().MatchLabels),
			),
			assess: deploymentPodsHaveOnlyMetadataEnrichmentInitContainer,
		},
		{
			name: "control metadata-enrichment with namespace-selector - pod",
			app: sample.NewApp(t, &testDynakube,
				sample.WithName("pod-metadata-label"),
				sample.WithNamespaceLabels(testDynakube.MetadataEnrichment().GetNamespaceSelector().MatchLabels),
			),
			assess: podHasOnlyMetadataEnrichmentInitContainer,
		},
		{
			name: "control oneagent-injection with annotations (metadata enrichment will be enabled regardless of namespace selector/annotations) - pod",
			app: sample.NewApp(t, &testDynakube,
				sample.WithName("pod-oa-annotation"),
				sample.WithNamespaceLabels(injectEverythingLabels),
				sample.WithAnnotations(map[string]string{
					oacommon.AnnotationInject:   "true",
					metacommon.AnnotationInject: "false",
				})),
			assess: podHasCompleteInitContainer,
		},
		{
			name: "control oneagent-injection with namespace-selector (metadata enrichment will be enabled regardless of namespace selector/annotations) - pod",
			app: sample.NewApp(t, &testDynakube,
				sample.WithName("pod-oa-label"),
				sample.WithNamespaceLabels(testDynakube.OneAgent().GetNamespaceSelector().MatchLabels),
			),
			assess: podHasCompleteInitContainer,
		},
		{
			name: "namespace-selectors don't conflict - pod",
			app: sample.NewApp(t, &testDynakube,
				sample.WithName("pod-all-label"),
				sample.WithNamespaceLabels(injectEverythingLabels),
			),
			assess: podHasCompleteInitContainer,
		},
	}

	// dynakubeComponents install
	dynakubeComponents.Install(builder, helpers.LevelAssess, &secretConfig, testDynakube)

	// Register actual test
	for _, test := range testCases {
		builder.Assess(fmt.Sprintf("%s: Installing sample app", test.name), test.app.Install())
		builder.Assess(fmt.Sprintf("%s: Checking sample app", test.name), test.assess(test.app))
		builder.WithTeardown(fmt.Sprintf("%s: Uninstalling sample app", test.name), test.app.Uninstall())
	}

	dynakubeComponents.Delete(builder, helpers.LevelTeardown, testDynakube)

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
	enrichmentMetadata := metadataenrichment.GetMetadataFromPod(ctx, t, resource, testPod)

	assert.Equal(t, "pod", enrichmentMetadata.WorkloadKind)
	assert.Equal(t, testPod.Name, enrichmentMetadata.WorkloadName)
}

func deploymentPodsHaveOnlyMetadataEnrichmentInitContainer(sampleApp *sample.App) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		query := k8sdeployment.NewQuery(ctx, envConfig.Client().Resources(), client.ObjectKey{
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

// podHasCompleteInitContainer checks if the sample has BOTH the metadata-enrichment and oneagent parts added to it.
func podHasCompleteInitContainer(samplePod *sample.App) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		testPod := samplePod.GetPods(ctx, t, envConfig.Client().Resources()).Items[0]
		initContainers := testPod.Spec.InitContainers

		require.Len(t, initContainers, 1)

		assert.Contains(t, testPod.Annotations, metacommon.AnnotationWorkloadKind)
		assert.Contains(t, testPod.Annotations, metacommon.AnnotationWorkloadName)

		return ctx
	}
}

func assessDeploymentHasMetadataEnrichmentFile(ctx context.Context, t *testing.T, resource *resources.Resources, deploymentName string) k8sdeployment.PodConsumer {
	return func(pod corev1.Pod) {
		enrichmentMetadata := metadataenrichment.GetMetadataFromPod(ctx, t, resource, pod)

		assert.Equal(t, "deployment", enrichmentMetadata.WorkloadKind)
		assert.Equal(t, deploymentName, enrichmentMetadata.WorkloadName)
	}
}

func assessOnlyMetadataEnrichmentIsInjected(t *testing.T) k8sdeployment.PodConsumer {
	return func(pod corev1.Pod) {
		initContainers := pod.Spec.InitContainers

		require.Len(t, initContainers, 1)

		// The `--metadata-enrichment` is what turns on the feature in the init-container
		assert.Contains(t, initContainers[0].Args, "--"+bootstrapper.MetadataEnrichmentFlag)
		// The `--target=/mnt/bin` is a sign that the init-container will download/configure the oneagent
		assert.NotContains(t, initContainers[0].Args, "--"+bootstrapper.TargetFolderFlag+"="+consts.AgentInitBinDirMount)
		assert.Contains(t, pod.Annotations, metacommon.AnnotationWorkloadKind)
		assert.Contains(t, pod.Annotations, metacommon.AnnotationWorkloadName)
	}
}
