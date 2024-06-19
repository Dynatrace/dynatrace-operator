package istio

import (
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func getConditionTypeName(component string) string {
	return "IstioFor" + component
}

func setServiceEntryUpdatedConditionForComponent(conditions *[]metav1.Condition, component string) {
	condition := metav1.Condition{
		Type:    getConditionTypeName(component),
		Status:  metav1.ConditionTrue,
		Reason:  "IstioFor" + component + "Changed",
		Message: "ServiceEntries and VirtualServices for " + component + " have been configured.",
	}
	_ = meta.SetStatusCondition(conditions, condition)
}

func setServiceEntryFailedConditionForComponent(conditions *[]metav1.Condition, component string, err error) {
	condition := metav1.Condition{
		Type:    getConditionTypeName(component),
		Status:  metav1.ConditionFalse,
		Reason:  "IstioFor" + component + "Failed",
		Message: "Failed to configure Istio ServiceEntries and VirtualServices for " + component + " with error: " + err.Error(),
	}
	_ = meta.SetStatusCondition(conditions, condition)
}
