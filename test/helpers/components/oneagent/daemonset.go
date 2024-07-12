//go:build e2e

package oneagent

import (
	"context"

	dynakubev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	dynakubev1beta2 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta2/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/daemonset"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/pod"
	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func WaitForDaemonset(dynakube dynakubev1beta2.DynaKube) features.Func {
	return helpers.ToFeatureFunc(daemonset.WaitFor(dynakube.OneAgentDaemonsetName(), dynakube.Namespace), true)
}

func WaitForDaemonsetV1Beta1(dynakube dynakubev1beta1.DynaKube) features.Func {
	return helpers.ToFeatureFunc(daemonset.WaitFor(dynakube.OneAgentDaemonsetName(), dynakube.Namespace), true)
}

func WaitForDaemonSetPodsDeletion(dynakube dynakubev1beta2.DynaKube) features.Func {
	return pod.WaitForPodsDeletionWithOwner(dynakube.OneAgentDaemonsetName(), dynakube.Namespace)
}

func Get(ctx context.Context, resource *resources.Resources, dynakube dynakubev1beta2.DynaKube) (appsv1.DaemonSet, error) {
	return daemonset.NewQuery(ctx, resource, client.ObjectKey{
		Name:      dynakube.OneAgentDaemonsetName(),
		Namespace: dynakube.Namespace,
	}).Get()
}
