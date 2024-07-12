//go:build e2e

package activegate

import (
	"context"
	"fmt"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta2/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/pod"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/statefulset"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/logs"
	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
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

func Get(ctx context.Context, resource *resources.Resources, dk dynakube.DynaKube) (appsv1.StatefulSet, error) {
	return statefulset.NewQuery(ctx, resource, client.ObjectKey{
		Name:      GetActiveGateStateFulSetName(&dk, "activegate"),
		Namespace: dk.Namespace,
	}).Get()
}

func WaitForStatefulSetPodsDeletion(dk *dynakube.DynaKube, component string) features.Func {
	return pod.WaitForPodsDeletionWithOwner(GetActiveGateStateFulSetName(dk, component), dk.Namespace)
}
