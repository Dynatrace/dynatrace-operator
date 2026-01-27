package kspmsettings

import (
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	existsReason   = "Exists"
	outdatedReason = "Outdated"
	skippedReason  = "Skipped"
	errorReason    = "Error"

	conditionType = "KSPMSettings"
)

func setExistsCondition(conditions *[]metav1.Condition) {
	condition := metav1.Condition{
		Type:    conditionType,
		Status:  metav1.ConditionTrue,
		Reason:  existsReason,
		Message: "KSPM settings exist.",
	}
	_ = meta.SetStatusCondition(conditions, condition)
}

func setErrorCondition(conditions *[]metav1.Condition, message string) {
	condition := metav1.Condition{
		Type:    conditionType,
		Status:  metav1.ConditionFalse,
		Reason:  errorReason,
		Message: "KSPM settings creation encountered an error: " + message,
	}
	_ = meta.SetStatusCondition(conditions, condition)
}

func setSkippedCondition(conditions *[]metav1.Condition, message string) {
	condition := metav1.Condition{
		Type:    conditionType,
		Status:  metav1.ConditionFalse,
		Reason:  skippedReason,
		Message: "KSPM settings creation was skipped: " + message,
	}
	_ = meta.SetStatusCondition(conditions, condition)
}
