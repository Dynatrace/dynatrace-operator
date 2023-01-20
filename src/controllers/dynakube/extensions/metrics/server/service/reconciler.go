package service

import (
	"context"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/extensions/metrics/common"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	regv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Reconciler struct {
	*builder
	services *kubeobjects.ApiRequests[
		corev1.Service,
		*corev1.Service,
		corev1.ServiceList,
		*corev1.ServiceList,
	]
	apiServices *kubeobjects.ApiRequests[
		regv1.APIService,
		*regv1.APIService,
		regv1.APIServiceList,
		*regv1.APIServiceList,
	]
	clusterRoles *kubeobjects.ApiRequests[
		rbacv1.ClusterRole,
		*rbacv1.ClusterRole,
		rbacv1.ClusterRoleList,
		*rbacv1.ClusterRoleList,
	]
}

var (
	_ controllers.Reconciler = (*Reconciler)(nil)

	dynaMetricClusterRole = &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: common.DynaMetricClusterRoleName,
		},
	}
)

//nolint:revive
func NewReconciler(
	context context.Context,
	reader client.Reader,
	client client.Client,
	scheme *runtime.Scheme,
	dynakube *dynatracev1beta1.DynaKube,
	deployment *appsv1.Deployment,
) controllers.Reconciler {
	return &Reconciler{
		builder: newBuilder(dynakube, deployment),
		services: kubeobjects.NewApiRequests[
			corev1.Service,
			*corev1.Service,
			corev1.ServiceList,
			*corev1.ServiceList,
		](
			context,
			reader,
			client,
			scheme,
		),
		apiServices: kubeobjects.NewApiRequests[
			regv1.APIService,
			*regv1.APIService,
			regv1.APIServiceList,
			*regv1.APIServiceList,
		](
			context,
			reader,
			client,
			scheme,
		),
		clusterRoles: kubeobjects.NewApiRequests[
			rbacv1.ClusterRole,
			*rbacv1.ClusterRole,
			rbacv1.ClusterRoleList,
			*rbacv1.ClusterRoleList,
		](
			context,
			reader,
			client,
			scheme,
		),
	}
}

func (reconciler *Reconciler) Reconcile() error {
	service := reconciler.builder.newService()
	err := reconciler.services.Create(
		reconciler.builder.DynaKube,
		service)
	if err == nil {
		common.Log.Info(
			"created service",
			"name", service.ObjectMeta.Name)

		var apiServiceOwner *rbacv1.ClusterRole
		apiServiceOwner, err = reconciler.clusterRoles.Get(dynaMetricClusterRole)
		if err == nil {
			apiService := reconciler.builder.newApiService()
			err = reconciler.apiServices.Create(
				apiServiceOwner,
				apiService)
			if err == nil {
				common.Log.Info(
					"created API service",
					"name", apiService.ObjectMeta.Name)
			}
		}
	}

	return errors.WithStack(err)
}
