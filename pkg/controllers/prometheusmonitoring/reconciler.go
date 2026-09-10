// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package prometheusmonitoring

import (
	"context"
	"errors"
	"fmt"
	"maps"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/prometheusmonitoring"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/image"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/token"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/prometheusmonitoring/gateway"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/prometheusmonitoring/scraper"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/prometheusmonitoring/targetallocator"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8senv"
	k8sobject "github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

var (
	errDynaKubeNotFound           = errors.New("dynakube not found")
	errDynaKubeNotReady           = errors.New("dynakube not ready")
	errDataIngestTokenUnavailable = errors.New("data-ingest token not available")
)

func Add(mgr manager.Manager, _ string) error {
	return NewReconciler(mgr.GetClient()).SetupWithManager(mgr)
}

func NewReconciler(c client.Client) *Reconciler {
	return &Reconciler{
		Client:             c,
		targetAllocator:    &targetallocator.Reconciler{Client: c},
		gateway:            &gateway.Reconciler{Client: c},
		scraper:            &scraper.Reconciler{Client: c},
		newDynatraceClient: dynatrace.NewClientFromDynakube,
	}
}

type Reconciler struct {
	client.Client

	targetAllocator targetAllocatorReconciler
	gateway         gatewayReconciler
	scraper         scraperReconciler

	newDynatraceClient dynatrace.ClientFactory
}

type targetAllocatorReconciler interface {
	Reconcile(ctx context.Context, dtp *prometheusmonitoring.PrometheusMonitoring, dk *dynakube.DynaKube, imageClient image.Client) error
}

type gatewayReconciler interface {
	Reconcile(ctx context.Context, dtp *prometheusmonitoring.PrometheusMonitoring, dk *dynakube.DynaKube, imageClient image.Client) error
}

type scraperReconciler interface {
	Reconcile(ctx context.Context, dtp *prometheusmonitoring.PrometheusMonitoring, dk *dynakube.DynaKube, imageClient image.Client) error
}

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, reterr error) {
	log := logd.FromContext(ctx)

	dtp := &prometheusmonitoring.PrometheusMonitoring{}
	if err := r.Get(ctx, req.NamespacedName, dtp); err != nil {
		if !k8serrors.IsNotFound(err) {
			return ctrl.Result{}, fmt.Errorf("get prometheusmonitoring: %w", err)
		}

		log.Info("resource already deleted")

		return ctrl.Result{}, nil
	}

	defer func() {
		reterr = setPhase(dtp, reterr)

		if applyErr := k8sobject.ApplyStatus(ctx, r, dtp); applyErr != nil {
			if reterr != nil {
				// The reconciler error has higher precedence than updating the status, but the information should not be lost.
				log.Error(applyErr, "failed applying status")

				return
			}

			reterr = applyErr
		}
	}()

	log = log.WithValues("dynaKubeRef", dtp.Spec.DynaKubeRef)

	dk := &dynakube.DynaKube{}
	if err := r.Get(ctx, client.ObjectKey{Name: dtp.Spec.DynaKubeRef, Namespace: dtp.Namespace}, dk); err != nil {
		if !k8serrors.IsNotFound(err) {
			return ctrl.Result{}, fmt.Errorf("get dynakube %s: %w", dtp.Spec.DynaKubeRef, err)
		}

		log.Info("skipping reconcile due to missing DynaKube")

		return ctrl.Result{}, errDynaKubeNotFound
	}

	if dk.Status.Phase != status.Running {
		log.Info("skipping reconcile due to pending DynaKube", "phase", dk.Status.Phase)

		return ctrl.Result{}, errDynaKubeNotReady
	}

	if tokens, err := token.NewReader(r, dk).ReadTokens(ctx); err != nil || !token.CheckForDataIngestToken(tokens) {
		log.Info("skipping reconcile: data-ingest token not available")

		return ctrl.Result{}, errDataIngestTokenUnavailable
	}

	dtClient, err := r.buildDynatraceClient(ctx, dk)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("build dynatrace client: %w", err)
	}

	if err := r.gateway.Reconcile(ctx, dtp, dk, dtClient.Images); err != nil {
		return ctrl.Result{}, fmt.Errorf("reconcile gateway: %w", err)
	}

	if err := r.targetAllocator.Reconcile(ctx, dtp, dk, dtClient.Images); err != nil {
		return ctrl.Result{}, fmt.Errorf("reconcile target allocator: %w", err)
	}

	// Reconciled last: its config references the gateway and target allocator services.
	if err := r.scraper.Reconcile(ctx, dtp, dk, dtClient.Images); err != nil {
		return ctrl.Result{}, fmt.Errorf("reconcile scraper: %w", err)
	}

	return ctrl.Result{}, nil
}

func (r *Reconciler) buildDynatraceClient(ctx context.Context, dk *dynakube.DynaKube) (*dynatrace.Client, error) {
	tokens, err := token.NewReader(r, dk).ReadTokens(ctx)
	if err != nil {
		return nil, err
	}

	return r.newDynatraceClient(
		ctx,
		r,
		dk,
		tokens.APIToken().Value,
		tokens.PaasToken().Value,
		"prometheusmonitoring",
		k8senv.GetOperatorDTClientConnectionTimeout(ctx),
	)
}

