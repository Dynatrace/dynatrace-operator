// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package k8sconditions

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8smeta"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// IsOutdated determines if a given is considered outdated according to the DynaKube's FeatureApiRequestThreshold
// This is used for those conditions that are (also) used for limiting API requests.
func IsOutdated(dk *dynakube.DynaKube, conditionType string) bool {
	condition := meta.FindStatusCondition(*dk.Conditions(), conditionType)
	if condition == nil {
		return true
	}

	if condition.Status == metav1.ConditionFalse && condition.Reason != OptionalScopeMissingReason {
		return true
	}

	return k8smeta.TimeoutReached(&condition.LastTransitionTime, dk.APIRequestThreshold())
}
