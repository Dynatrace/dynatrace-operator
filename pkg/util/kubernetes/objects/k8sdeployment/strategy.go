// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package k8sdeployment

import (
	appsv1 "k8s.io/api/apps/v1"
)

// MergeStrategy applies the fields set in desired on top of current. Fields left empty in desired
// keep the value stored on the object, including what the apiserver defaulted.
func MergeStrategy(current, desired appsv1.DeploymentStrategy) appsv1.DeploymentStrategy {
	if desired.Type != "" {
		current.Type = desired.Type
	}

	if desired.RollingUpdate != nil {
		var rollingUpdate appsv1.RollingUpdateDeployment

		if current.RollingUpdate != nil {
			rollingUpdate = *current.RollingUpdate
		}

		if desired.RollingUpdate.MaxSurge != nil {
			rollingUpdate.MaxSurge = desired.RollingUpdate.MaxSurge
		}

		if desired.RollingUpdate.MaxUnavailable != nil {
			rollingUpdate.MaxUnavailable = desired.RollingUpdate.MaxUnavailable
		}

		current.RollingUpdate = &rollingUpdate
	}

	// The apiserver rejects a rollingUpdate block for Recreate.
	if current.Type == appsv1.RecreateDeploymentStrategyType {
		current.RollingUpdate = nil
	}

	return current
}
