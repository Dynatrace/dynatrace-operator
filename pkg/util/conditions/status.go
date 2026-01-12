package conditions

import (
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	StatusUpdatedReason  = "StatusUpdated"
	StatusOutdatedReason = "StatusOutdated"
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
