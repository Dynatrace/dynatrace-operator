// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

// Package condition centralises how the DTPrometheus component reconcilers report
// their availability, so every component surfaces the same condition shape.
package condition

import (
	"errors"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Set writes the availability condition for a DTPrometheus component based on the
// outcome of its reconcile.
//
// The ready and pending messages are derived from component so all components
// report in the same shape. isReady is only called when err is nil, so callers can
// pass a closure that dereferences state which is unset on error paths.
func Set(conditions *[]metav1.Condition, conditionType, component string, isReady func() bool, err error) {
	condition := metav1.Condition{Type: conditionType}

	switch {
	case err != nil:
		condition.Status = metav1.ConditionFalse
		condition.Reason = status.ReasonError
		condition.Message = safeUnwrap(err).Error()
	case isReady():
		condition.Status = metav1.ConditionTrue
		condition.Reason = status.ReasonAvailable
		condition.Message = component + " is ready"
	default:
		condition.Status = metav1.ConditionFalse
		condition.Reason = status.ReasonReconciling
		condition.Message = component + " is pending"
	}

	_ = meta.SetStatusCondition(conditions, condition)
}

// safeUnwrap returns the innermost wrapped error for cleaner condition messages.
func safeUnwrap(err error) error {
	if u := errors.Unwrap(err); u != nil {
		return u
	}

	return err
}
