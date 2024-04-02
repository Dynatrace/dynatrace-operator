package conditions

import (
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	StatefulSetCreatedReason  = "StatefulSetCreated"
	StatefulSetUpdatedReason  = "StatefulSetUpdated"
	StatefulSetOutdatedReason = "StatefulSetOutdated"
)

func SetStatefulSetCreated(conditions *[]metav1.Condition, conditionType, name string) {
	condition := metav1.Condition{
		Type:    conditionType,
		Status:  metav1.ConditionTrue,
		Reason:  StatefulSetCreatedReason,
		Message: appendCreatedSuffix(name),
	}
	_ = meta.SetStatusCondition(conditions, condition)
}

func SetStatefulSetUpdated(conditions *[]metav1.Condition, conditionType, name string) {
	condition := metav1.Condition{
		Type:    conditionType,
		Status:  metav1.ConditionTrue,
		Reason:  StatefulSetUpdatedReason,
		Message: appendUpdatedSuffix(name),
	}
	_ = meta.SetStatusCondition(conditions, condition)
}

func SetStatefulSetDeleted(conditions *[]metav1.Condition, conditionType, name string) {
	condition := metav1.Condition{
		Type:    conditionType,
		Status:  metav1.ConditionTrue,
		Reason:  StatefulSetUpdatedReason,
		Message: appendDeletedSuffix(name),
	}
	_ = meta.SetStatusCondition(conditions, condition)
}

func SetStatefulSetOutdated(conditions *[]metav1.Condition, conditionType, name string) {
	condition := metav1.Condition{
		Type:    conditionType,
		Status:  metav1.ConditionFalse,
		Reason:  StatefulSetOutdatedReason,
		Message: appendOutdatedSuffix(name),
	}
	_ = meta.SetStatusCondition(conditions, condition)
}
