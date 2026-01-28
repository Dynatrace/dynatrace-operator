package logmonsettings

import (
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	existsReason  = "Exists"
	skippedReason = "Skipped"
	errorReason   = "Error"

	ConditionType = "LogMonitoringSettings"
)

func setExistsCondition(conditions *[]metav1.Condition) {
	condition := metav1.Condition{
		Type:    ConditionType,
		Status:  metav1.ConditionTrue,
		Reason:  existsReason,
		Message: "LogMonitoring settings exist.",
	}
	_ = meta.SetStatusCondition(conditions, condition)
}

func setErrorCondition(conditions *[]metav1.Condition, message string) {
	condition := metav1.Condition{
		Type:    ConditionType,
		Status:  metav1.ConditionFalse,
		Reason:  errorReason,
		Message: "LogMonitoring settings creation has encountered an error: " + message,
	}
	_ = meta.SetStatusCondition(conditions, condition)
}

func setSkippedCondition(conditions *[]metav1.Condition, message string) {
	condition := metav1.Condition{
		Type:    ConditionType,
		Status:  metav1.ConditionFalse,
		Reason:  skippedReason,
		Message: "LogMonitoring settings creation was skipped: " + message,
	}
	_ = meta.SetStatusCondition(conditions, condition)
}
