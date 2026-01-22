package rules

import (
	"context"
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/metadataenrichment"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/settings"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8sconditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	"k8s.io/apimachinery/pkg/api/meta"
)

type Reconciler struct {
	dtc          settings.APIClient
	dk           *dynakube.DynaKube
	timeProvider *timeprovider.Provider
}

type ReconcilerBuilder func(dtc settings.APIClient, dk *dynakube.DynaKube) controllers.Reconciler

func NewReconciler(dtc settings.APIClient, dk *dynakube.DynaKube) controllers.Reconciler {
	return &Reconciler{
		dtc:          dtc,
		dk:           dk,
		timeProvider: timeprovider.New(),
	}
}

func (r *Reconciler) Reconcile(ctx context.Context) error {
	if !r.dk.MetadataEnrichment().IsEnabled() && !r.dk.OneAgent().IsAppInjectionNeeded() && !r.dk.OTLPExporterConfiguration().IsEnabled() {
		if meta.FindStatusCondition(*r.dk.Conditions(), conditionType) == nil {
			return nil
		}

		r.dk.Status.MetadataEnrichment.Rules = nil
		meta.RemoveStatusCondition(r.dk.Conditions(), conditionType)

		return nil
	}

	if !k8sconditions.IsOutdated(r.timeProvider, r.dk, conditionType) {
		return nil
	}

	k8sconditions.SetStatusOutdated(r.dk.Conditions(), conditionType, "Metadata-enrichment rules are outdated in the status")

	if !k8sconditions.IsOptionalScopeAvailable(r.dk, dtclient.ConditionTypeAPITokenSettingsRead) {
		log.Info("metadata-enrichment rules are not set in the status because the optional scope is not available", "scope", dtclient.TokenScopeSettingsRead)
		k8sconditions.SetOptionalScopeMissing(r.dk.Conditions(), conditionType, "Metadata-enrichment rules are not set in the status because the optional 'settings.read' scope is not available")

		return nil
	}

	rules, err := r.getEnrichmentRules(ctx)
	if err != nil {
		return err
	}

	r.dk.Status.MetadataEnrichment.Rules = rules
	k8sconditions.SetStatusUpdated(r.dk.Conditions(), conditionType, "Metadata-enrichment rules are up-to-date in the status")
	log.Info("update rules in the status", "len(rules)", len(rules))

	return nil
}

func (r *Reconciler) getEnrichmentRules(ctx context.Context) ([]metadataenrichment.Rule, error) {
	rules, err := r.dtc.GetRules(ctx, r.dk.Status.KubeSystemUUID, r.dk.Status.KubernetesClusterMEID)
	if err != nil {
		k8sconditions.SetDynatraceAPIError(r.dk.Conditions(), conditionType, err)

		return nil, fmt.Errorf("error trying to check if rules exist: %w", err)
	}

	return rules, nil
}
