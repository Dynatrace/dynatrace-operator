package rules

import (
	"context"
	"errors"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
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
	if !r.dk.MetadataEnrichmentEnabled() {
		if meta.FindStatusCondition(*r.dk.Conditions(), conditionType) == nil {
			return nil
		}

		r.dk.Status.MetadataEnrichment.Rules = nil
		meta.RemoveStatusCondition(r.dk.Conditions(), conditionType)

		return nil
	}

	if !conditions.IsOutdated(r.timeProvider, r.dk, conditionType) {
		return nil
	}

	conditions.SetStatusOutdated(r.dk.Conditions(), conditionType, "Metadata-enrichment rules are outdated in the status")

	rules, err := r.getEnrichmentRules(ctx)
	if err != nil {
		return err
	}

	r.dk.Status.MetadataEnrichment.Rules = rules
	conditions.SetStatusUpdated(r.dk.Conditions(), conditionType, "Metadata-enrichment rules are up-to-date in the status")
	log.Info("update rules in the status", "len(rules)", len(rules))

	return nil
}

func (r *Reconciler) getEnrichmentRules(ctx context.Context) ([]dynakube.EnrichmentRule, error) {
	rulesResponse, err := r.dtc.GetRulesSettings(ctx, r.dk.Status.KubeSystemUUID, r.dk.Status.KubernetesClusterMEID)
	if err != nil {
		conditions.SetDynatraceApiError(r.dk.Conditions(), conditionType, err)

		return nil, errors.Join(err, errors.New("error trying to check if rules exist"))
	}

	var rules []dynakube.EnrichmentRule
	// Shouldn't be necessary, because we only get a single item back from the API, but still, its more "complete" this way
	for _, item := range rulesResponse.Items {
		rules = append(rules, item.Value.Rules...)
	}

	return rules, nil
}
