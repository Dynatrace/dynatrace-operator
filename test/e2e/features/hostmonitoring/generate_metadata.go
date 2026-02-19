//go:build e2e

package hostmonitoring

import (
	"context"
	"strings"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/activegate"
	componentDynakube "github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubernetes/objects/k8sdaemonset"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubernetes/objects/k8spod"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/shell"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/tenant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	generateMetadataFile = "/var/lib/dynatrace/enrichment/dt_node_metadata.properties"
)

var expectedMetadataFields = []string{
	"k8s.cluster.name",
	"k8s.cluster.uid",
	"k8s.node.name",
	"dt.entity.kubernetes_cluster",
}

func GenerateMetadata(t *testing.T) features.Feature {
	builder := features.New("generate-metadata")

	secretConfig := tenant.GetSingleTenantSecret(t)

	options := []componentDynakube.Option{
		componentDynakube.WithAPIURL(secretConfig.APIURL),
		componentDynakube.WithActiveGate(),
		componentDynakube.WithHostMonitoringSpec(&oneagent.HostInjectSpec{}),
	}
	testDynakube := *componentDynakube.New(options...)

	// Register Dynakube install
	componentDynakube.Install(builder, helpers.LevelAssess, &secretConfig, testDynakube)
	builder.Assess("OneAgent started", k8sdaemonset.IsReady(testDynakube.OneAgent().GetDaemonsetName(), testDynakube.Namespace))
	builder.Assess("active gate pod is running", activegate.CheckContainer(&testDynakube))

	builder.Assess("Checking if all OneAgent pods have generated metadata", oneAgentHaveGeneratedMetadata(testDynakube))

	// Register sample, dynakube and operator uninstall
	componentDynakube.Delete(builder, helpers.LevelTeardown, testDynakube)
	builder.WithTeardown("Deleted tenant secret", tenant.DeleteTenantSecret(testDynakube.Name, testDynakube.Namespace))

	return builder.Feature()
}

func oneAgentHaveGeneratedMetadata(dk dynakube.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		r := envConfig.Client().Resources()

		q := k8sdaemonset.NewQuery(ctx, envConfig.Client().Resources(), client.ObjectKey{
			Name:      dk.OneAgent().GetDaemonsetName(),
			Namespace: dk.Namespace,
		})

		err := q.ForEachPod(assertGeneratedMetadataFields(ctx, t, r))
		require.NoError(t, err)

		return ctx
	}
}

func assertGeneratedMetadataFields(ctx context.Context, t *testing.T, resource *resources.Resources) k8sdaemonset.PodConsumer {
	return func(pod corev1.Pod) {
		generatedMetadata := getGeneratedMetadataFromPod(ctx, t, resource, pod)
		assert.NotEmpty(t, generatedMetadata, "generated metadata should not be empty")
		for _, attribute := range expectedMetadataFields {
			assert.Containsf(t, generatedMetadata, attribute, "generated metadata should contain %s attribute", attribute)
			assert.NotEmptyf(t, generatedMetadata[attribute], "generated metadata %s attribute should not be empty", attribute)
		}
	}
}

func getGeneratedMetadataFromPod(ctx context.Context, t *testing.T, resource *resources.Resources, oaPod corev1.Pod) map[string]string {
	readGeneratedMetadataCmd := shell.ReadFile(generateMetadataFile)
	require.NotEmpty(t, oaPod.Spec.Containers, "OneAgent pod should have at least one container")
	container := oaPod.Spec.Containers[0].Name
	result, err := k8spod.Exec(ctx, resource, oaPod, container, readGeneratedMetadataCmd...)

	require.NoError(t, err)

	assert.Zero(t, result.StdErr.Len())
	assert.NotEmpty(t, result.StdOut)

	return parseGeneratedMetadata(result.StdOut.String())
}

func parseGeneratedMetadata(text string) map[string]string {
	var m = make(map[string]string)

	for line := range strings.Lines(text) {
		l := strings.TrimSpace(line)
		if l == "" {
			continue
		}

		key, value, found := strings.Cut(l, "=")
		if !found {
			continue
		}

		m[key] = value
	}

	return m
}
