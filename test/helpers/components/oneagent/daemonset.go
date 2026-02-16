//go:build e2e

package oneagent

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubernetes/objects/k8sdaemonset"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubernetes/objects/k8spod"
	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func WaitForDaemonset(dsName, namespace string) features.Func {
	return helpers.ToFeatureFunc(k8sdaemonset.WaitFor(dsName, namespace), true)
}

func WaitForDaemonSetPodsDeletion(dsName, namespace string) features.Func {
	return k8spod.WaitForDeletionWithOwner(dsName, namespace)
}

func Get(ctx context.Context, resource *resources.Resources, dk dynakube.DynaKube) (appsv1.DaemonSet, error) {
	return k8sdaemonset.NewQuery(ctx, resource, client.ObjectKey{
		Name:      dk.OneAgent().GetDaemonsetName(),
		Namespace: dk.Namespace,
	}).Get()
}
