package conditions

import (
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	KubeApiErrorReason      = "KubeApiError"
	DynatraceApiErrorReason = "DynatraceApiError"
)

func SetKubeApiErrorCondition(conditions *[]metav1.Condition, conditionType string, err error) {
	condition := metav1.Condition{
		Type:    conditionType,
		Status:  metav1.ConditionFalse,
		Reason:  KubeApiErrorReason,
		Message: "A problem occurred when using the Kubernetes API: " + err.Error(),
	}
	_ = meta.SetStatusCondition(conditions, condition)
}

func SetDynatraceApiErrorCondition(conditions *[]metav1.Condition, conditionType string, err error) {
	condition := metav1.Condition{
		Type:    conditionType,
		Status:  metav1.ConditionFalse,
		Reason:  DynatraceApiErrorReason,
		Message: "A problem occurred when using the Dynatrace API: " + err.Error(),
	}
	_ = meta.SetStatusCondition(conditions, condition)
}
