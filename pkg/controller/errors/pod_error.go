package errors

import (
	"context"
	"fmt"
	"github.com/Dynatrace/dynatrace-activegate-operator/pkg/controller/builder"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func HandleCreatePodError(client client.Client, pod *corev1.Pod, err error, log logr.Logger) (reconcile.Result, error) {
	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating a new Pod", "Pod.Namespace", pod.Namespace, "Pod.Name", pod.Name)
		err = client.Create(context.TODO(), pod)
		if err != nil {
			return reconcile.Result{}, err
		}

		// Pod created successfully - requeue after five minutes
		return builder.ReconcileAfterFiveMinutes(), nil
	} else if err != nil {
		return reconcile.Result{}, err
	}
	return reconcile.Result{}, fmt.Errorf("cannot handle 'nil' error")
}
