package server

import (
	"context"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/extensions/metrics/common"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/extensions/metrics/server/service"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Reconciler struct {
	*dynatracev1beta1.DynaKube
	deployments *kubeobjects.ApiRequests[
		appsv1.Deployment,
		*appsv1.Deployment,
		appsv1.DeploymentList,
		*appsv1.DeploymentList,
	]
	toIdentifyDeployment *appsv1.Deployment
	foundDeployment      *appsv1.Deployment
}

var _ controllers.Reconciler = (*Reconciler)(nil)

//nolint:revive
func NewReconciler(
	context context.Context,
	reader client.Reader,
	client client.Client,
	scheme *runtime.Scheme,
	dynaKube *dynatracev1beta1.DynaKube,
) controllers.Reconciler {
	return &Reconciler{
		DynaKube: dynaKube,
		deployments: kubeobjects.NewApiRequests[
			appsv1.Deployment,
			*appsv1.Deployment,
			appsv1.DeploymentList,
			*appsv1.DeploymentList,
		](
			context,
			reader,
			client,
			scheme,
		),
		toIdentifyDeployment: &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      common.KubjectNamePrefix,
				Namespace: dynaKube.Namespace,
			},
		},
	}
}

func (reconciler *Reconciler) Reconcile() (err error) {
	toReconcile, err := newBuilder(reconciler.DynaKube).newDeployment()
	if err != nil {
		return errors.WithStack(err)
	}

	err = reconciler.findDeployment()
	if err != nil {
		return errors.WithStack(err)
	}

	if reconciler.ignores(toReconcile) {
		return nil
	}

	if reconciler.foundDeployment == nil {
		err = reconciler.create(toReconcile)
	} else {
		err = reconciler.update(toReconcile)
	}

	return errors.WithStack(err)
}

func (reconciler *Reconciler) findDeployment() (err error) {
	reconciler.foundDeployment, err = reconciler.deployments.Get(
		reconciler.toIdentifyDeployment)
	if apierrors.IsNotFound(err) {
		err = nil
	}

	return errors.WithStack(err)
}

func (reconciler *Reconciler) ignores(toReconcile *appsv1.Deployment) bool {
	return reconciler.foundDeployment != nil &&
		toReconcile.GetAnnotations()[kubeobjects.AnnotationHash] ==
			reconciler.foundDeployment.GetAnnotations()[kubeobjects.AnnotationHash]
}

func (reconciler *Reconciler) create(toCreate *appsv1.Deployment) error {
	err := reconciler.deployments.Create(
		reconciler.DynaKube,
		toCreate)
	if err == nil {
		common.Log.Info(
			"created deployment",
			"name", toCreate.ObjectMeta.Name)

		err = service.NewReconciler(
			reconciler.deployments.Context,
			reconciler.deployments.Reader,
			reconciler.deployments.Client,
			reconciler.deployments.Scheme,
			reconciler.DynaKube,
			toCreate,
		).Reconcile()
	}

	return errors.WithStack(err)
}

func (reconciler *Reconciler) update(toUpdate *appsv1.Deployment) error {
	err := reconciler.deployments.Update(reconciler.DynaKube, toUpdate)
	if err == nil {
		common.Log.Info(
			"updated deployment",
			"name", toUpdate.ObjectMeta.Name)
	}

	return errors.WithStack(err)
}
