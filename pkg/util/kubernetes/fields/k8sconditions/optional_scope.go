package k8sconditions

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	OptionalScopeMissingReason = "ScopeMissing"
	OptionalScopePresentReason = "ScopePresent"
)

func IsOptionalScopeAvailable(dk *dynakube.DynaKube, conditionType string) bool {
	condition := meta.FindStatusCondition(*dk.Conditions(), conditionType)
	if condition == nil {
		return false
	}

	return condition.Status == metav1.ConditionTrue
}

func SetOptionalScopeMissing(conditions *[]metav1.Condition, conditionType, msg string) {
	condition := metav1.Condition{
		Type:    conditionType,
		Status:  metav1.ConditionFalse,
		Reason:  OptionalScopeMissingReason,
		Message: msg,
	}
	_ = meta.SetStatusCondition(conditions, condition)
}

func SetOptionalScopeAvailable(conditions *[]metav1.Condition, conditionType string, msg string) {
	tokenCondition := metav1.Condition{
		Type:    conditionType,
		Status:  metav1.ConditionTrue,
		Reason:  OptionalScopePresentReason,
		Message: msg,
	}
	_ = meta.SetStatusCondition(conditions, tokenCondition)
}
