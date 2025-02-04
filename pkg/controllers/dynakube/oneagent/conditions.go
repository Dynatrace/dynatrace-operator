package oneagent

import (
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	oaConditionType = "OneAgentDaemonSet"

	daemonSetCreatedReason          = "DaemonSetCreated"
	daemonSetGenerationFailedReason = "DaemonSetGenerationFailed"
)

func setDaemonSetCreatedCondition(conditions *[]metav1.Condition) {
	condition := metav1.Condition{
		Type:    oaConditionType,
		Status:  metav1.ConditionTrue,
		Reason:  daemonSetCreatedReason,
		Message: "The OneAgent DaemonSet was created successfully.",
	}
	_ = meta.SetStatusCondition(conditions, condition)
}

func setDaemonSetGenerationFailedCondition(conditions *[]metav1.Condition, err error) {
	condition := metav1.Condition{
		Type:    oaConditionType,
		Status:  metav1.ConditionFalse,
		Reason:  daemonSetGenerationFailedReason,
		Message: "Failed to generate the DaemonSet configuration, error: " + err.Error(),
	}
	_ = meta.SetStatusCondition(conditions, condition)
}
