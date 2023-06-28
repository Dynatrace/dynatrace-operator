//go:build e2e

package activegate

import (
	"context"
	"fmt"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/statefulset"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/logs"
	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func WaitForStatefulSet(testDynakube *dynatracev1beta1.DynaKube, component string) features.Func {
	return statefulset.WaitFor(GetActiveGateStateFulSetName(testDynakube, component), testDynakube.Namespace)
}

func GetActiveGateStateFulSetName(testDynakube *dynatracev1beta1.DynaKube, component string) string {
	return fmt.Sprintf("%s-%s", testDynakube.Name, component)
}

func GetActiveGatePodName(testDynakube *dynatracev1beta1.DynaKube, component string) string {
	return fmt.Sprintf("%s-0", GetActiveGateStateFulSetName(testDynakube, component))
}

func ReadActiveGateLog(ctx context.Context, t *testing.T, environmentConfig *envconf.Config, testDynakube *dynatracev1beta1.DynaKube, component string) string {
	return logs.ReadLog(ctx, t, environmentConfig, testDynakube.Namespace, GetActiveGatePodName(testDynakube, component), consts.ActiveGateContainerName)
}

func Get(ctx context.Context, resource *resources.Resources, dynakube dynatracev1beta1.DynaKube) (appsv1.StatefulSet, error) {
	return statefulset.NewQuery(ctx, resource, client.ObjectKey{
		Name:      GetActiveGateStateFulSetName(&dynakube, "activegate"),
		Namespace: dynakube.Namespace,
	}).Get()
}
