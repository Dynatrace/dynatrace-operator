package k8sconditions

import (
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	SecretCreatedReason          = "SecretCreated"
	SecretUpdatedReason          = "SecretUpdated"
	SecretCreatedOrUpdatedReason = "SecretCreatedOrUpdated"
	SecretOutdatedReason         = "SecretOutdated"
	SecretGenerationFailed       = "SecretGenerationFailed"
)

func SetSecretCreated(conditions *[]metav1.Condition, conditionType, name string) {
	condition := metav1.Condition{
		Type:    conditionType,
		Status:  metav1.ConditionTrue,
		Reason:  SecretCreatedReason,
		Message: appendCreatedSuffix(name),
	}
	_ = meta.SetStatusCondition(conditions, condition)
}

func SetSecretCreatedOrUpdated(conditions *[]metav1.Condition, conditionType, name string) {
	condition := metav1.Condition{
		Type:    conditionType,
		Status:  metav1.ConditionTrue,
		Reason:  SecretCreatedOrUpdatedReason,
		Message: appendCreatedOrUpdatedSuffix(name),
	}
	_ = meta.SetStatusCondition(conditions, condition)
}

func SetSecretGenFailed(conditions *[]metav1.Condition, conditionType string, err error) {
	condition := metav1.Condition{
		Type:    conditionType,
		Status:  metav1.ConditionFalse,
		Reason:  SecretGenerationFailed,
		Message: "Failed to generate secret: " + err.Error(),
	}
	_ = meta.SetStatusCondition(conditions, condition)
}
