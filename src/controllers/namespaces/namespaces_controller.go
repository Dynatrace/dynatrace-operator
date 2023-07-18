package namespaces

import (
	"os"
	"time"

	"github.com/Dynatrace/dynatrace-operator/src/api/status"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1/dynakube"
	dtingestendpoint "github.com/Dynatrace/dynatrace-operator/src/ingestendpoint"
	"github.com/Dynatrace/dynatrace-operator/src/initgeneration"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/src/mapper"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/src/webhook"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	// updateInterval = 5 * time.Minute
	updateInterval = 1 * time.Minute
)

func Add(mgr manager.Manager, _ string) error {
	return newController(mgr.GetClient(), mgr.GetAPIReader(), mgr.GetScheme()).SetupWithManager(mgr)
}

func newController(kubeClient client.Client, apiReader client.Reader, scheme *runtime.Scheme) *Controller {
	return &Controller{
		client:            kubeClient,
		apiReader:         apiReader,
		scheme:            scheme,
		operatorNamespace: os.Getenv(kubeobjects.EnvPodNamespace),
	}
}

/*
// if namespace is created with a label matching any dynakube selector then namespace mutator adds InjectionInstanceLabel
//
//	kubectl create namespace starhole --dry-run=client -o json|jq '.metadata += {"labels":{"monitor":"star"}}' | kubectl apply -f -
func namespacePredicate(controller *Controller) predicate.Predicate {
	// return predicate.Or(predicate.Funcs{CreateFunc: createFunc(controller)},
	//	predicate.And(predicate.LabelChangedPredicate{}, predicate.Funcs{UpdateFunc: updateFunc(controller)}))
	return predicate.Funcs{
		CreateFunc: createFunc(controller),
		UpdateFunc: updateFunc(controller),
	}
}

func createFunc(controller *Controller) func(createEvent event.CreateEvent) bool {
	return func(createEvent event.CreateEvent) bool {
		log.Info("create namespace", "namespace", createEvent.Object.GetNamespace(), "name", createEvent.Object.GetName())
		isLabelled := false
		labels := createEvent.Object.GetLabels()
		if labels != nil {
			if dynakubeName, ok := labels[dtwebhook.InjectionInstanceLabel]; ok {
				err := controller.reconcileDynakubeSecretsForNamespace(context.TODO(), dynakubeName, controller.operatorNamespace, createEvent.Object.GetName())
				if err != nil {
					log.Error(err, "error while creating namespace", "dynakubeName", dynakubeName, "dynakubeNamespace", controller.operatorNamespace)
				}
				isLabelled = true
				log.Info("create namespace done", "namespace", createEvent.Object.GetNamespace(), "name", createEvent.Object.GetName(), "labels", isLabelled, "dynakubeName", dynakubeName, "dynakubeNamespace", controller.operatorNamespace)
			}
		}
		if !isLabelled {
			log.Info("create namespace done", "namespace", createEvent.Object.GetNamespace(), "name", createEvent.Object.GetName(), "labels", isLabelled)
		}
		return false
	}
}

func updateFunc(controller *Controller) func(updateEvent event.UpdateEvent) bool { // nolint:revive
	return func(updateEvent event.UpdateEvent) bool {
		log.Info("update namespace", "namespace", updateEvent.ObjectNew.GetNamespace(), "name", updateEvent.ObjectNew.GetName())

		if reflect.DeepEqual(updateEvent.ObjectOld.GetLabels(), updateEvent.ObjectNew.GetLabels()) {
			log.Info("update namespace done - labels not changed", "namespace", updateEvent.ObjectNew.GetName(), "labels", updateEvent.ObjectNew.GetLabels())
			return false
		}

		log.Info("namespace labels old", "namespace", updateEvent.ObjectOld.GetName(), "labels", updateEvent.ObjectOld.GetLabels())
		log.Info("namespace labels new", "namespace", updateEvent.ObjectNew.GetName(), "labels", updateEvent.ObjectNew.GetLabels())

		newInstanceLabel := ""
		if updateEvent.ObjectNew.GetLabels() != nil {
			if dynakubeName, ok := updateEvent.ObjectNew.GetLabels()[dtwebhook.InjectionInstanceLabel]; ok {
				newInstanceLabel = dynakubeName
			}
		}
		if newInstanceLabel != "" {
			err := controller.reconcileDynakubeSecretsForNamespace(context.TODO(), newInstanceLabel, controller.operatorNamespace, updateEvent.ObjectNew.GetName())
			if err != nil {
				log.Info("error while updating namespace", "new dynakubeName", newInstanceLabel, "dynakubeNamespace", controller.operatorNamespace, "error", err)
			}

			log.Info("update namespace done", "namespace", updateEvent.ObjectNew.GetNamespace(), "name", updateEvent.ObjectNew.GetName(), "dynakubeName", newInstanceLabel, "dynakubeNamespace", controller.operatorNamespace)
		} else {
			log.Info("update namespace done - event ignored", "namespace", updateEvent.ObjectNew.GetNamespace(), "name", updateEvent.ObjectNew.GetName())
		}
		return false
	}
}
*/

