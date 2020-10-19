package activegate

import (
	"context"

	"github.com/Dynatrace/dynatrace-activegate-operator/pkg/apis/dynatrace/v1alpha1"
	"github.com/Dynatrace/dynatrace-activegate-operator/pkg/controller/builder"
	"github.com/Dynatrace/dynatrace-activegate-operator/pkg/controller/dao"
	"github.com/Dynatrace/dynatrace-activegate-operator/pkg/dtclient"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func (r *ReconcileActiveGate) createDesiredStatefulSet(instance *v1alpha1.ActiveGate, dtc dtclient.Client) (*appsv1.StatefulSet, error) {
	tenantInfo, err := dtc.GetTenantInfo()
	if err != nil {
		return nil, err
	}

	uid, err := dao.FindKubeSystemUID(r.client)
	if err != nil {
		return nil, err
	}

	desiredStatefulSet, err := r.newStatefulSetForCR(instance, tenantInfo, uid)
	if err != nil {
		return nil, err
	}
	return desiredStatefulSet, nil
}

func (r *ReconcileActiveGate) manageStatefulSet(desiredStatefulSet *appsv1.StatefulSet, actualStatefulSet *appsv1.StatefulSet, log logr.Logger) (*reconcile.Result, error) {
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: desiredStatefulSet.Name, Namespace: desiredStatefulSet.Namespace}, actualStatefulSet)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating new statefulset")
		if err = r.client.Create(context.TODO(), desiredStatefulSet); err != nil {
			return &reconcile.Result{}, err
		} else {
			result := builder.ReconcileImmediately()
			return &result, nil
		}
	} else if err != nil {
		return &reconcile.Result{}, err
	} else if hasStatefulSetChanged(desiredStatefulSet, actualStatefulSet) {
		log.Info("Updating existing statefulset")
		if err = r.client.Update(context.TODO(), desiredStatefulSet); err != nil {
			return &reconcile.Result{}, err
		}
	}
	return nil, nil
}
