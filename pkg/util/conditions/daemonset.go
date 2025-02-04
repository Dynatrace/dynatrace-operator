package conditions

import (
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	DaemonSetSetCreatedReason  = "DaemonSetCreated"
	DaemonSetSetOutdatedReason = "DaemonSetOutdated"
)

func SetDaemonSetCreated(conditions *[]metav1.Condition, conditionType, name string) {
	condition := metav1.Condition{
		Type:    conditionType,
		Status:  metav1.ConditionTrue,
		Reason:  DaemonSetSetCreatedReason,
		Message: appendCreatedSuffix(name),
	}
	_ = meta.SetStatusCondition(conditions, condition)
}

func SetDaemonSetOutdated(conditions *[]metav1.Condition, conditionType, name string) {
	condition := metav1.Condition{
		Type:    conditionType,
		Status:  metav1.ConditionFalse,
		Reason:  DaemonSetSetOutdatedReason,
		Message: appendOutdatedSuffix(name),
	}
	_ = meta.SetStatusCondition(conditions, condition)
}
