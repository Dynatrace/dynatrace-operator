package conditions

import (
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	SecretCreatedReason  = "SecretCreated"
	SecretUpdatedReason  = "SecretUpdated"
	SecretOutdatedReason = "SecretOutdated"
)

func SetSecretCreated(conditions *[]metav1.Condition, conditionType, message string) {
	condition := metav1.Condition{
		Type:    conditionType,
		Status:  metav1.ConditionTrue,
		Reason:  SecretCreatedReason,
		Message: message, // TODO: maybe only pass the name of the secret, and have the rest of the message more general?
	}
	_ = meta.SetStatusCondition(conditions, condition)
}

func SetSecretUpdated(conditions *[]metav1.Condition, conditionType, message string) {
	condition := metav1.Condition{
		Type:    conditionType,
		Status:  metav1.ConditionTrue,
		Reason:  SecretUpdatedReason,
		Message: message,
	}
	_ = meta.SetStatusCondition(conditions, condition)
}

func SetSecretOutdated(conditions *[]metav1.Condition, conditionType, message string) {
	condition := metav1.Condition{
		Type:    conditionType,
		Status:  metav1.ConditionFalse,
		Reason:  SecretOutdatedReason,
		Message: message,
	}
	_ = meta.SetStatusCondition(conditions, condition)
}
