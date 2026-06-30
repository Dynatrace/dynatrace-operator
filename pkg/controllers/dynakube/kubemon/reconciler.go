// Package kubemon reconciles the dedicated Kubernetes Monitoring operand.
//
// Non-obvious behavior:
//   - A single condition (KubernetesMonitoringAvailable) is owned by this orchestrator.
//   - kubemon uses a single Kubernetes client dependency; caller chooses the concrete
//     implementation (cached or direct)
//   - Condition mapping is based on sub-reconciler errors:
//   - nil => True/Available
//   - persistent sentinels => False/Error
//   - all other errors => Unknown/Reconciling
//   - Cleanup keeps running while disabled until all owned resources are deleted; only then
//     the condition is removed.
package kubemon

import (
	"context"
	"errors"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	kubemonapi "github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/kubemon"
	agclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/activegate"
	kubemonconnectioninfo "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/kubemon/connectioninfo"
	kubemonstatefulset "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/kubemon/statefulset"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/token"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8sstatefulset"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	reasonAvailable   = "Available"
	reasonError       = "Error"
	reasonReconciling = "Reconciling"
)

type connectionInfoReconciler interface {
	Reconcile(ctx context.Context, agClient agclient.Client, dk *dynakube.DynaKube) error
}

type statefulsetReconciler interface {
	Reconcile(ctx context.Context, dk *dynakube.DynaKube) error
}

// Reconciler orchestrates the kubemon operand. Sub-reconciler fields are interfaces so they
// can be mocked in tests.
type Reconciler struct {
	connectionInfoReconciler connectionInfoReconciler
	statefulsetReconciler    statefulsetReconciler
}

func NewReconciler(kubeClient client.Client) *Reconciler {
	return &Reconciler{
		connectionInfoReconciler: kubemonconnectioninfo.NewReconciler(kubeClient),
		statefulsetReconciler:    kubemonstatefulset.NewReconciler(kubeClient),
	}
}

// Reconcile is the operand entry point called by the parent DynaKube controller.
// Sub-reconcilers mutate dk.Status.KubernetesMonitoring.* and dk.Status.Conditions in-memory;
// the parent controller persists status changes via deferred Status().Update().
func (r *Reconciler) Reconcile(ctx context.Context, dk *dynakube.DynaKube, agClient agclient.Client, _ token.Tokens) error {
	ctx, log := logd.NewFromContext(ctx, "dynakube-kubemon")

	// Fast-path guard: never created, nothing to converge.
	if !dk.KubernetesMonitoring().IsEnabled() && !hasCondition(dk) {
		log.Debug("kubemon not enabled, skipping")

		return nil
	}

	log.Debug("reconciling kubernetes monitoring")

	if err := r.connectionInfoReconciler.Reconcile(ctx, agClient, dk); err != nil {
		r.setConditionByError(dk, err)

		return err
	}

	if err := r.statefulsetReconciler.Reconcile(ctx, dk); err != nil {
		r.setConditionByError(dk, err)

		return err
	}

	r.setAvailableCondition(dk)

	log.Debug("reconciled kubernetes monitoring")

	return nil
}

func hasCondition(dk *dynakube.DynaKube) bool {
	return meta.FindStatusCondition(*dk.Conditions(), kubemonapi.KubeMonAvailableConditionType) != nil
}

func (r *Reconciler) setAvailableCondition(dk *dynakube.DynaKube) {
	if !dk.KubernetesMonitoring().IsEnabled() {
		meta.RemoveStatusCondition(dk.Conditions(), kubemonapi.KubeMonAvailableConditionType)

		return
	}

	cond := metav1.Condition{
		Type:    kubemonapi.KubeMonAvailableConditionType,
		Status:  metav1.ConditionTrue,
		Reason:  reasonAvailable,
		Message: "kubernetes monitoring resources are ready",
	}

	_ = meta.SetStatusCondition(dk.Conditions(), cond)
}

func (r *Reconciler) setConditionByError(dk *dynakube.DynaKube, err error) {
	condition := metav1.Condition{
		Type: kubemonapi.KubeMonAvailableConditionType,
	}

	switch {
	case isPersistent(err):
		condition.Status = metav1.ConditionFalse
		condition.Reason = reasonError
		condition.Message = err.Error()
	case errors.Is(err, k8sstatefulset.ErrRolloutInProgress):
		condition.Status = metav1.ConditionUnknown
		condition.Reason = reasonReconciling
		condition.Message = "kubernetes monitoring rollout is still in progress"
	default:
		condition.Status = metav1.ConditionUnknown
		condition.Reason = reasonReconciling
		condition.Message = "kubernetes monitoring reconciliation is in progress"
	}

	_ = meta.SetStatusCondition(dk.Conditions(), condition)
}

func isPersistent(err error) bool {
	return errors.Is(err, kubemonstatefulset.ErrImageRequired)
}
