//go:build e2e

package oneagent

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubeobjects/daemonset"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubeobjects/pod"
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

func Get(ctx context.Context, resource *resources.Resources, dk dynakube.DynaKube) (appsv1.DaemonSet, error) {
	return daemonset.NewQuery(ctx, resource, client.ObjectKey{
		Name:      dk.OneAgent().GetDaemonsetName(),
		Namespace: dk.Namespace,
	}).Get()
}
