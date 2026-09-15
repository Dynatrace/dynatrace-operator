// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package k8sstatefulset

import appsv1 "k8s.io/api/apps/v1"

// MergeUpdateStrategy applies the fields set in desired on top of current. Fields left empty in
// desired keep the value stored on the object, including what the apiserver defaulted.
func MergeUpdateStrategy(current, desired appsv1.StatefulSetUpdateStrategy) appsv1.StatefulSetUpdateStrategy {
	if desired.Type != "" {
		current.Type = desired.Type
	}

	if desired.RollingUpdate != nil {
		var rollingUpdate appsv1.RollingUpdateStatefulSetStrategy

		if current.RollingUpdate != nil {
			rollingUpdate = *current.RollingUpdate
		}

		if desired.RollingUpdate.Partition != nil {
			rollingUpdate.Partition = desired.RollingUpdate.Partition
		}

		if desired.RollingUpdate.MaxUnavailable != nil {
			rollingUpdate.MaxUnavailable = desired.RollingUpdate.MaxUnavailable
		}

		current.RollingUpdate = &rollingUpdate
	}

	// The apiserver rejects a rollingUpdate block for OnDelete.
	if current.Type == appsv1.OnDeleteStatefulSetStrategyType {
		current.RollingUpdate = nil
	}

	return current
}
