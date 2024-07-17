package extension

import (
	"fmt"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func setSecretCreatedSuccessCondition(conditions *[]metav1.Condition) {
	condition := metav1.Condition{
		Type:    secretConditionType,
		Status:  metav1.ConditionTrue,
		Reason:  secretCreatedReason,
		Message: secretCreatedMessageSuccess,
	}
	_ = meta.SetStatusCondition(conditions, condition)
}

func setSecretCreatedFailureCondition(conditions *[]metav1.Condition, err error) {
	condition := metav1.Condition{
		Type:    secretConditionType,
		Status:  metav1.ConditionFalse,
		Reason:  secretCreatedReason,
		Message: fmt.Sprintf(secretCreatedMessageFailure, err),
	}
	_ = meta.SetStatusCondition(conditions, condition)
}

func removeSecretCreatedCondition(conditions *[]metav1.Condition) bool {
	return meta.RemoveStatusCondition(conditions, secretConditionType)
}
