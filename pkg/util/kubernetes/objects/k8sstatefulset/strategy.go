// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package k8sstatefulset

import (
	appsv1 "k8s.io/api/apps/v1"
)

// NormalizeUpdateStrategy fills in the same defaults the apiserver applies in SetDefaults_StatefulSet, so the
// result compares equal to what the apiserver stores once the object is created or updated. Without this,
// a desired strategy that only sets a subset of fields (or none at all) would never match the
// apiserver-defaulted value already on the stored object, causing a perpetual Update.
func NormalizeUpdateStrategy(current, desired appsv1.StatefulSetUpdateStrategy) appsv1.StatefulSetUpdateStrategy {
	normalized := *desired.DeepCopy()

	if normalized.Type == "" {
		normalized.Type = appsv1.RollingUpdateStatefulSetStrategyType
	}

	if normalized.Type != appsv1.RollingUpdateStatefulSetStrategyType {
		normalized.RollingUpdate = nil

		return normalized
	}

	if normalized.RollingUpdate == nil {
		normalized.RollingUpdate = &appsv1.RollingUpdateStatefulSetStrategy{}
	}

	if normalized.RollingUpdate.Partition == nil {
		normalized.RollingUpdate.Partition = new(int32(0))
	}

	// MaxUnavailable defaulting on the apiserver is gated behind the MaxUnavailableStatefulSet feature gate,
	// so it can't be hardcoded here without risking a permanent mismatch on clusters where the gate is off.
	// Instead, carry forward whatever the apiserver already put on the stored object.
	if normalized.RollingUpdate.MaxUnavailable == nil && current.RollingUpdate != nil {
		normalized.RollingUpdate.MaxUnavailable = current.RollingUpdate.MaxUnavailable
	}

	return normalized
}
