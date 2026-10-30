// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package k8sdeployment

import (
	"fmt"

	k8sobject "github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/equality"
)

// CheckImmutableFields checks for immutable fields and returns [k8sobject.ErrRecreate] if any immutable field changed
func CheckImmutableFields(current, desired *appsv1.Deployment) error {
	if current.ResourceVersion == "" {
		return nil
	}

	if desired.Spec.Selector != nil && !equality.Semantic.DeepEqual(current.Spec.Selector, desired.Spec.Selector) {
		return fmt.Errorf("deployment selector changed: %w", k8sobject.ErrRecreate)
	}

	return nil
}
