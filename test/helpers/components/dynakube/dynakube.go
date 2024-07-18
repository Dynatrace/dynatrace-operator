//go:build e2e

package dynakube

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1"
	dynakubev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube" //nolint:staticcheck
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta2"
	dynakubev1beta3 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
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

func Install(builder *features.FeatureBuilder, level features.Level, secretConfig *tenant.Secret, testDynakube dynakubev1beta3.DynaKube) {
	Create(builder, level, secretConfig, testDynakube)
	VerifyStartup(builder, level, testDynakube)
}

func InstallPreviousVersion(builder *features.FeatureBuilder, level features.Level, secretConfig *tenant.Secret, previousVersionDK dynakubev1beta1.DynaKube) {
	CreatePreviousVersion(builder, level, secretConfig, previousVersionDK)
	VerifyStartupPreviousVersion(builder, level, previousVersionDK)
}

func Create(builder *features.FeatureBuilder, level features.Level, secretConfig *tenant.Secret, testDynakube dynakubev1beta3.DynaKube) {
	if secretConfig != nil {
		builder.WithStep("created tenant secret", level, tenant.CreateTenantSecret(*secretConfig, testDynakube.Name, testDynakube.Namespace))
	}
	builder.WithStep(
		fmt.Sprintf("'%s' dynakube created", testDynakube.Name),
		level,
		create(testDynakube))
}

func Update(builder *features.FeatureBuilder, level features.Level, testDynakube dynakubev1beta3.DynaKube) {
	builder.WithStep("dynakube updated", level, update(testDynakube))
}

func CreatePreviousVersion(builder *features.FeatureBuilder, level features.Level, secretConfig *tenant.Secret, testDynakube dynakubev1beta1.DynaKube) {
	if secretConfig != nil {
		builder.WithStep("created tenant secret", level, tenant.CreateTenantSecret(*secretConfig, testDynakube.Name, testDynakube.Namespace))
	}
	builder.WithStep(
		fmt.Sprintf("'%s' dynakube created", testDynakube.Name),
		level,
		createPreviousVersion(testDynakube))
}

func VerifyStartupPreviousVersion(builder *features.FeatureBuilder, level features.Level, testDynakube dynakubev1beta1.DynaKube) {
	if testDynakube.NeedsOneAgent() {
		builder.WithStep("oneagent started", level, oneagent.WaitForDaemonsetV1Beta1(testDynakube))
	}
	builder.WithStep(
		fmt.Sprintf("'%s' dynakube phase changes to 'Running'", testDynakube.Name),
		level,
		WaitForPhasePreviousVersion(testDynakube, status.Running))
}

func Delete(builder *features.FeatureBuilder, level features.Level, testDynakube dynakubev1beta3.DynaKube) {
	builder.WithStep("dynakube deleted", level, remove(testDynakube))
	if testDynakube.NeedsOneAgent() {
		builder.WithStep("oneagent pods stopped", level, oneagent.WaitForDaemonSetPodsDeletion(testDynakube))
	}
	if testDynakube.ClassicFullStackMode() {
		oneagent.RunClassicUninstall(builder, level, testDynakube)
	}
}

func VerifyStartup(builder *features.FeatureBuilder, level features.Level, testDynakube dynakubev1beta3.DynaKube) {
	if testDynakube.NeedsOneAgent() {
		builder.WithStep("oneagent started", level, oneagent.WaitForDaemonset(testDynakube))
	}
	builder.WithStep(
		fmt.Sprintf("'%s' dynakube phase changes to 'Running'", testDynakube.Name),
		level,
		WaitForPhase(testDynakube, status.Running))
}

func WaitForPhase(dk dynakubev1beta3.DynaKube, phase status.DeploymentPhase) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resources := envConfig.Client().Resources()

		const timeout = 5 * time.Minute
		err := wait.For(conditions.New(resources).ResourceMatch(&dk, func(object k8s.Object) bool {
			dynakube, isDynakube := object.(*dynakubev1beta3.DynaKube)

			return isDynakube && dynakube.Status.Phase == phase
		}), wait.WithTimeout(timeout))

		require.NoError(t, err)

		return ctx
	}
}

func WaitForPhasePreviousVersion(dk dynakubev1beta1.DynaKube, phase status.DeploymentPhase) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resources := envConfig.Client().Resources()

		const timeout = 5 * time.Minute
		err := wait.For(conditions.New(resources).ResourceMatch(&dk, func(object k8s.Object) bool {
			dynakube, isDynakube := object.(*dynakubev1beta1.DynaKube)

			return isDynakube && dynakube.Status.Phase == phase
		}), wait.WithTimeout(timeout))

		require.NoError(t, err)

		return ctx
	}
}

func create(dk dynakubev1beta3.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		require.NoError(t, v1beta2.AddToScheme(envConfig.Client().Resources().GetScheme()))
		require.NoError(t, envConfig.Client().Resources().Create(ctx, &dk))

		return ctx
	}
}

func createPreviousVersion(dk dynakubev1beta1.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		require.NoError(t, v1beta1.AddToScheme(envConfig.Client().Resources().GetScheme()))
		require.NoError(t, envConfig.Client().Resources().Create(ctx, &dk))

		return ctx
	}
}

func update(dk dynakubev1beta3.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		require.NoError(t, v1beta2.AddToScheme(envConfig.Client().Resources().GetScheme()))
		var oldDK dynakubev1beta3.DynaKube
		require.NoError(t, envConfig.Client().Resources().Get(ctx, oldDK.Name, oldDK.Namespace, &oldDK))
		oldDK.ResourceVersion = dk.ResourceVersion
		require.NoError(t, envConfig.Client().Resources().Update(ctx, &oldDK))

		return ctx
	}
}

func remove(dk dynakubev1beta3.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resources := envConfig.Client().Resources()

		err := v1beta2.AddToScheme(resources.GetScheme())
		require.NoError(t, err)

		err = resources.Delete(ctx, &dk)
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
