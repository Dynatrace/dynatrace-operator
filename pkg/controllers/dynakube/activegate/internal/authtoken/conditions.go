package authtoken

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8sconditions"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const ActiveGateAuthTokenSecretConditionType string = "ActiveGateAuthTokenSecret"

func setAuthSecretCreated(conditions *[]metav1.Condition, conditionType string, msg string) {
	condition := metav1.Condition{
		Type:    conditionType,
		Status:  metav1.ConditionTrue,
		Reason:  k8sconditions.SecretCreatedReason,
		Message: msg,
	}
	_ = meta.SetStatusCondition(conditions, condition)
}
