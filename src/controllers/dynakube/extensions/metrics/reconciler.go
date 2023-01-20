package metrics

import (
	"context"
	"os"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/extensions/metrics/common"
	metricsvr "github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/extensions/metrics/server"
	"github.com/Dynatrace/dynatrace-operator/src/functional"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	regv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Reconciler struct {
	apiServices *kubeobjects.ApiRequests[
		regv1.APIService,
		*regv1.APIService,
		regv1.APIServiceList,
		*regv1.APIServiceList,
	]
	foundApiService    *regv1.APIService
	foundDynaKubes     []dynatracev1beta1.DynaKube
	configDynaKubeName string
	configDynaKube     *dynatracev1beta1.DynaKube
}

var (
	_ controllers.Reconciler = (*Reconciler)(nil)

	toIdentifyApiService = &regv1.APIService{
		ObjectMeta: metav1.ObjectMeta{
			Name: common.ApiServiceVersionGroup,
		},
	}

	operatorNamespace = client.InNamespace(os.Getenv("POD_NAMESPACE"))
)

//nolint:revive
func NewReconciler(
	context context.Context,
	reader client.Reader,
	client client.Client,
	scheme *runtime.Scheme,
) controllers.Reconciler {
	return &Reconciler{
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
	}
}

func (reconciler *Reconciler) Reconcile() error {
	err := reconciler.findApiService()
	if err != nil {
		return errors.WithStack(err)
	}
	err = reconciler.findDynaKubesBySynMonitoring()
	if err != nil {
		return errors.Wrapf(err, "could not list DynaKubes")
	}
	reconciler.findConfigDynaKube()

	switch {
	case reconciler.creates():
		err = reconciler.create()
	case reconciler.updates():
		err = reconciler.update()
	case reconciler.deletes():
		err = reconciler.delete()
	}

	if err == nil {
		common.Log.Info("reconciled DynaMetrics")
	}
	return errors.WithStack(err)
}

func (reconciler *Reconciler) findApiService() (err error) {
	reconciler.foundApiService, err = reconciler.apiServices.Get(toIdentifyApiService)
	switch {
	case apierrors.IsNotFound(err):
		err = nil
	case reconciler.foundApiService != nil &&
		reconciler.foundApiService.GetAnnotations() != nil:
		reconciler.configDynaKubeName = reconciler.foundApiService.GetAnnotations()[common.ControlledByDynaKubeAnnotation]
	}

	return err
}

func (reconciler *Reconciler) findDynaKubesBySynMonitoring() error {
	dynakubes, err := kubeobjects.NewApiRequests[
		dynatracev1beta1.DynaKube,
		*dynatracev1beta1.DynaKube,
		dynatracev1beta1.DynaKubeList,
		*dynatracev1beta1.DynaKubeList,
	](
		reconciler.apiServices.Context,
		reconciler.apiServices.Reader,
		reconciler.apiServices.Client,
		reconciler.apiServices.Scheme,
	).List(operatorNamespace)

	if err != nil {
		return errors.WithStack(err)
	}

	reconciler.foundDynaKubes = functional.Filter(
		dynakubes.Items,
		controlsSynMonitoring)
	return nil
}

func controlsSynMonitoring(dynaKube dynatracev1beta1.DynaKube) bool {
	return dynaKube.IsSyntheticMonitoringEnabled() &&
		dynaKube.Status.ConnectionInfo.TenantUUID != ""
}

func (reconciler *Reconciler) creates() bool {
	return len(reconciler.foundDynaKubes) > 0 &&
		reconciler.configDynaKube == nil
}

func (reconciler *Reconciler) findConfigDynaKube() {
	configDynaKubes := functional.Filter(
		reconciler.foundDynaKubes,
		reconciler.configuresDynaMetrics)

	if len(configDynaKubes) > 0 {
		reconciler.configDynaKube = &configDynaKubes[0]
	}
}

func (reconciler *Reconciler) configuresDynaMetrics(dynaKube dynatracev1beta1.DynaKube) bool {
	return dynaKube.Name == reconciler.configDynaKubeName
}

func (reconciler *Reconciler) create() error {
	return metricsvr.NewReconciler(
		reconciler.apiServices.Context,
		reconciler.apiServices.Reader,
		reconciler.apiServices.Client,
		reconciler.apiServices.Scheme,
		&reconciler.foundDynaKubes[0],
	).Reconcile()
}

func (reconciler *Reconciler) updates() bool {
	return len(reconciler.foundDynaKubes) > 0 &&
		reconciler.configDynaKube != nil
}

func (reconciler *Reconciler) update() error {
	return metricsvr.NewReconciler(
		reconciler.apiServices.Context,
		reconciler.apiServices.Reader,
		reconciler.apiServices.Client,
		reconciler.apiServices.Scheme,
		reconciler.configDynaKube,
	).Reconcile()
}

func (reconciler *Reconciler) deletes() bool {
	return len(reconciler.foundDynaKubes) == 0 &&
		reconciler.foundApiService != nil
}

func (reconciler *Reconciler) delete() error {
	err := reconciler.apiServices.Delete(toIdentifyApiService)
	if err == nil {
		common.Log.Info(
			"deleted component",
			"resource", *toIdentifyApiService)
	}

	return errors.WithStack(err)
}
