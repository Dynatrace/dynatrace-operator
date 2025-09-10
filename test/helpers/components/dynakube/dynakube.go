//go:build e2e

package dynakube

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	prevDynakube "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta5/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/oneagent"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/tenant"
	"github.com/stretchr/testify/require"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	defaultName = "dynakube"
)

func Install(builder *features.FeatureBuilder, level features.Level, secretConfig *tenant.Secret, dk dynakube.DynaKube) {
	Create(builder, level, secretConfig.APIToken, secretConfig.DataIngestToken, dk)
	VerifyStartup(builder, level, dk)
}

func InstallWithoutSettingsScopes(builder *features.FeatureBuilder, level features.Level, secretConfig *tenant.Secret, dk dynakube.DynaKube) {
	Create(builder, level, secretConfig.APITokenNoSettings, secretConfig.DataIngestToken, dk)
	VerifyStartup(builder, level, dk)
}

func InstallPreviousVersion(builder *features.FeatureBuilder, level features.Level, secretConfig *tenant.Secret, prevDk prevDynakube.DynaKube) {
	CreatePreviousVersion(builder, level, secretConfig.APIToken, secretConfig.DataIngestToken, prevDk)
	VerifyStartupPreviousVersion(builder, level, prevDk)
}

func Create(builder *features.FeatureBuilder, level features.Level, apiToken, dataIngestToken string, testDynakube dynakube.DynaKube) {
	if apiToken != "" || dataIngestToken != "" {
		builder.WithStep("created tenant secret", level, tenant.CreateTenantSecret(apiToken, dataIngestToken, testDynakube.Name, testDynakube.Namespace))
	}
	builder.WithStep(
		fmt.Sprintf("'%s' dynakube created", testDynakube.Name),
		level,
		create(testDynakube))
}

func Update(builder *features.FeatureBuilder, level features.Level, testDynakube dynakube.DynaKube) {
	builder.WithStep("dynakube updated", level, update(testDynakube))
}

func CreatePreviousVersion(builder *features.FeatureBuilder, level features.Level, apiToken, dataIngestToken string, prevDk prevDynakube.DynaKube) {
	if apiToken != "" || dataIngestToken != "" {
		builder.WithStep("created tenant secret", level, tenant.CreateTenantSecret(apiToken, dataIngestToken, prevDk.Name, prevDk.Namespace))
	}
	builder.WithStep(
		fmt.Sprintf("'%s' dynakube created", prevDk.Name),
		level,
		createPreviousVersion(prevDk))
}

func VerifyStartupPreviousVersion(builder *features.FeatureBuilder, level features.Level, prevDk prevDynakube.DynaKube) {
	if prevDk.OneAgent().IsDaemonsetRequired() {
		builder.WithStep("oneagent started", level, oneagent.WaitForDaemonset(prevDk.OneAgent().GetDaemonsetName(), prevDk.Namespace))
	}
	builder.WithStep(
		fmt.Sprintf("'%s' dynakube phase changes to 'Running'", prevDk.Name),
		level,
		WaitForPhasePreviousVersion(prevDk, status.Running))
}

func Delete(builder *features.FeatureBuilder, level features.Level, dk dynakube.DynaKube) {
	builder.WithStep("dynakube deleted", level, remove(dk))
	if dk.OneAgent().IsDaemonsetRequired() {
		builder.WithStep("oneagent pods stopped", level, oneagent.WaitForDaemonSetPodsDeletion(dk.OneAgent().GetDaemonsetName(), dk.Namespace))
	}
	if dk.OneAgent().IsClassicFullStackMode() {
		oneagent.RunClassicUninstall(builder, level, dk)
	}
}

func VerifyStartup(builder *features.FeatureBuilder, level features.Level, dk dynakube.DynaKube) {
	if dk.OneAgent().IsDaemonsetRequired() {
		builder.WithStep("oneagent started", level, oneagent.WaitForDaemonset(dk.OneAgent().GetDaemonsetName(), dk.Namespace))
	}
	builder.WithStep(
		fmt.Sprintf("'%s' dynakube phase changes to 'Running'", dk.Name),
		level,
		WaitForPhase(dk, status.Running))
}

func WaitForPhase(dk dynakube.DynaKube, phase status.DeploymentPhase) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resources := envConfig.Client().Resources()

		const timeout = 8 * time.Minute
		err := wait.For(conditions.New(resources).ResourceMatch(&dk, func(object k8s.Object) bool {
			dynakube, isDynakube := object.(*dynakube.DynaKube)

			return isDynakube && dynakube.Status.Phase == phase
		}), wait.WithTimeout(timeout))

		require.NoError(t, err)

		return ctx
	}
}

func WaitForPhasePreviousVersion(dk prevDynakube.DynaKube, phase status.DeploymentPhase) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resources := envConfig.Client().Resources()

		const timeout = 5 * time.Minute
		err := wait.For(conditions.New(resources).ResourceMatch(&dk, func(object k8s.Object) bool {
			dynakube, isDynakube := object.(*prevDynakube.DynaKube)

			return isDynakube && dynakube.Status.Phase == phase
		}), wait.WithTimeout(timeout))

		require.NoError(t, err)

		return ctx
	}
}

func create(dk dynakube.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		require.NoError(t, envConfig.Client().Resources().Create(ctx, &dk))

		return ctx
	}
}

func createPreviousVersion(dk prevDynakube.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		require.NoError(t, envConfig.Client().Resources().Create(ctx, &dk))

		return ctx
	}
}

func update(dk dynakube.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		var oldDK dynakube.DynaKube
		require.NoError(t, envConfig.Client().Resources().Get(ctx, dk.Name, dk.Namespace, &oldDK))
		dk.ResourceVersion = oldDK.ResourceVersion
		require.NoError(t, envConfig.Client().Resources().Update(ctx, &dk))

		return ctx
	}
}

func remove(dk dynakube.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resources := envConfig.Client().Resources()

		err := resources.Delete(ctx, &dk)
		isNoKindMatchErr := meta.IsNoMatchError(err)

		if err != nil {
			if k8serrors.IsNotFound(err) || isNoKindMatchErr {
				// If the dynakube itself or the crd does not exist, everything is fine
				err = nil
			}
			require.NoError(t, err)
		}

		err = wait.For(conditions.New(resources).ResourceDeleted(&dk), wait.WithTimeout(1*time.Minute))
		require.NoError(t, err)

		return ctx
	}
}
