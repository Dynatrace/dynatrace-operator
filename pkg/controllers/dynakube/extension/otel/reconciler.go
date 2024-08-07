package otel

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/statefulset"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type reconciler struct {
	client    client.Client
	apiReader client.Reader

	dk *dynakube.DynaKube
}

type ReconcilerBuilder func(clt client.Client, apiReader client.Reader, dk *dynakube.DynaKube) controllers.Reconciler

var _ ReconcilerBuilder = NewReconciler

func NewReconciler(clt client.Client, apiReader client.Reader, dk *dynakube.DynaKube) controllers.Reconciler {
	return &reconciler{
		client:    clt,
		apiReader: apiReader,
		dk:        dk,
	}
}

func (r *reconciler) Reconcile(ctx context.Context) error {
	if !r.dk.PrometheusEnabled() {
		if meta.FindStatusCondition(*r.dk.Conditions(), otelControllerStatefulSetConditionType) == nil {
			return nil
		}
		defer meta.RemoveStatusCondition(r.dk.Conditions(), otelControllerStatefulSetConditionType)

		sts, err := statefulset.Build(r.dk, dynakube.ExtensionsCollectorStatefulsetName, corev1.Container{})
		if err != nil {
			log.Error(err, "could not build "+dynakube.ExtensionsCollectorStatefulsetName+" during cleanup")

			return err
		}

		err = statefulset.Query(r.client, r.apiReader, log).Delete(ctx, sts)

		if err != nil {
			log.Error(err, "failed to clean up "+dynakube.ExtensionsCollectorStatefulsetName+" statufulset")

			return nil
		}

		return nil
	}

	return r.createOrUpdateStatefulset(ctx)
}
