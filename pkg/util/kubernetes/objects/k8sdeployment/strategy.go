// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package k8sdeployment

import (
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// NormalizeStrategy fills in the same defaults the apiserver applies in SetDefaults_Deployment, so the
// result compares equal to what the apiserver stores once the object is created or updated. Without this,
// a desired strategy that only sets a subset of fields (or none at all) would never match the
// apiserver-defaulted value already on the stored object, causing a perpetual Update.
func NormalizeStrategy(desired appsv1.DeploymentStrategy) appsv1.DeploymentStrategy {
	normalized := *desired.DeepCopy()

	if normalized.Type == "" {
		normalized.Type = appsv1.RollingUpdateDeploymentStrategyType
	}

	if normalized.Type != appsv1.RollingUpdateDeploymentStrategyType {
		normalized.RollingUpdate = nil

		return normalized
	}

	if normalized.RollingUpdate == nil {
		normalized.RollingUpdate = &appsv1.RollingUpdateDeployment{}
	}

	if normalized.RollingUpdate.MaxUnavailable == nil {
		maxUnavailable := intstr.FromString("25%")
		normalized.RollingUpdate.MaxUnavailable = &maxUnavailable
	}

	if normalized.RollingUpdate.MaxSurge == nil {
		maxSurge := intstr.FromString("25%")
		normalized.RollingUpdate.MaxSurge = &maxSurge
	}

	return normalized
}
