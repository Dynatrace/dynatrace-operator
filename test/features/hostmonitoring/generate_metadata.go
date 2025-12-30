//go:build e2e

package hostmonitoring

import (
	"bufio"
	"context"
	"strings"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/test/helpers"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/activegate"
	componentDynakube "github.com/Dynatrace/dynatrace-operator/test/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/daemonset"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/pod"
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
	builder.Assess("OneAgent started", daemonset.IsReady(testDynakube.OneAgent().GetDaemonsetName(), testDynakube.Namespace))
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

		q := daemonset.NewQuery(ctx, envConfig.Client().Resources(), client.ObjectKey{
			Name:      dk.OneAgent().GetDaemonsetName(),
			Namespace: dk.Namespace,
		})

		err := q.ForEachPod(assertGeneratedMetadataFields(ctx, t, r))
		require.NoError(t, err)

		return ctx
	}
}

func assertGeneratedMetadataFields(ctx context.Context, t *testing.T, resource *resources.Resources) daemonset.PodConsumer {
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
	result, err := pod.Exec(ctx, resource, oaPod, container, readGeneratedMetadataCmd...)

	require.NoError(t, err)

	assert.Zero(t, result.StdErr.Len())
	assert.NotEmpty(t, result.StdOut)

	// fmt.Printf("generated metadata: \n%s", result.StdOut.String())
	return parseGeneratedMetadata(result.StdOut.String())
}

func parseGeneratedMetadata(text string) map[string]string {
	numColumns := 2
	var m = make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(text))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, "=", numColumns)
		if len(parts) != numColumns {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		m[key] = value
	}

	return m
}
