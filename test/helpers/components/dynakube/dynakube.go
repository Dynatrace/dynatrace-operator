//go:build e2e

package dynakube

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1"
	dynakubev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	dynatracev1beta2 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta2"
	dynakubev1beta2 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta2/dynakube"
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

func Install(builder *features.FeatureBuilder, level features.Level, secretConfig *tenant.Secret, testDynakube dynakubev1beta2.DynaKube) {
	Create(builder, level, secretConfig, testDynakube)
	VerifyStartup(builder, level, testDynakube)
}

func InstallPreviousVersion(builder *features.FeatureBuilder, level features.Level, secretConfig *tenant.Secret, previousVersionDK dynakubev1beta1.DynaKube) {
	CreatePreviousVersion(builder, level, secretConfig, previousVersionDK)
	VerifyStartupPreviousVersion(builder, level, previousVersionDK)
}

func Create(builder *features.FeatureBuilder, level features.Level, secretConfig *tenant.Secret, testDynakube dynakubev1beta2.DynaKube) {
	if secretConfig != nil {
		builder.WithStep("created tenant secret", level, tenant.CreateTenantSecret(*secretConfig, testDynakube.Name, testDynakube.Namespace))
	}
	builder.WithStep(
		fmt.Sprintf("'%s' dynakube created", testDynakube.Name),
		level,
		create(testDynakube))
}

func Update(builder *features.FeatureBuilder, level features.Level, testDynakube dynakubev1beta2.DynaKube) {
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

func Delete(builder *features.FeatureBuilder, level features.Level, testDynakube dynakubev1beta2.DynaKube) {
	builder.WithStep("dynakube deleted", level, remove(testDynakube))
	if testDynakube.NeedsOneAgent() {
		builder.WithStep("oneagent pods stopped", level, oneagent.WaitForDaemonSetPodsDeletion(testDynakube))
	}
	if testDynakube.ClassicFullStackMode() {
		oneagent.RunClassicUninstall(builder, level, testDynakube)
	}
}

func VerifyStartup(builder *features.FeatureBuilder, level features.Level, testDynakube dynakubev1beta2.DynaKube) {
	if testDynakube.NeedsOneAgent() {
		builder.WithStep("oneagent started", level, oneagent.WaitForDaemonset(testDynakube))
	}
	builder.WithStep(
		fmt.Sprintf("'%s' dynakube phase changes to 'Running'", testDynakube.Name),
		level,
		WaitForPhase(testDynakube, status.Running))
}

func WaitForPhase(dynakube dynakubev1beta2.DynaKube, phase status.DeploymentPhase) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resources := envConfig.Client().Resources()

		const timeout = 5 * time.Minute
		err := wait.For(conditions.New(resources).ResourceMatch(&dynakube, func(object k8s.Object) bool {
			dynakube, isDynakube := object.(*dynakubev1beta2.DynaKube)

			return isDynakube && dynakube.Status.Phase == phase
		}), wait.WithTimeout(timeout))

		require.NoError(t, err)

		return ctx
	}
}

func WaitForPhasePreviousVersion(dynakube dynakubev1beta1.DynaKube, phase status.DeploymentPhase) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resources := envConfig.Client().Resources()

		const timeout = 5 * time.Minute
		err := wait.For(conditions.New(resources).ResourceMatch(&dynakube, func(object k8s.Object) bool {
			dynakube, isDynakube := object.(*dynakubev1beta1.DynaKube)

			return isDynakube && dynakube.Status.Phase == phase
		}), wait.WithTimeout(timeout))

		require.NoError(t, err)

		return ctx
	}
}

func create(dynakube dynakubev1beta2.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		require.NoError(t, dynatracev1beta2.AddToScheme(envConfig.Client().Resources().GetScheme()))
		require.NoError(t, envConfig.Client().Resources().Create(ctx, &dynakube))

		return ctx
	}
}

func createPreviousVersion(dynakube dynakubev1beta1.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		require.NoError(t, dynatracev1beta1.AddToScheme(envConfig.Client().Resources().GetScheme()))
		require.NoError(t, envConfig.Client().Resources().Create(ctx, &dynakube))

		return ctx
	}
}

func update(dynakube dynakubev1beta2.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		require.NoError(t, dynatracev1beta2.AddToScheme(envConfig.Client().Resources().GetScheme()))
		var dk dynakubev1beta2.DynaKube
		require.NoError(t, envConfig.Client().Resources().Get(ctx, dynakube.Name, dynakube.Namespace, &dk))
		dynakube.ResourceVersion = dk.ResourceVersion
		require.NoError(t, envConfig.Client().Resources().Update(ctx, &dynakube))

		return ctx
	}
}

func remove(dynakube dynakubev1beta2.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resources := envConfig.Client().Resources()

		err := dynatracev1beta2.AddToScheme(resources.GetScheme())
		require.NoError(t, err)

		err = resources.Delete(ctx, &dynakube)
		isNoKindMatchErr := meta.IsNoMatchError(err)

		if err != nil {
			if k8serrors.IsNotFound(err) || isNoKindMatchErr {
				// If the dynakube itself or the crd does not exist, everything is fine
				err = nil
			}
			require.NoError(t, err)
		}

		err = wait.For(conditions.New(resources).ResourceDeleted(&dynakube), wait.WithTimeout(1*time.Minute))
		require.NoError(t, err)

		return ctx
	}
}
