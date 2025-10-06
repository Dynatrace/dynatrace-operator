package conditions

import (
	"strings"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	DeploymentsAppliedReason = "DeploymentsApplied"
)

func SetDeploymentsApplied(conditions *[]metav1.Condition, conditionType string, names []string) {
	condition := metav1.Condition{
		Type:    conditionType,
		Status:  metav1.ConditionTrue,
		Reason:  DeploymentsAppliedReason,
		Message: strings.Join(names, ","),
	}
	_ = meta.SetStatusCondition(conditions, condition)
}
