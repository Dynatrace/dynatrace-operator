//go:build e2e

package activegate

import (
	"context"
	"fmt"
	"testing"

	dynatracev1 "github.com/Dynatrace/dynatrace-operator/src/api/v1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/statefulset"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/logs"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func WaitForStatefulSet(testDynakube *dynatracev1.DynaKube, component string) features.Func {
	return statefulset.WaitFor(GetActiveGateStateFulSetName(testDynakube, component), testDynakube.Namespace)
}

func GetActiveGateStateFulSetName(testDynakube *dynatracev1.DynaKube, component string) string {
	return fmt.Sprintf("%s-%s", testDynakube.Name, component)
}

func GetActiveGatePodName(testDynakube *dynatracev1.DynaKube, component string) string {
	return fmt.Sprintf("%s-0", GetActiveGateStateFulSetName(testDynakube, component))
}

func ReadActiveGateLog(ctx context.Context, t *testing.T, environmentConfig *envconf.Config, testDynakube *dynatracev1.DynaKube, component string) string {
	return logs.ReadLog(ctx, t, environmentConfig, testDynakube.Namespace, GetActiveGatePodName(testDynakube, component), consts.ActiveGateContainerName)
}
