package mapper

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	oneAgentNamespacesMonitoredConditionType           = "OneAgentNamespacesMonitored"
	metadataEnrichmentNamespacesMonitoredConditionType = "MetadataEnrichmentNamespacesMonitored"

	matchesFoundReason = "MatchesFound"
	noMatchesReason    = "NoMatches"

	maxNamesInMsg = 10
)

func setNamespacesMonitoredSelectorCondition(conditions *[]metav1.Condition, selectorType string, configured bool, names []string) {
	var condType string

	switch selectorType {
	case "OneAgent":
		condType = oneAgentNamespacesMonitoredConditionType
	case "MetadataEnrichment":
		condType = metadataEnrichmentNamespacesMonitoredConditionType
	}

	log.Info("namespaces monitored",
		"selector", selectorType,
		"count (at most 10 are displayed)", len(names),
		"namespaces", names,
	)

	cond := metav1.Condition{Type: condType}

	switch {
	case !configured:
		_ = meta.RemoveStatusCondition(conditions, condType)

		return
	case len(names) == 0:
		cond.Status = metav1.ConditionFalse
		cond.Reason = noMatchesReason
		cond.Message = "0 namespaces match"
	default:
		cond.Status = metav1.ConditionTrue
		cond.Reason = matchesFoundReason
		msg := formatMatchMessage(names, maxNamesInMsg)
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
