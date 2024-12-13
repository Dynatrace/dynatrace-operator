//go:build e2e

package oneagent

import (
	"context"

	dynakubev1beta3 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/daemonset"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/pod"
	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func WaitForDaemonset(dsName, namespace string) features.Func {
	return helpers.ToFeatureFunc(daemonset.WaitFor(dsName, namespace), true)
}

func WaitForDaemonSetPodsDeletion(dsName, namespace string) features.Func {
	return pod.WaitForPodsDeletionWithOwner(dsName, namespace)
}

func Get(ctx context.Context, resource *resources.Resources, dk dynakubev1beta3.DynaKube) (appsv1.DaemonSet, error) {
	return daemonset.NewQuery(ctx, resource, client.ObjectKey{
		Name:      dk.OneAgent().OneAgentDaemonsetName(),
		Namespace: dk.Namespace,
	}).Get()
}
