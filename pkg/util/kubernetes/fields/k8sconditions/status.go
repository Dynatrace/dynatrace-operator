// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package k8sconditions

import (
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	StatusUpdatedReason  = "StatusUpdated"
	StatusOutdatedReason = "StatusOutdated"

	// MissingMEIDMessage is used by conditions that can't proceed without a Kubernetes cluster MEID.
	MissingMEIDMessage = "no KubernetesMEID found"
)

func SetStatusUpdated(conditions *[]metav1.Condition, conditionType, msg string) {
	condition := metav1.Condition{
		Type:    conditionType,
		Status:  metav1.ConditionTrue,
		Reason:  StatusUpdatedReason,
		Message: msg,
	}
	_ = meta.SetStatusCondition(conditions, condition)
}

func SetStatusOutdated(conditions *[]metav1.Condition, conditionType, msg string) {
	condition := metav1.Condition{
		Type:    conditionType,
		Status:  metav1.ConditionFalse,
		Reason:  StatusOutdatedReason,
		Message: msg,
	}
	_ = meta.SetStatusCondition(conditions, condition)
}
