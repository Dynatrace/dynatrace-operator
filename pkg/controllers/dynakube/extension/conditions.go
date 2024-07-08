package extension

import (
	"fmt"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	secretConditionType = "Secret"
)

func setSecretCreated(conditions *[]metav1.Condition) {
	condition := metav1.Condition{
		Type:    secretConditionType,
		Status:  metav1.ConditionTrue,
		Reason:  "SecretCreated",
		Message: "EEC token created",
	}
	_ = meta.SetStatusCondition(conditions, condition)
}

func setSecretCreationFailed(conditions *[]metav1.Condition, err error) {
	condition := metav1.Condition{
		Type:    secretConditionType,
		Status:  metav1.ConditionFalse,
		Reason:  "SecretCreationFailed",
		Message: fmt.Sprintf("EEC token creation failed: %s", err),
	}
	_ = meta.SetStatusCondition(conditions, condition)
}
