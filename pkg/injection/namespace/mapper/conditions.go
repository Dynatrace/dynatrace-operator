// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package mapper

import (
	"context"
	"fmt"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	matchesFoundReason    = "MatchesFound"
	noMatchesReason       = "NoMatches"
	invalidSelectorReason = "InvalidSelector"

	maxNamesInMsg = 10
)

type conditionType string

func (c conditionType) String() string {
	return string(c)
}

const (
	oneAgentNamespacesMonitoredConditionType           conditionType = "OneAgentNamespacesMonitored"
	metadataEnrichmentNamespacesMonitoredConditionType conditionType = "MetadataEnrichmentNamespacesMonitored"
	otlpExporterNamespacesMonitoredConditionType       conditionType = "OTLPExporterConfigurationNamespacesMonitored"
)

// selectorMatchStatus bundles the inputs setNamespacesMonitoredSelectorCondition needs to decide a
// condition's Status/Reason/Message for one feature (OneAgent, MetadataEnrichment, or OTLP).
type selectorMatchStatus struct {
	configured bool
	invalid    bool
	names      []string
}

func setNamespacesMonitoredSelectorCondition(ctx context.Context, conditions *[]metav1.Condition, condType conditionType, status selectorMatchStatus) {
	log := logd.FromContext(ctx)
	log.Info("namespaces monitored",
		"condition", condType,
		"count", len(status.names),
		"namespaces (max 10 listed)", status.names,
		"selectorInvalid", status.invalid,
	)

	cond := metav1.Condition{Type: condType.String()}

	switch {
	case !status.configured:
		_ = meta.RemoveStatusCondition(conditions, condType.String())

		return
	case status.invalid:
		cond.Status = metav1.ConditionFalse
		cond.Reason = invalidSelectorReason
		cond.Message = "namespaceSelector is invalid; treated as matching no namespaces until fixed"
	case len(status.names) == 0:
		cond.Status = metav1.ConditionFalse
		cond.Reason = noMatchesReason
		cond.Message = "0 namespaces match"
	default:
		cond.Status = metav1.ConditionTrue
		cond.Reason = matchesFoundReason
		msg := formatMatchMessage(status.names, maxNamesInMsg)
		cond.Message = msg
	}

	cond.LastTransitionTime = metav1.Now()
	_ = meta.SetStatusCondition(conditions, cond)
}

func formatMatchMessage(names []string, limit int) string {
	if len(names) == 0 {
		return "no namespaces match"
	}

	if len(names) > limit {
		return fmt.Sprintf("%d namespaces match: %s (at most %d are displayed)", len(names), strings.Join(names[:limit], ", "), limit)
	}

	return fmt.Sprintf("%d namespaces match: %s", len(names), strings.Join(names, ", "))
}
