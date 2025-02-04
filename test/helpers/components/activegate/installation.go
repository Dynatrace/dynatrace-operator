//go:build e2e

package activegate

import (
	"context"
	"fmt"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/statefulset"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/logs"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func WaitForStatefulSet(dk *dynakube.DynaKube, component string) features.Func {
	return statefulset.WaitFor(GetActiveGateStateFulSetName(dk, component), dk.Namespace)
}

func GetActiveGateStateFulSetName(dk *dynakube.DynaKube, component string) string {
	return fmt.Sprintf("%s-%s", dk.Name, component)
}

func GetActiveGatePodName(dk *dynakube.DynaKube, component string) string {
	return fmt.Sprintf("%s-0", GetActiveGateStateFulSetName(dk, component))
}

func ReadActiveGateLog(ctx context.Context, t *testing.T, envConfig *envconf.Config, dk *dynakube.DynaKube, component string) string {
	return logs.ReadLog(ctx, t, envConfig, dk.Namespace, GetActiveGatePodName(dk, component), consts.ActiveGateContainerName)
}
