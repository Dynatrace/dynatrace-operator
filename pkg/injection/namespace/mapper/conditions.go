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
	matchesFoundReason = "MatchesFound"
	noMatchesReason    = "NoMatches"

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

func setNamespacesMonitoredSelectorCondition(ctx context.Context, conditions *[]metav1.Condition, condType conditionType, configured bool, names []string) {
	log := logd.FromContext(ctx)
	log.Info("namespaces monitored",
		"condition", condType,
		"count", len(names),
		"namespaces (max 10 listed)", names,
	)

	cond := metav1.Condition{Type: condType.String()}

	switch {
	case !configured:
		_ = meta.RemoveStatusCondition(conditions, condType.String())

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
