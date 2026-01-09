package k8sconditions

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	DeploymentsAppliedReason = "DeploymentsApplied"
)

type Setter interface {
	metav1.Object
	Conditions() *[]metav1.Condition
}

func SetDeploymentsApplied(obj Setter, conditionType string, names []string) {
	const maxNames = 3
	if len(names) > maxNames {
		more := len(names) - maxNames
		// Don't mutate input names
		names = append([]string{}, names[:maxNames]...)
		names = append(names, fmt.Sprintf("... %d more omitted", more))
	}

	condition := metav1.Condition{
		Type:               conditionType,
		Status:             metav1.ConditionTrue,
		Reason:             DeploymentsAppliedReason,
		Message:            strings.Join(names, ", "),
		ObservedGeneration: obj.GetGeneration(),
	}
	_ = meta.SetStatusCondition(obj.Conditions(), condition)
}
