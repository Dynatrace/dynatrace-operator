package activegate

import (
	"context"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/apis/dynatrace/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/pkg/controller/builder"
	"github.com/Dynatrace/dynatrace-operator/pkg/controller/dao"
	"github.com/Dynatrace/dynatrace-operator/pkg/dtclient"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func (r *ReconcileActiveGate) createDesiredStatefulSet(instance *v1alpha1.DynaKube, dtc dtclient.Client) (*appsv1.StatefulSet, error) {
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

func (r *ReconcileActiveGate) manageStatefulSet(log logr.Logger, instance *v1alpha1.DynaKube, desiredStatefulSet *appsv1.StatefulSet, actualStatefulSet *appsv1.StatefulSet) (*reconcile.Result, error) {
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

		// Reset update timestamp after change of stateful set to enable immediate update check
		instance.Status.UpdatedTimestamp = metav1.NewTime(time.Now().Add(-5 * time.Minute))
		err = r.client.Status().Update(context.TODO(), instance)
		result := builder.ReconcileImmediately()
		return &result, err
	}
	return nil, nil
}
