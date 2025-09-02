package logmonsettings

import (
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	alreadyExistReason = "AlreadyExist"
	skippedReason      = "Skipped"
	errorReason        = "Error"
	createdReason      = "Created"

	ConditionType = "LogMonitoringSettings"
)

func setCreatedCondition(conditions *[]metav1.Condition) {
	condition := metav1.Condition{
		Type:    ConditionType,
		Status:  metav1.ConditionTrue,
		Reason:  createdReason,
		Message: "LogMonitoring settings have been created.",
	}
	_ = meta.SetStatusCondition(conditions, condition)
}

func setAlreadyExistsCondition(conditions *[]metav1.Condition) {
	condition := metav1.Condition{
		Type:    ConditionType,
		Status:  metav1.ConditionTrue,
		Reason:  alreadyExistReason,
		Message: "LogMonitoring settings already exist, will not create new ones.",
	}
	_ = meta.SetStatusCondition(conditions, condition)
}

func setErrorCondition(conditions *[]metav1.Condition, message string) {
	condition := metav1.Condition{
		Type:    ConditionType,
		Status:  metav1.ConditionFalse,
		Reason:  errorReason,
		Message: "LogMonitoring settings creation was skipped: " + message,
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
