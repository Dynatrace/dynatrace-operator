package k8sconditions

import (
	"errors"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	KubeAPIErrorReason      = "KubeApiError"
	DynatraceAPIErrorReason = "DynatraceApiError"
)

func SetKubeAPIError(conditions *[]metav1.Condition, conditionType string, err error) {
	if err == nil {
		return
	}

	condition := metav1.Condition{
		Type:    conditionType,
		Status:  metav1.ConditionFalse,
		Reason:  KubeAPIErrorReason,
		Message: "A problem occurred when using the Kubernetes API: " + err.Error(),
	}
	_ = meta.SetStatusCondition(conditions, condition)
}

func SetDynatraceAPIError(conditions *[]metav1.Condition, conditionType string, err error) {
	if err == nil {
		return
	}

	condition := metav1.Condition{
		Type:    conditionType,
		Status:  metav1.ConditionFalse,
		Reason:  DynatraceAPIErrorReason,
		Message: "A problem occurred when using the Dynatrace API: " + err.Error(),
	}
	_ = meta.SetStatusCondition(conditions, condition)
}

func IsKubeAPIError(err error) bool {
	status, ok := err.(k8serrors.APIStatus)

	return ok || errors.As(err, &status)
}
