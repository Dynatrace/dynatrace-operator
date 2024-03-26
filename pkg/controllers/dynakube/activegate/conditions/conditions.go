package conditions

import (
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ReasonCreated         string = "Created"
	ReasonError           string = "Error"
	ReasonUnexpectedError string = "UnexpectedError"
	ReasonUpToDate        string = "UpToDate"
)

// ActiveGate related conditions.
const (
	ActiveGateConnectionInfoConditionType string = "ActiveGateConnectionInfo"
	ActiveGateStatefulSetConditionType    string = "ActiveGateStatefulSet"
	ActiveGateVersionConditionType        string = "ActiveGateVersion"
)

func SetActiveGateConnectionInfoCondition(conditions *[]metav1.Condition, err error) {
	if err != nil {
		meta.SetStatusCondition(conditions, metav1.Condition{
			Type:    ActiveGateConnectionInfoConditionType,
			Status:  metav1.ConditionFalse,
			Reason:  ReasonUnexpectedError,
			Message: err.Error(),
		})

		return
	}

	meta.SetStatusCondition(conditions, metav1.Condition{
		Type:   ActiveGateConnectionInfoConditionType,
		Status: metav1.ConditionTrue,
		Reason: ReasonCreated,
	})
}

func SetActiveGateStatefulSetErrorCondition(conditions *[]metav1.Condition, err error) {
	if err != nil {
		meta.SetStatusCondition(conditions, metav1.Condition{
			Type:    ActiveGateStatefulSetConditionType,
			Status:  metav1.ConditionFalse,
			Reason:  ReasonError,
			Message: err.Error(),
		})
	}
}

func SetActiveGateVersionCondition(conditions *[]metav1.Condition, version string, err error) {
	if err != nil {
		meta.SetStatusCondition(conditions, metav1.Condition{
			Type:    ActiveGateVersionConditionType,
			Status:  metav1.ConditionFalse,
			Reason:  ReasonError,
			Message: err.Error(),
		})

		return
	}

	meta.SetStatusCondition(conditions, metav1.Condition{
		Type:    ActiveGateVersionConditionType,
		Status:  metav1.ConditionTrue,
		Reason:  ReasonUpToDate,
		Message: version,
	})
}
