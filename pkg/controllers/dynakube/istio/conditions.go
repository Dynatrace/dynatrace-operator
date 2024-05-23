package istio

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func createConditionTypeForComponent(component string) string {
	return fmt.Sprintf("IstioServiceConfigurationFor%s", strings.ToTitle(component))
}

func setServiceEntryUpdatedConditionForComponent(conditions *[]metav1.Condition, component string) {
	condition := metav1.Condition{
		Type:    createConditionTypeForComponent(component),
		Status:  metav1.ConditionTrue,
		Reason:  fmt.Sprintf("IstioServiceConfigurationFor%sChanged", strings.ToTitle(component)),
		Message: fmt.Sprintf("ServiceEntries and VirtualServices for %s have been configured.", strings.ToTitle(component)),
	}
	_ = meta.SetStatusCondition(conditions, condition)
}

func setServiceEntryFailedConditionForComponent(conditions *[]metav1.Condition, component string, err error) {
	condition := metav1.Condition{
		Type:    createConditionTypeForComponent(component),
		Status:  metav1.ConditionFalse,
		Reason:  fmt.Sprintf("ServiceEntryFor%sFailed", strings.ToTitle(component)),
		Message: fmt.Sprintf("Failed to configure Istio ServiceEntries and VirtualServices for %s with error: %s", strings.ToTitle(component), err.Error()),
	}
	_ = meta.SetStatusCondition(conditions, condition)
}

//
// func setIPServiceEntryForActiveGateFailedCondition(conditions *[]metav1.Condition, err error) {
// 	condition := metav1.Condition{
// 		Type:    serviceEntryForActiveGateConditionType,
// 		Status:  metav1.ConditionFalse,
// 		Reason:  ipServiceEntryForActiveGateFailedReason,
// 		Message: "Failed to create an IP ServiceEntry for ActiveGate, error: " + err.Error(),
// 	}
// 	_ = meta.SetStatusCondition(conditions, condition)
// }
