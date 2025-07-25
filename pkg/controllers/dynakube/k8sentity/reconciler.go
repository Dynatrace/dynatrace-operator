// Package k8sentity provides functionality for reconciling Kubernetes Cluster Monitored Entities for a given DynaKube.
// The only purpose of this package is to look for (and not create) an already existing Kubernetes Cluster Monitored Entity in the Dynatrace Environment and store relevant info in the DynaKube's Status.
//
// A Kubernetes Cluster Monitored Entity (example: KUBERNETES_CLUSTER-A1234567BCD8EFGH) is calculated in the Dynatrace Environment.
// - This happens when certain Setting (`builtin:cloud.kubernetes`) is created
//   - Looking at this Setting via the API we can determine the Kubernetes Cluster Monitored Entity
//
// This ME(Monitored Entity) is an important configuration, yet optional, for most Dynatrace Components.
// - If the Operator provides the ID and Name of the ME when possible, then it reduces the computational cost on ingest.
package k8sentity

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
)

type Reconciler struct {
	dk           *dynakube.DynaKube
	dtClient     dynatrace.Client
	timeProvider *timeprovider.Provider
}

type ReconcilerBuilder func(dtClient dynatrace.Client, dk *dynakube.DynaKube) controllers.Reconciler

func NewReconciler( //nolint
	dtClient dynatrace.Client,
	dk *dynakube.DynaKube,
) controllers.Reconciler {
	return &Reconciler{
		dk:           dk,
		dtClient:     dtClient,
		timeProvider: timeprovider.New(),
	}
}

func (r *Reconciler) Reconcile(ctx context.Context) error {
	log.Info("start reconciling Kubernetes Cluster MEID")

	if !conditions.IsOutdated(r.timeProvider, r.dk, meIDConditionType) {
		log.Info("Kubernetes Cluster MEID not outdated, skipping reconciliation")

		return nil
	}

	conditions.SetStatusOutdated(r.dk.Conditions(), meIDConditionType, "Kubernetes Cluster MEID is outdated in the status")

	if !conditions.IsOptionalScopeAvailable(r.dk, dynatrace.ConditionTypeAPITokenSettingsRead) {
		msg := dynatrace.TokenScopeSettingsRead + " optional scope not available"
		log.Info(msg)
		conditions.SetScopeMissing(r.dk.Conditions(), meIDConditionType, msg)

		return nil
	}

	k8sEntity, err := r.dtClient.GetK8sClusterME(ctx, r.dk.Status.KubeSystemUUID)
	if err != nil {
		log.Info("failed to retrieve MEs")

		return err
	}

	if k8sEntity.ID == "" {
		log.Info("no MEs found, no kubernetesClusterMEID will be set in the dynakube status")

		return nil
	}

	r.dk.Status.KubernetesClusterMEID = k8sEntity.ID
	r.dk.Status.KubernetesClusterName = k8sEntity.Name
	conditions.SetStatusUpdated(r.dk.Conditions(), meIDConditionType, "Kubernetes Cluster MEID is up to date")

	log.Info("kubernetesClusterMEID set in dynakube status, done reconciling", "kubernetesClusterMEID", r.dk.Status.KubernetesClusterMEID)

	return nil
}
