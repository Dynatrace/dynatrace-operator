package k8sconditions

import (
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ServiceCreatedReason    = "ServiceCreated"
	ServiceGenerationFailed = "ServiceGenerationFailed"
)

func SetServiceCreated(conditions *[]metav1.Condition, conditionType, name string) {
	condition := metav1.Condition{
		Type:    conditionType,
		Status:  metav1.ConditionTrue,
		Reason:  ServiceCreatedReason,
		Message: appendCreatedSuffix(name),
	}
	_ = meta.SetStatusCondition(conditions, condition)
}

func SetServiceGenFailed(conditions *[]metav1.Condition, conditionType string, err error) {
	condition := metav1.Condition{
		Type:    conditionType,
		Status:  metav1.ConditionFalse,
		Reason:  ServiceGenerationFailed,
		Message: "Failed to generate service: " + err.Error(),
	}
	_ = meta.SetStatusCondition(conditions, condition)
}
