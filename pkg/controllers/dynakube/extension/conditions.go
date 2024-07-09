package extension

import (
	"fmt"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	secretConditionType       = "Secret"
	secretCreatedReason       = "SecretCreated"
	secretCreatedMessageTrue  = "EEC token created"
	secretCreatedMessageFalse = "Error creating extensions secret: %s"
)

func setSecretCreated(conditions *[]metav1.Condition) {
	condition := metav1.Condition{
		Type:    secretConditionType,
		Status:  metav1.ConditionTrue,
		Reason:  secretCreatedReason,
		Message: secretCreatedMessageTrue,
	}
	_ = meta.SetStatusCondition(conditions, condition)
}

func setSecretCreatedFalse(conditions *[]metav1.Condition, err error) {
	condition := metav1.Condition{
		Type:    secretConditionType,
		Status:  metav1.ConditionFalse,
		Reason:  secretCreatedReason,
		Message: fmt.Sprintf(secretCreatedMessageFalse, err),
	}
	_ = meta.SetStatusCondition(conditions, condition)
}