func watchMapper(controller *Controller) handler.MapFunc {
	return func(object client.Object) []reconcile.Request {
		labels := object.GetLabels()
		if labels != nil {
			if dynakubeName, ok := labels[dtwebhook.InjectionInstanceLabel]; ok {
				log.Info("watchMapper - dynakube found", "namespaceName", object.GetName(), "dynakube", dynakubeName)
				return []reconcile.Request{
					{
						NamespacedName: types.NamespacedName{
							Name:      dynakubeName,
							Namespace: controller.operatorNamespace,
						},
					},
				}
			}
		}
		log.Info("watchMapper - dynakube not found", "namespaceName", object.GetName())
		return []reconcile.Request{}
	}
}

func (controller *Controller) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&dynatracev1beta1.DynaKube{}).
		// Watches(&source.Kind{Type: &corev1.Namespace{}}, &handler.EnqueueRequestForObject{}, builder.WithPredicates(namespacePredicate(controller))).
		Watches(&source.Kind{Type: &corev1.Namespace{}}, handler.EnqueueRequestsFromMapFunc(watchMapper(controller))).
		Owns(&corev1.Secret{}).
		Complete(controller)
}

// Controller reconciles a Namespace object
type Controller struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the api-server
	client            client.Client
	apiReader         client.Reader
	scheme            *runtime.Scheme
	operatorNamespace string
}

// Reconcile reads that state of the cluster for a DynaKube object and makes changes based on the state read
// and what is in the DynaKube.Spec
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (controller *Controller) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	if request.Namespace == "" {
		log.Info("reconciling Namespace (secrets)", "namespace", request.Namespace, "name", request.Name)
		log.Info("reconciling Namespace (secrets) - done", "namespace", request.Namespace, "name", request.Name)
		return reconcile.Result{}, nil
	} else {
		return controller.reconcileDynakube(ctx, request.Name, request.Namespace)
	}
}

func (controller *Controller) reconcileDynakube(ctx context.Context, dynakubeName, dynakubeNamespace string) (reconcile.Result, error) {
	dynakube, err := controller.getDynakubeOrUnmap(ctx, dynakubeName, dynakubeNamespace)
	if err != nil {
		log.Info("reconciling DynaKube (secrets) - couldn't get dynakbe", "namespace", dynakubeNamespace, "dynakube", dynakubeName, "error", err)
		return reconcile.Result{RequeueAfter: updateInterval}, err
	} else if dynakube == nil {
		log.Info("reconciling DynaKube (secrets) - not found", "namespace", dynakubeNamespace, "dynakube", dynakubeName)
		return reconcile.Result{}, nil
	}

	oldNamespaceSecretsPhase := dynakube.Status.NamespaceSecretsPhase

	result, err := controller.reconcileDynakubeLabelsAndSecrets(ctx, dynakube)

	if err != nil {
		dynakube.Status.SetNamespaceSecretsPhase(status.Error)
	} else {
		dynakube.Status.SetNamespaceSecretsPhase(status.Running)
	}

	if val, ok := dynakube.Annotations["namespacesecretsphase"]; ok {
		if val == "error" {
			log.Info("reconciling DynaKube (secrets) - namespacesecretsphase set to 'error'", "namespace", dynakubeNamespace, "dynakube", dynakubeName)
			dynakube.Status.SetNamespaceSecretsPhase(status.Error)
		}
	}

	if oldNamespaceSecretsPhase != dynakube.Status.NamespaceSecretsPhase {
		log.Info("status changed, updating DynaKube Status.NamespaceSecretsPhase", "old", oldNamespaceSecretsPhase, "new", dynakube.Status.NamespaceSecretsPhase)
		if errClient := controller.updateDynakubeStatus(ctx, dynakube); errClient != nil {
			return reconcile.Result{}, errors.WithMessagef(errClient, "failed to update DynaKube Status.NamespaceSecretsPhase after failure, original error: %s", err)
		}
	}
	return result, err
}

