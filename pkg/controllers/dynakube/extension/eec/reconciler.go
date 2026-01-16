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
	// TODO: Remove as part of DAQ-18375
	meta.RemoveStatusCondition(r.dk.Conditions(), "ExtensionsControllerStatefulSet")

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
		}

		r.deleteLegacyStatefulset(ctx)

		return nil
	}

	// TODO: this was misused, "Outdated" was only ever meant to reset the condition's timestamp
	// which is in general just feels like an anti-pattern (we used it to use the condition's timestamp for throttling)
	// The "lastTransitionTime" should only be updated when the status actually changes, so it can tell you how long was everything in a given state
	// and the "state" is "is everything ok/expected" and not "is everything ok for this exact spec"
	if r.dk.Status.ActiveGate.ConnectionInfo.TenantUUID == "" {
		err := errors.New("tenantUUID unknown, " + extensionControllerStatefulSetConditionType + " cannot be created")

		k8sconditions.SetStatefulSetGenFailed(r.dk.Conditions(), extensionControllerStatefulSetConditionType, err)

		return err
	}

	if r.dk.Status.KubeSystemUUID == "" {
		err := errors.New("kubeSystemUUID unknown, " + extensionControllerStatefulSetConditionType + " cannot be created")

		k8sconditions.SetStatefulSetGenFailed(r.dk.Conditions(), extensionControllerStatefulSetConditionType, err)

		return err
	}

	defer r.deleteLegacyStatefulset(ctx)

	return r.createOrUpdateStatefulset(ctx)
}
