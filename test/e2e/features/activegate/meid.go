//go:build e2e

package activegate

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/settings"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers"
	dynakubeComponents "github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/tenant"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

// NoUpdateMEID checks that the Operator does not update the MEID we create.
// The creation part is not checked, to keep the test focused and not flaky.
// The dynakubes used should either create it, or just use the ME already there.
func NoUpdateMEID(t *testing.T) features.Feature {
	assessME := func(builder *features.FeatureBuilder, dk *dynakube.DynaKube, prev *settings.K8sClusterME) {
		builder.Assess("checking current ME id and name", func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
			dk := dk.DeepCopy()
			require.NoError(t, c.Client().Resources().Get(ctx, dk.Name, dk.Namespace, dk))
			prev.ID = dk.Status.KubernetesClusterMEID
			prev.Name = dk.Status.KubernetesClusterName
			require.NotEmpty(t, prev)

			return ctx
		})
	}

	reAssessME := func(builder *features.FeatureBuilder, dk *dynakube.DynaKube, prev *settings.K8sClusterME) {
		builder.Assess("checking current ME id and name against previous", func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
			dk := dk.DeepCopy()
			require.NoError(t, c.Client().Resources().Get(ctx, dk.Name, dk.Namespace, dk))
			current := &settings.K8sClusterME{ID: dk.Status.KubernetesClusterMEID, Name: dk.Status.KubernetesClusterName}
			require.Equal(t, prev, current)

			return ctx
		})
	}

	secretConfig := tenant.GetSingleTenantSecret(t)
	testDynakube := *dynakubeComponents.New(
		dynakubeComponents.WithActiveGate(),
		dynakubeComponents.WithAPIURL(secretConfig.APIURL))

	builder := features.New("meid-no-update")

	prevME := &settings.K8sClusterME{}
	dynakubeComponents.Install(builder, helpers.LevelAssess, &secretConfig, testDynakube)
	assessME(builder, &testDynakube, prevME)
	dynakubeComponents.Delete(builder, helpers.LevelAssess, testDynakube)

	dynakubeComponents.Install(builder, helpers.LevelAssess, &secretConfig, testDynakube)
	reAssessME(builder, &testDynakube, prevME)
	dynakubeComponents.Delete(builder, helpers.LevelAssess, testDynakube)

	return builder.Feature()
}
