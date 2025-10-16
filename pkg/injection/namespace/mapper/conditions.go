package mapper

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	namespacesMonitoredConditionType = "NamespacesMonitored"

	oneAgentNamespacesMonitoredConditionType           = "OneAgentNamespacesMonitored"
	metadataEnrichmentNamespacesMonitoredConditionType = "MetadataEnrichmentNamespacesMonitored"

	matchesFoundReason          = "MatchesFound"
	noMatchesReason             = "NoMatches"
	selectorNotConfiguredReason = "SelectorNotConfigured"

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

func updateCollectedNamespacesMonitoredCondition(conditions *[]metav1.Condition) {
	oa := meta.FindStatusCondition(*conditions, oneAgentNamespacesMonitoredConditionType)
	me := meta.FindStatusCondition(*conditions, metadataEnrichmentNamespacesMonitoredConditionType)

	if oa == nil && me == nil {
		_ = meta.RemoveStatusCondition(conditions, namespacesMonitoredConditionType)

		return
	}

	collected := metav1.Condition{
		Type:               namespacesMonitoredConditionType,
		LastTransitionTime: metav1.Now(),
	}

	if (oa != nil && oa.Status == metav1.ConditionTrue) || (me != nil && me.Status == metav1.ConditionTrue) {
		collected.Status = metav1.ConditionTrue
		collected.Reason = matchesFoundReason
		collected.Message = "Either no selector is configured or at least one selector matches one or more namespaces"
		_ = meta.SetStatusCondition(conditions, collected)

		return
	}

	collected.Status = metav1.ConditionFalse
	collected.Reason = noMatchesReason
	collected.Message = "No namespaces match the configured selectors"
	_ = meta.SetStatusCondition(conditions, collected)
}

func formatMatchMessage(names []string, limit int) string {
	if len(names) == 0 {
		return "no namespaces match"
	}

	if len(names) > limit {
		return fmt.Sprintf("%d namespaces match: %s (at most %d shown)", len(names), strings.Join(names[:limit], ", "), limit)
	}

	return fmt.Sprintf("%d namespaces match: %s", len(names), strings.Join(names, ", "))
}
