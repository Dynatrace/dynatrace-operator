//go:build e2e

package dynakube

import (
	"context"
	"strconv"
	"strings"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/api/status"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	dynakubev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1/dynakube"
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

func Create(dynakube dynakubev1beta1.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		require.NoError(t, dynatracev1beta1.AddToScheme(envConfig.Client().Resources().GetScheme()))
		require.NoError(t, envConfig.Client().Resources().Create(ctx, &dynakube))
		return ctx
	}
}

func Update(dynakube dynakubev1beta1.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		require.NoError(t, dynatracev1beta1.AddToScheme(envConfig.Client().Resources().GetScheme()))
		var dk dynakubev1beta1.DynaKube
		require.NoError(t, envConfig.Client().Resources().Get(ctx, dynakube.Name, dynakube.Namespace, &dk))
		dynakube.ResourceVersion = dk.ResourceVersion
		require.NoError(t, envConfig.Client().Resources().Update(ctx, &dynakube))
		return ctx
	}
}

func Delete(dynakube dynakubev1beta1.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resources := envConfig.Client().Resources()

		err := dynatracev1beta1.AddToScheme(resources.GetScheme())
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

		err = wait.For(conditions.New(resources).ResourceDeleted(&dynakube))
		require.NoError(t, err)
		return ctx
	}
}

func WaitForDynakubePhase(dynakube dynakubev1beta1.DynaKube, phase status.DeploymentPhase) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resources := envConfig.Client().Resources()

		err := wait.For(conditions.New(resources).ResourceMatch(&dynakube, func(object k8s.Object) bool {
			dynakube, isDynakube := object.(*dynakubev1beta1.DynaKube)
			return isDynakube && dynakube.Status.Phase == phase
		}))

		require.NoError(t, err)

		return ctx
	}
}

func SyntheticLocationOrdinal(dynakube dynakubev1beta1.DynaKube) uint64 {
	const defaultOrd = uint64(0)
	_, suffix, found := strings.Cut(dynakube.FeatureSyntheticLocationEntityId(), "-")
	if !found {
		return defaultOrd
	}

	parsed, err := strconv.ParseUint(suffix, 16, 64)
	if err != nil {
		return defaultOrd
	}

	return parsed
}
