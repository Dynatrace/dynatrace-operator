//go:build e2e

package activegate

import (
	"context"
	"fmt"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/logs"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func GetActiveGateStateFulSetName(dk *dynakube.DynaKube, component string) string {
	return fmt.Sprintf("%s-%s", dk.Name, component)
}

func GetActiveGatePodName(dk *dynakube.DynaKube, component string) string {
	return fmt.Sprintf("%s-0", GetActiveGateStateFulSetName(dk, component))
}

func ReadActiveGateLog(ctx context.Context, t *testing.T, envConfig *envconf.Config, dk *dynakube.DynaKube, component string) string {
	return logs.ReadLog(ctx, t, envConfig, dk.Namespace, GetActiveGatePodName(dk, component), consts.ActiveGateContainerName)
}

func CheckContainer(dk *dynakube.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resources := envConfig.Client().Resources()

		var activeGatePod corev1.Pod
		require.NoError(t, resources.WithNamespace(dk.Namespace).Get(ctx, GetActiveGatePodName(dk, "activegate"), dk.Namespace, &activeGatePod))

		require.NotNil(t, activeGatePod.Spec)
		require.NotEmpty(t, activeGatePod.Spec.Containers)

		return ctx
	}
}
