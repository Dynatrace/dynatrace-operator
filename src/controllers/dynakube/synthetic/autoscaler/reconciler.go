package autoscaler

import (
	"context"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	scalingv2 "k8s.io/api/autoscaling/v2"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Reconciler struct {
	*builder
	autoscalers *kubeobjects.ApiRequests[
		scalingv2.HorizontalPodAutoscaler,
		*scalingv2.HorizontalPodAutoscaler,
		scalingv2.HorizontalPodAutoscalerList,
		*scalingv2.HorizontalPodAutoscalerList,
	]
}

var _ controllers.Reconciler = (*Reconciler)(nil)

//nolint:revive
func NewReconciler(
	context context.Context,
	reader client.Reader,
	client client.Client,
	scheme *runtime.Scheme,
	dynakube *dynatracev1beta1.DynaKube,
	statefulSet *appsv1.StatefulSet,
) controllers.Reconciler {
	return &Reconciler{
		builder: newBuilder(dynakube, statefulSet),
		autoscalers: kubeobjects.NewApiRequests[
			scalingv2.HorizontalPodAutoscaler,
			*scalingv2.HorizontalPodAutoscaler,
			scalingv2.HorizontalPodAutoscalerList,
			*scalingv2.HorizontalPodAutoscalerList,
		](
			context,
			reader,
			client,
			scheme,
		),
	}
}

func (reconciler *Reconciler) Reconcile() error {
	toReconcile := reconciler.builder.newAutoscaler()

	_, err := reconciler.autoscalers.Get(toReconcile)
	switch {
	case apierrors.IsNotFound(err):
		if reconciler.builder.DynaKube.IsSyntheticMonitoringEnabled() {
			err = reconciler.create(toReconcile)
		} else {
			err = nil
		}
	case err == nil &&
		!reconciler.builder.DynaKube.IsSyntheticMonitoringEnabled():
		err = reconciler.delete(toReconcile)
	}

	return errors.WithStack(err)
}

func (reconciler *Reconciler) create(toCreate *scalingv2.HorizontalPodAutoscaler) error {
	err := reconciler.autoscalers.Create(
		reconciler.builder.DynaKube,
		toCreate)
	if err != nil {
		log.Error(
			err,
			"could not create",
			"object", *toCreate)
		return errors.WithStack(err)
	}

	log.Info("created autoscaler")
	return nil
}

func (reconciler *Reconciler) delete(toDelete *scalingv2.HorizontalPodAutoscaler) error {
	err := reconciler.autoscalers.Delete(toDelete)
	if err != nil {
		log.Error(
			err,
			"could not delete",
			"object", *toDelete)
		return errors.WithStack(err)
	}

	log.Info("deleted autoscaler")
	return nil
}