func (controller *Controller) reconcileDynakubeLabelsAndSecrets(ctx context.Context, dynakube *dynatracev1beta1.DynaKube) (reconcile.Result, error) {
	log.Info("reconciling DynaKube labels and secrets", "namespace", dynakube.Namespace, "name", dynakube.Name, "generation", dynakube.GetGeneration())
	if result, err := controller.reconcileDynakubeLabels(ctx, dynakube); err != nil {
		return result, err
	}

	return controller.reconcileDynakubeSecrets(ctx, dynakube)
}

/*
	func (controller *Controller) reconcileDynakubeSecretsForNamespace(ctx context.Context, dynakubeName, dynakubeNamespace string, targetNs string) error {
		log.Info("reconciling DynaKube for namespace (secrets)", "namespace", dynakubeNamespace, "name", dynakubeName, "targetNs", targetNs)

		dynakube, err := controller.getDynakubeOrUnmap(ctx, dynakubeName, dynakubeNamespace)
		if err != nil {
			log.Info("reconciling DynaKube for namespace (secrets) - couldn't get dynakbe", "namespace", dynakubeNamespace, "dynakube", dynakubeName, "error", err)
			return err
		} else if dynakube == nil {
			log.Info("reconciling DynaKube for namespace (secrets) - not found", "namespace", dynakubeNamespace, "dynakube", dynakubeName)
			return nil
		}

		log.Info("reconciling DynaKube for namespace (secrets) - generation", "namespace", dynakube.Namespace, "name", dynakube.Name, "generation", "targetNs", targetNs, dynakube.GetGeneration())

		oldNamespaceSecretsPhase := dynakube.Status.NamespaceSecretsPhase

		if dynakube.NeedAppInjection() {
			err = controller.setupAppInjectionForNamespace(ctx, dynakube, targetNs)
		}

		if err != nil {
			dynakube.Status.SetNamespaceSecretsPhase(status.Error)
		} else {
			dynakube.Status.SetNamespaceSecretsPhase(status.Running)
		}

		if val, ok := dynakube.Annotations["namespacesecretsphase"]; ok {
			if val == "error" {
				log.Info("reconciling DynaKube for namespace (secrets) - namespacesecretsphase set to 'error'", "namespace", dynakubeNamespace, "dynakube", dynakubeName)
				dynakube.Status.SetNamespaceSecretsPhase(status.Error)
			}
		}

		if oldNamespaceSecretsPhase != dynakube.Status.NamespaceSecretsPhase && dynakube.Status.NamespaceSecretsPhase == status.Error {
			log.Info("reconciling DynaKube for namespace (secrets) - status changed, updating DynaKube Status.NamespaceSecretsPhase", "old", oldNamespaceSecretsPhase, "new", dynakube.Status.NamespaceSecretsPhase)
			if errClient := controller.updateDynakubeStatus(ctx, dynakube); errClient != nil {
				return errors.WithMessagef(errClient, "failed to update DynaKube Status.NamespaceSecretsPhase after failure, original error: %s", err)
			}
		}

		log.Info("reconciling DynaKube for namespace (secrets) - done", "namespace", dynakubeNamespace, "name", dynakubeName, "targetNs", targetNs, "generation", dynakube.GetGeneration())
		return err
	}
*/
func (controller *Controller) reconcileDynakubeSecrets(ctx context.Context, dynakube *dynatracev1beta1.DynaKube) (reconcile.Result, error) {
	log.Info("reconciling DynaKube (secrets)", "namespace", dynakube.Namespace, "name", dynakube.Name)

	err := controller.reconcileAppInjection(ctx, dynakube)
	if err != nil {
		return reconcile.Result{RequeueAfter: updateInterval}, err
	}

	log.Info("reconciling Dynakube (secrets) - done", "namespace", dynakube.Namespace, "name", dynakube.Name, "requeueAfter", updateInterval)
	return reconcile.Result{RequeueAfter: updateInterval}, nil
}

func (controller *Controller) reconcileDynakubeLabels(ctx context.Context, dynakube *dynatracev1beta1.DynaKube) (reconcile.Result, error) {
	log.Info("reconciling DynaKube (labels)", "namespace", dynakube.Namespace, "name", dynakube.Name)

	if dynakube.NeedAppInjection() {
		dkMapper := controller.createDynakubeMapper(ctx, dynakube)

		if err := dkMapper.MapFromDynakube(); err != nil {
			log.Info("update of a map of namespaces failed")
			return reconcile.Result{RequeueAfter: updateInterval}, err
		}
	}
	log.Info("reconciling DynaKube (labels) done", "namespace", dynakube.Namespace, "name", dynakube.Name)
	return reconcile.Result{}, nil
}

