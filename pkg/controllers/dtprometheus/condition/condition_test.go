// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package condition

import (
	"errors"
	"fmt"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	testType      = "ComponentAvailable"
	testComponent = "component"
)

func TestSet(t *testing.T) {
	wrapped := fmt.Errorf("wrap: %w", errors.New("boom"))

	tests := []struct {
		name        string
		err         error
		ready       bool
		wantStatus  metav1.ConditionStatus
		wantReason  string
		wantMessage string
	}{
		{"not ready -> reconciling", nil, false, metav1.ConditionFalse, status.ReasonReconciling, "component is pending"},
		{"ready -> available", nil, true, metav1.ConditionTrue, status.ReasonAvailable, "component is ready"},
		{"error -> error, unwrapped message", wrapped, false, metav1.ConditionFalse, status.ReasonError, "boom"},
		{"error takes precedence over ready", wrapped, true, metav1.ConditionFalse, status.ReasonError, "boom"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var conditions []metav1.Condition

			Set(&conditions, testType, testComponent, func() bool { return test.ready }, test.err)

			condition := meta.FindStatusCondition(conditions, testType)
			require.NotNil(t, condition)
			assert.Equal(t, test.wantStatus, condition.Status)
			assert.Equal(t, test.wantReason, condition.Reason)
			assert.Equal(t, test.wantMessage, condition.Message)
		})
	}

	t.Run("isReady is not called when err is set", func(t *testing.T) {
		var conditions []metav1.Condition

		Set(&conditions, testType, testComponent, func() bool {
			t.Fatal("isReady must not be called on the error path")

			return false
		}, errors.New("boom"))

		assert.NotEmpty(t, conditions)
	})

	t.Run("unwraps only one level", func(t *testing.T) {
		var conditions []metav1.Condition

		inner := fmt.Errorf("inner: %w", errors.New("root"))
		Set(&conditions, testType, testComponent, func() bool { return false }, fmt.Errorf("outer: %w", inner))

		assert.Equal(t, "inner: root", meta.FindStatusCondition(conditions, testType).Message)
	})
}
