// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

// Package kubemon reconciles the dedicated Kubernetes Monitoring operand.
//
// Non-obvious behavior:
//   - A single condition (KubernetesMonitoringAvailable) is owned by this orchestrator.
//   - kubemon uses a single Kubernetes client dependency; caller chooses the concrete
//     implementation (cached or direct)
//   - Condition mapping is based on sub-reconciler errors:
//   - nil => True/Available
//   - converging sentinels (rollout, connection info) => False/Reconciling
//   - all other errors => False/Error
//   - Cleanup keeps running while disabled until all owned resources are deleted; only then
//     the condition is removed.
package kubemon

import (
	"context"
	"errors"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	kubemonapi "github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/kubemon"
	agclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/activegate"
	kubemonauthtoken "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/kubemon/authtoken"
	kubemonconnectioninfo "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/kubemon/connectioninfo"
	kubemonstatefulset "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/kubemon/statefulset"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/token"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8senv"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8sstatefulset"
	pkgerrors "github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	reasonAvailable   = "Available"
	reasonError       = "Error"
	reasonReconciling = "Reconciling"

	messageAvailable = "kubernetes monitoring resources are ready"
)

type connectionInfoReconciler interface {
	Reconcile(ctx context.Context, agClient agclient.Client, dk *dynakube.DynaKube) error
}

type authTokenReconciler interface {
	Reconcile(ctx context.Context, agClient agclient.Client, dk *dynakube.DynaKube) error
}

type statefulsetReconciler interface {
	Reconcile(ctx context.Context, dk *dynakube.DynaKube) error
}

// Reconciler orchestrates the kubemon operand. Sub-reconciler fields are interfaces so they
// can be mocked in tests.
type Reconciler struct {
	connectionInfoReconciler connectionInfoReconciler
	authTokenReconciler      authTokenReconciler
	statefulsetReconciler    statefulsetReconciler
}

func NewReconciler(kubeClient client.Client) *Reconciler {
	return &Reconciler{
		connectionInfoReconciler: kubemonconnectioninfo.NewReconciler(kubeClient),
		authTokenReconciler:      kubemonauthtoken.NewReconciler(kubeClient, kubemonauthtoken.DefaultRotationInterval),
		statefulsetReconciler:    kubemonstatefulset.NewReconciler(kubeClient),
	}
}

// Reconcile is the operand entry point called by the parent DynaKube controller.
// Sub-reconcilers mutate dk.Status.KubernetesMonitoring.* and dk.Status.Conditions in-memory;
// the parent controller persists status changes via deferred Status().Update().
func (r *Reconciler) Reconcile(ctx context.Context, dk *dynakube.DynaKube, agClient agclient.Client, _ token.Tokens) (err error) {
	ctx, log := logd.NewFromContext(ctx, "dynakube-kubemon")

	// Temporary gate, to be removed once kubemon is complete
	if !k8senv.IsKubemonOperandEnabled() {
		log.Debug("kubemon gate not enabled, skipping")

		return nil
	}

	log.Debug("reconciling kubernetes monitoring")

	defer func() { r.reconcileCondition(dk, err) }()

	if err = r.connectionInfoReconciler.Reconcile(ctx, agClient, dk); err != nil {
		return err
	}

	if err = r.authTokenReconciler.Reconcile(ctx, agClient, dk); err != nil {
		return err
	}

	if err = r.statefulsetReconciler.Reconcile(ctx, dk); err != nil {
		return err
	}

	log.Debug("reconciled kubernetes monitoring")

	return nil
}

func (r *Reconciler) reconcileCondition(dk *dynakube.DynaKube, err error) {
	if !dk.KubernetesMonitoring().IsEnabled() {
		meta.RemoveStatusCondition(dk.Conditions(), kubemonapi.KubeMonAvailableConditionType)

		return
	}

	condition := metav1.Condition{
		Type: kubemonapi.KubeMonAvailableConditionType,
	}

	switch {
	case err == nil:
		condition.Status = metav1.ConditionTrue
		condition.Reason = reasonAvailable
		condition.Message = messageAvailable
	case errors.Is(err, k8sstatefulset.ErrRolloutInProgress),
		errors.Is(err, kubemonconnectioninfo.ErrConnectionInfoNotReady):
		condition.Status = metav1.ConditionFalse
		condition.Reason = reasonReconciling
		condition.Message = pkgerrors.Cause(err).Error()
	default:
		condition.Status = metav1.ConditionFalse
		condition.Reason = reasonError
		condition.Message = pkgerrors.Cause(err).Error()
	}

	_ = meta.SetStatusCondition(dk.Conditions(), condition)
}
