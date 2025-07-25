package conditions

import (
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	DataIngestTokenMissing = "DataIngestTokenMissing"
	ScopeMissingReason     = "ScopeMissing"
)

func SetDataIngestTokenMissing(conditions *[]metav1.Condition, conditionType string, msg string) {
	condition := metav1.Condition{
		Type:    conditionType,
		Status:  metav1.ConditionFalse,
		Reason:  DataIngestTokenMissing,
		Message: msg,
	}
	_ = meta.SetStatusCondition(conditions, condition)
}

func SetScopeMissing(conditions *[]metav1.Condition, conditionType, msg string) {
	condition := metav1.Condition{
		Type:    conditionType,
		Status:  metav1.ConditionFalse,
		Reason:  ScopeMissingReason,
		Message: msg,
	}
	_ = meta.SetStatusCondition(conditions, condition)
}
