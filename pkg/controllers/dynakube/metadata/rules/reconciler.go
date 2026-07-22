// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package rules

import (
	"context"
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/metadataenrichment"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/settings"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/token"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8sconditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/tenant/optionalscope"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	"k8s.io/apimachinery/pkg/api/meta"
)

type Reconciler struct {
	timeProvider *timeprovider.Provider
}

func NewReconciler() *Reconciler {
	return &Reconciler{
		timeProvider: timeprovider.New(),
	}
}

func (r *Reconciler) Reconcile(ctx context.Context, dtClient settings.Client, dk *dynakube.DynaKube) error {
	ctx, log := logd.NewFromContext(ctx, "metadata-enrichment-rules")

	if !dk.MetadataEnrichment().IsEnabled() && !dk.OneAgent().IsAppInjectionNeeded() && !dk.OTLPExporterConfiguration().IsEnabled() {
		if meta.FindStatusCondition(*dk.Conditions(), conditionType) == nil {
			return nil
		}

		dk.Status.MetadataEnrichment.Rules = nil
		meta.RemoveStatusCondition(dk.Conditions(), conditionType)

		return nil
	}

	if !k8sconditions.IsOutdated(r.timeProvider, dk, conditionType) {
		return nil
	}

	k8sconditions.SetStatusOutdated(dk.Conditions(), conditionType, "Metadata-enrichment rules are outdated in the status")

	if !optionalscope.IsAvailable(dk, token.ScopeSettingsRead) {
		log.Info("metadata-enrichment rules are not set in the status because the optional scope is not available", "scope", token.ScopeSettingsRead)
		k8sconditions.SetOptionalScopeMissing(dk.Conditions(), conditionType, "Metadata-enrichment rules are not set in the status because the optional 'settings.read' scope is not available")

		return nil
	}

	rules, err := r.getEnrichmentRules(ctx, dtClient, dk)
	if err != nil {
		if !core.IsForbidden(err) {
			return err
		}

		msg := "provided token cannot read metadata-enrichment rules due to missing scopes"
		log.Info(msg)
		k8sconditions.SetOptionalScopeMissing(dk.Conditions(), conditionType, msg)

		return nil
	}

	dk.Status.MetadataEnrichment.Rules = rules
	k8sconditions.SetStatusUpdated(dk.Conditions(), conditionType, "Metadata-enrichment rules are up-to-date in the status")
	log.Info("update rules in the status", "len(rules)", len(rules))

	return nil
}

func (r *Reconciler) getEnrichmentRules(ctx context.Context, dtClient settings.Client, dk *dynakube.DynaKube) ([]metadataenrichment.Rule, error) {
	rules, err := dtClient.GetRules(ctx, dk.Status.KubeSystemUUID, dk.Status.KubernetesClusterMEID)
	if err != nil {
		k8sconditions.SetDynatraceAPIError(dk.Conditions(), conditionType, err)

		return nil, fmt.Errorf("error trying to check if rules exist: %w", err)
	}

	return rules, nil
}
