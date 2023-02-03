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
	foundAutoscaler *scalingv2.HorizontalPodAutoscaler
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
	toReconcile, err := reconciler.builder.newAutoscaler()
	if err != nil {
		return errors.WithStack(err)
	}

	err = reconciler.findAutoscaler(toReconcile)
	if err != nil {
		return errors.WithStack(err)
	}

	if reconciler.ignores(toReconcile) {
		return nil
	}

	switch {
	case reconciler.foundAutoscaler == nil &&
		reconciler.builder.DynaKube.IsSyntheticMonitoringEnabled():
		err = reconciler.create(toReconcile)
	case reconciler.foundAutoscaler != nil:
		if reconciler.builder.DynaKube.IsSyntheticMonitoringEnabled() {
			err = reconciler.update(toReconcile)
		} else {
			err = reconciler.delete()
		}
	}

	return errors.WithStack(err)
}

func (reconciler *Reconciler) findAutoscaler(toFind *scalingv2.HorizontalPodAutoscaler) (err error) {
	reconciler.foundAutoscaler, err = reconciler.autoscalers.Get(toFind)
	if apierrors.IsNotFound(err) {
		err = nil
	}

	return err
}

func (reconciler *Reconciler) ignores(toReconcile *scalingv2.HorizontalPodAutoscaler) bool {
	return reconciler.foundAutoscaler != nil &&
		toReconcile.GetAnnotations()[kubeobjects.AnnotationHash] ==
			reconciler.foundAutoscaler.GetAnnotations()[kubeobjects.AnnotationHash]
}

func (reconciler *Reconciler) create(toCreate *scalingv2.HorizontalPodAutoscaler) error {
	err := reconciler.autoscalers.Create(
		reconciler.builder.DynaKube,
		toCreate)
	if err != nil {
		log.Error(
			err,
			"could not create",
			"name", toCreate.Name)
		return errors.WithStack(err)
	}

	log.Info("created", "name", toCreate.Name)
	return nil
}

func (reconciler *Reconciler) update(toUpdate *scalingv2.HorizontalPodAutoscaler) error {
	err := reconciler.autoscalers.Update(reconciler.builder.DynaKube, toUpdate)
	if err == nil {
		log.Info("updated", "name", toUpdate.Name)
	}

	return errors.WithStack(err)
}

func (reconciler *Reconciler) delete() error {
	err := reconciler.autoscalers.Delete(reconciler.foundAutoscaler)
	if err != nil {
		log.Error(
			err,
			"could not delete",
			"name", reconciler.foundAutoscaler.Name)
		return errors.WithStack(err)
	}

	log.Info("deleted", "name", reconciler.foundAutoscaler.Name)
	return nil
}
