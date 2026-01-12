package oaconnectioninfo

import (
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	oaConnectionInfoConditionType = "OneAgentConnectionInfo"

	EmptyCommunicationHostsReason = "EmptyCommunicationHosts"
)

func setEmptyCommunicationHostsCondition(conditions *[]metav1.Condition) {
	condition := metav1.Condition{
		Type:    oaConnectionInfoConditionType,
		Status:  metav1.ConditionFalse,
		Reason:  EmptyCommunicationHostsReason,
		Message: "No communication endpoints for the OneAgent are available",
	}
	_ = meta.SetStatusCondition(conditions, condition)
}
