package rules

import (
	"context"
	"errors"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/metadataenrichment"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8sconditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	"k8s.io/apimachinery/pkg/api/meta"
)

type Reconciler struct {
	dtc          dtclient.Client
	dk           *dynakube.DynaKube
	timeProvider *timeprovider.Provider
}

type ReconcilerBuilder func(dtc dtclient.Client, dk *dynakube.DynaKube) controllers.Reconciler

func NewReconciler(dtc dtclient.Client, dk *dynakube.DynaKube) controllers.Reconciler {
	return &Reconciler{
		dtc:          dtc,
		dk:           dk,
		timeProvider: timeprovider.New(),
	}
}

func (r *Reconciler) Reconcile(ctx context.Context) error {
	if !r.dk.MetadataEnrichment().IsEnabled() && !r.dk.FF().IsNodeImagePull() && !r.dk.OTLPExporterConfiguration().IsEnabled() {
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
	rulesResponse, err := r.dtc.GetRulesSettings(ctx, r.dk.Status.KubeSystemUUID, r.dk.Status.KubernetesClusterMEID)
	if err != nil {
		k8sconditions.SetDynatraceAPIError(r.dk.Conditions(), conditionType, err)

		return nil, errors.Join(err, errors.New("error trying to check if rules exist"))
	}

	var rules []metadataenrichment.Rule
	// Shouldn't be necessary, because we only get a single item back from the API, but still, its more "complete" this way
	for _, item := range rulesResponse.Items {
		rules = append(rules, item.Value.Rules...)
	}

	return rules, nil
}
