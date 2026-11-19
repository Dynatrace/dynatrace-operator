// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

//go:build e2e

package k8sobject

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	apimachineryruntime "k8s.io/apimachinery/pkg/runtime"
	apimachinerywait "k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
)

// ConditionsNotMatchedContextFunc match iff the object does not match any of the provided status conditions
//
//	A condition matches on type, status, reason
//	If the condition match, an error is returned with details found on the object
func ConditionsNotMatchedContextFunc(r *resources.Resources, obj k8s.Object, terminalStatusConditions ...metav1.Condition) apimachinerywait.ConditionWithContextFunc {
	return func(ctx context.Context) (bool, error) {
		if len(terminalStatusConditions) == 0 {
			return false, fmt.Errorf("requested to match against empty status conditions on the object %s/%s", obj.GetNamespace(), obj.GetName())
		}

		if err := r.Get(ctx, obj.GetName(), obj.GetNamespace(), obj); err != nil {
			return false, err
		}

		conditions, err := statusConditions(obj)
		if err != nil {
			return false, err
		}

		for _, terminal := range terminalStatusConditions {
			current := meta.FindStatusCondition(conditions, terminal.Type)
			if current != nil && current.Status == terminal.Status && current.Reason == terminal.Reason {
				return false, fmt.Errorf("terminal condition %s=%s (reason %s): %s", current.Type, current.Status, current.Reason, current.Message)
			}
		}

		return true, nil
	}
}

// CompositeWaitConditions wait for all conditions to be done. If any condition err, fails the wait
func CompositeWaitConditions(waitConditions ...apimachinerywait.ConditionWithContextFunc) apimachinerywait.ConditionWithContextFunc {

	return func(ctx context.Context) (bool, error) {
		allDone := true

		// evaluate every condition
		// early termination may hide an error in a condition further down the list
		for _, waitCond := range waitConditions {
			done, err := waitCond(ctx)
			if err != nil {
				return false, err
			}

			allDone = allDone && done
		}

		return allDone, nil
	}
}

// statusConditions extracts .status.conditions from any typed object; nil if there are none.
func statusConditions(obj k8s.Object) ([]metav1.Condition, error) {
	u, err := apimachineryruntime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return nil, err
	}

	statusMap, found, err := unstructured.NestedMap(u, "status")
	if err != nil || !found {
		return nil, err
	}

	var status struct {
		Conditions []metav1.Condition `json:"conditions"`
	}

	if err := apimachineryruntime.DefaultUnstructuredConverter.FromUnstructured(statusMap, &status); err != nil {
		return nil, err
	}

	return status.Conditions, nil
}
