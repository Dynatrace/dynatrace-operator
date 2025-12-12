package eec

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8sconditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8sstatefulset"
	"github.com/pkg/errors"
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
	if ext := r.dk.Extensions(); !ext.IsAnyEnabled() {
		if meta.FindStatusCondition(*r.dk.Conditions(), extensionControllerStatefulSetConditionType) == nil {
			return nil
		}
		defer meta.RemoveStatusCondition(r.dk.Conditions(), extensionControllerStatefulSetConditionType)

		sts, err := k8sstatefulset.Build(r.dk, ext.GetExecutionControllerStatefulsetName(), corev1.Container{})
		if err != nil {
			log.Error(err, "could not build "+ext.GetExecutionControllerStatefulsetName()+" during cleanup")

			return err
		}

		err = k8sstatefulset.Query(r.client, r.apiReader, log).Delete(ctx, sts)
		if err != nil {
			log.Error(err, "failed to clean up "+ext.GetExecutionControllerStatefulsetName()+" statufulset")

			return nil
		}

		return nil
	}

	if r.dk.Status.ActiveGate.ConnectionInfo.TenantUUID == "" {
		k8sconditions.SetStatefulSetOutdated(r.dk.Conditions(), extensionControllerStatefulSetConditionType, r.dk.Extensions().GetExecutionControllerStatefulsetName())

		return errors.New("tenantUUID unknown")
	}

	if r.dk.Status.KubeSystemUUID == "" {
		k8sconditions.SetStatefulSetOutdated(r.dk.Conditions(), extensionControllerStatefulSetConditionType, r.dk.Extensions().GetExecutionControllerStatefulsetName())

		return errors.New("kubeSystemUUID unknown")
	}

	return r.createOrUpdateStatefulset(ctx)
}
