package conditions

import (
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	StatusUpdatedReason  = "StatusUpdated"
	StatusOutdatedReason = "StatusOutdated"

	OptionalScopeReason        = "OptionalScope"
	OptionalScopePresentReason = "ScopePresent"
)

func SetStatusUpdated(conditions *[]metav1.Condition, conditionType, msg string) {
	condition := metav1.Condition{
		Type:    conditionType,
		Status:  metav1.ConditionTrue,
		Reason:  StatusUpdatedReason,
		Message: msg,
	}
	_ = meta.SetStatusCondition(conditions, condition)
}

func SetStatusOutdated(conditions *[]metav1.Condition, conditionType, msg string) {
	condition := metav1.Condition{
		Type:    conditionType,
		Status:  metav1.ConditionFalse,
		Reason:  StatusOutdatedReason,
		Message: msg,
	}
	_ = meta.SetStatusCondition(conditions, condition)
}

func SetStatusOptionalScopeMissing(conditions *[]metav1.Condition, conditionType, msg string) {
	condition := metav1.Condition{
		Type:    conditionType,
		Status:  metav1.ConditionFalse,
		Reason:  OptionalScopeReason,
		Message: msg,
	}
	_ = meta.SetStatusCondition(conditions, condition)
}

func SetOptionalScopeAvailable(conditions *[]metav1.Condition, conditionType string, scope string) {
	tokenCondition := metav1.Condition{
		Type:    conditionType,
		Status:  metav1.ConditionTrue,
		Reason:  OptionalScopePresentReason,
		Message: scope + " is available",
	}
	_ = meta.SetStatusCondition(conditions, tokenCondition)
}

func SetOptionalScopeMissing(conditions *[]metav1.Condition, conditionType string, scope string) {
	tokenCondition := metav1.Condition{
		Type:    conditionType,
		Status:  metav1.ConditionFalse,
		Reason:  OptionalScopeReason,
		Message: scope + " is not available, some features may not work",
	}
	_ = meta.SetStatusCondition(conditions, tokenCondition)
}
