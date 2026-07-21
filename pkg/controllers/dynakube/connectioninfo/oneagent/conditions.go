// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package oaconnectioninfo

import (
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	oaConnectionInfoConditionType = "OneAgentConnectionInfo"

	EmptyCommunicationHostsReason   = "EmptyCommunicationHosts"
	StaleNetworkZoneEndpointsReason = "StaleNetworkZoneEndpoints"
)

func setEmptyCommunicationHostsCondition(conditions *[]metav1.Condition) {
	condition := metav1.Condition{
		Type:    oaConnectionInfoConditionType,
		Status:  metav1.ConditionFalse,
		Reason:  EmptyCommunicationHostsReason,
		Message: "No communication endpoints are available for the OneAgents",
	}
	_ = meta.SetStatusCondition(conditions, condition)
}

func setStaleNetworkZoneEndpointsCondition(conditions *[]metav1.Condition) {
	condition := metav1.Condition{
		Type:    oaConnectionInfoConditionType,
		Status:  metav1.ConditionFalse,
		Reason:  StaleNetworkZoneEndpointsReason,
		Message: "OneAgent endpoints do not advertise every local ActiveGate Service IP; postponing OneAgent deployment until the ActiveGate has re-registered",
	}
	_ = meta.SetStatusCondition(conditions, condition)
}
