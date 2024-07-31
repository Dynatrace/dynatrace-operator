package monitoredentities

import (
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	MEIDOutdatedReason = "MEIDOutdated"

	MEIDEmptyReason = "MEIDEmpty"

	MEIDConditionType = "MonitoredEntity"
)

func setEmptyMEIDCondition(conditions *[]metav1.Condition) {
	condition := metav1.Condition{
		Type:    MEIDConditionType,
		Status:  metav1.ConditionFalse,
		Reason:  MEIDEmptyReason,
		Message: "Monitored Entities are empty nothing will be stored in the status",
	}

	_ = meta.SetStatusCondition(conditions, condition)
}
