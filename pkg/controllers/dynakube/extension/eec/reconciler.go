package eec

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8sconditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8sstatefulset"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Reconciler struct {
	client    client.Client
	apiReader client.Reader
}

func NewReconciler(clt client.Client, apiReader client.Reader) *Reconciler {
	return &Reconciler{
		client:    clt,
		apiReader: apiReader,
	}
}

func (r *Reconciler) Reconcile(ctx context.Context, dk *dynakube.DynaKube) error {
	ctx, log := logd.NewFromContext(ctx, "extension-eec")

	// TODO: Remove as part of ICP-1086
	meta.RemoveStatusCondition(dk.Conditions(), "ExtensionsControllerStatefulSet")

	if ext := dk.Extensions(); !ext.IsAnyEnabled() {
		if meta.FindStatusCondition(*dk.Conditions(), extensionControllerStatefulSetConditionType) == nil {
			return nil
		}
		defer meta.RemoveStatusCondition(dk.Conditions(), extensionControllerStatefulSetConditionType)

		sts, err := k8sstatefulset.Build(dk, ext.GetExecutionControllerStatefulsetName(), corev1.Container{})
		if err != nil {
			log.Error(err, "could not build "+ext.GetExecutionControllerStatefulsetName()+" during cleanup")

			return err
		}

		err = k8sstatefulset.Query(r.client, r.apiReader).Delete(ctx, sts)
		if err != nil {
			log.Error(err, "failed to clean up "+ext.GetExecutionControllerStatefulsetName()+" statufulset")
		}

		return nil
	}

	if dk.Status.ActiveGate.ConnectionInfo.TenantUUID == "" {
		k8sconditions.SetStatefulSetOutdated(dk.Conditions(), extensionControllerStatefulSetConditionType, dk.Extensions().GetExecutionControllerStatefulsetName())

		return errors.New("tenantUUID unknown")
	}

	if dk.Status.KubeSystemUUID == "" {
		k8sconditions.SetStatefulSetOutdated(dk.Conditions(), extensionControllerStatefulSetConditionType, dk.Extensions().GetExecutionControllerStatefulsetName())

		return errors.New("kubeSystemUUID unknown")
	}

	return r.createOrUpdateStatefulset(ctx, dk)
}