// setPhase calculates the phase according to the conditions and the error.
// The error is only used when the conditions are empty and results in phases Deploying or Error.
//
// To accurately calculate the phase the reconcilers are expect to follow the following rules:
//   - Reconciliation successful: condition status True
//   - Reconciliation ongoing: condition status False/Unknown and reason Reconciling
//   - Reconciliation failed: anything that does not fit the above
func setPhase(dtp *prometheusmonitoring.PrometheusMonitoring, err error) error {
	if errors.Is(errDynaKubeNotFound, err) {
		if len(dtp.Status.Conditions) == 0 {
			dtp.Status.Phase = status.Deploying
		} else {
			dtp.Status.Phase = status.Error
		}

		return nil
	}

	if errors.Is(errDynaKubeNotReady, err) {
		dtp.Status.Phase = status.Deploying

		return nil
	}

	if errors.Is(errDataIngestTokenUnavailable, err) {
		dtp.Status.Phase = status.Error

		return nil
	}

	if len(dtp.Status.Conditions) == 0 {
		if err != nil {
			dtp.Status.Phase = status.Error
		} else {
			dtp.Status.Phase = status.Deploying
		}

		return err
	}

	phase := status.Running

	for _, c := range dtp.Status.Conditions {
		if c.Status != metav1.ConditionTrue {
			if c.Reason != status.ReasonReconciling {
				phase = status.Error

				break
			}

			phase = status.Deploying
		}
	}

	if err != nil && phase == status.Running {
		phase = status.Error
	}

	dtp.Status.Phase = phase

	return err
}

// SetupWithManager sets up the controller with the Manager.
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Add an index for the dynaKubeRef to allow using MatchingFields
	if err := mgr.GetFieldIndexer().IndexField(context.TODO(), &prometheusmonitoring.PrometheusMonitoring{}, "spec.dynaKubeRef", func(obj client.Object) []string {
		dtp, ok := obj.(*prometheusmonitoring.PrometheusMonitoring)
		if !ok {
			return nil
		}

		return []string{dtp.Spec.DynaKubeRef}
	}); err != nil {
		return fmt.Errorf("add dynaKubeRef index: %w", err)
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&prometheusmonitoring.PrometheusMonitoring{}).
		Owns(&appsv1.Deployment{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Service{}).
		Watches(
			&dynakube.DynaKube{},
			// Map requests from DynaKube to PrometheusMonitoring
			handler.EnqueueRequestsFromMapFunc(newPrometheusMonitoringFromDynaKubeMapper(mgr.GetClient())),
			// Filter out any DynaKube changes that are not relevant for PrometheusMonitoring
			builder.WithPredicates(newDynaKubeChangedPredicate()),
		).
		Named("prometheusmonitoring").
		Complete(r)
}

// Create a [handler.MapFunc] for DynaKubes that returns requests for PrometheusMonitoring objects whose spec.dynaKubeRef matches the DynaKube name.
func newPrometheusMonitoringFromDynaKubeMapper(c client.Client) handler.MapFunc {
	return func(ctx context.Context, obj client.Object) []ctrl.Request {
		_, log := logd.NewFromContext(ctx, "prometheusmonitoring-mapper")

		dk, ok := obj.(*dynakube.DynaKube)
		if !ok {
			log.Error(nil, fmt.Sprintf("expected DynaKube, but got %T", obj))

			return nil
		}

		dtpList := &prometheusmonitoring.PrometheusMonitoringList{}
		if err := c.List(ctx, dtpList, client.InNamespace(dk.Namespace), client.MatchingFields{"spec.dynaKubeRef": dk.Name}); err != nil {
			log.Error(err, "failed listing prometheusmonitoring objects", "dynaKubeRef", dk.Name)

			return nil
		}

		reqs := make([]ctrl.Request, len(dtpList.Items))
		for i := range dtpList.Items {
			reqs[i].NamespacedName = client.ObjectKeyFromObject(&dtpList.Items[i])
		}

		return reqs
	}
}

// Create [predicate.Funcs] that return true when DynaKube changed in a way that's relevant for a PrometheusMonitoring.
func newDynaKubeChangedPredicate() predicate.Funcs {
	return predicate.Funcs{
		CreateFunc: func(event.TypedCreateEvent[client.Object]) bool {
			return false
		},
		DeleteFunc: func(event.TypedDeleteEvent[client.Object]) bool {
			return true
		},
		UpdateFunc: func(e event.TypedUpdateEvent[client.Object]) bool {
			oldDK, _ := e.ObjectOld.(*dynakube.DynaKube)
			newDK, _ := e.ObjectNew.(*dynakube.DynaKube)

			if oldDK == nil || newDK == nil {
				// Don't need to drag a context or logger variable into this closure for this unexpected case.
				logd.Get().WithName("prometheusmonitoring-predicate").Error(nil, fmt.Sprintf("expected DynaKube, but got old:%T, new:%T", e.ObjectOld, e.ObjectNew))

				return false
			}

			return oldDK.Status.Phase != newDK.Status.Phase ||
				oldDK.Tokens() != newDK.Tokens() ||
				oldDK.Spec.TrustedCAs != newDK.Spec.TrustedCAs ||
				oldDK.GetDynatraceAPIRequestThreshold() != newDK.GetDynatraceAPIRequestThreshold() ||
				dynaKubeProxyChanged(oldDK, newDK) ||
				!maps.Equal(oldDK.Spec.ResourceAttributes, newDK.Spec.ResourceAttributes)
		},
		GenericFunc: func(event.TypedGenericEvent[client.Object]) bool {
			return false
		},
	}
}

func dynaKubeProxyChanged(oldDK, newDK *dynakube.DynaKube) bool {
	if (oldDK.Spec.Proxy != nil) != (newDK.Spec.Proxy != nil) {
		return true
	}

	if oldDK.Spec.Proxy != nil {
		return *oldDK.Spec.Proxy != *newDK.Spec.Proxy
	}

	return false
}