func (controller *Controller) getDynakubeOrUnmap(ctx context.Context, dkName, dkNamespace string) (*dynatracev1beta1.DynaKube, error) {
	dynakube := &dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dkName,
			Namespace: dkNamespace,
		},
	}
	err := controller.apiReader.Get(ctx, client.ObjectKey{Name: dynakube.Name, Namespace: dynakube.Namespace}, dynakube)
	if k8serrors.IsNotFound(err) {
		return nil, controller.createDynakubeMapper(ctx, dynakube).UnmapFromDynaKube()
	} else if err != nil {
		return nil, errors.WithStack(err)
	}
	return dynakube, nil
}

func (controller *Controller) reconcileAppInjection(ctx context.Context, dynakube *dynatracev1beta1.DynaKube) error {
	if dynakube.NeedAppInjection() {
		return controller.setupAppInjectionForDynakube(ctx, dynakube)
	}

	return controller.removeAppInjection(ctx, dynakube)
}

func (controller *Controller) setupAppInjectionForDynakube(ctx context.Context, dynakube *dynatracev1beta1.DynaKube) (err error) {
	log.Info("setup app injection", "dynakube", dynakube.Name)
	err = initgeneration.NewInitGenerator(controller.client, controller.apiReader, dynakube.Namespace).GenerateForDynakube(ctx, dynakube)
	if err != nil {
		log.Info("failed to generate init secret")
		return err
	}

	endpointSecretGenerator := dtingestendpoint.NewEndpointSecretGenerator(controller.client, controller.apiReader, dynakube.Namespace)
	err = endpointSecretGenerator.GenerateForDynakube(ctx, dynakube)
	if err != nil {
		log.Info("failed to generate data-ingest secret")
		return err
	}

	log.Info("app injection reconciled", "dynakube", dynakube.Name)
	return nil
}

/*
	func (controller *Controller) setupAppInjectionForNamespace(ctx context.Context, dynakube *dynatracev1beta1.DynaKube, targetNs string) (err error) {
		log.Info("setup app injection for namespace", "dynakube", dynakube.Name, "namespace", targetNs, "generation", dynakube.GetGeneration())
		err = initgeneration.NewInitGenerator(controller.client, controller.apiReader, dynakube.Namespace).GenerateForNamespace(ctx, *dynakube, targetNs)
		if err != nil {
			log.Info("setup app injection for namespace - failed to generate init secret")
			return err
		}

		err = dtingestendpoint.NewEndpointSecretGenerator(controller.client, controller.apiReader, dynakube.Namespace).GenerateForNamespace(ctx, dynakube.Name, targetNs)
		if err != nil {
			log.Info("setup app injection for namespace - failed to generate data-ingest secret")
			return err
		}

		log.Info("app injection for namespace reconciled", "dynakube", dynakube.Name, "namespace", targetNs, "generation", dynakube.GetGeneration())
		return nil
	}
*/
func (controller *Controller) removeAppInjection(ctx context.Context, dynakube *dynatracev1beta1.DynaKube) (err error) {
	endpointSecretGenerator := dtingestendpoint.NewEndpointSecretGenerator(controller.client, controller.apiReader, dynakube.Namespace)
	dkMapper := controller.createDynakubeMapper(ctx, dynakube)

	if err := dkMapper.UnmapFromDynaKube(); err != nil {
		log.Info("could not unmap dynakube from namespace")
		return err
	}
	err = endpointSecretGenerator.RemoveEndpointSecrets(ctx, dynakube)
	if err != nil {
		log.Info("could not remove data-ingest secret")
		return err
	}
	return nil
}

func (controller *Controller) createDynakubeMapper(ctx context.Context, dynakube *dynatracev1beta1.DynaKube) *mapper.DynakubeMapper {
	dkMapper := mapper.NewDynakubeMapper(ctx, controller.client, controller.apiReader, controller.operatorNamespace, dynakube)
	return &dkMapper
}

func (controller *Controller) updateDynakubeStatus(ctx context.Context, dynakube *dynatracev1beta1.DynaKube) error {
	dynakube.Status.UpdatedTimestamp = metav1.Now()
	err := controller.client.Status().Update(ctx, dynakube)
	if err != nil && k8serrors.IsConflict(err) {
		log.Info("could not update dynakube due to conflict", "name", dynakube.Name)
		return nil
	}
	return errors.WithStack(err)
}
