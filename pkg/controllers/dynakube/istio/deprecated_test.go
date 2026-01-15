package istio

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestMigrateDeprecatedCondition(t *testing.T) {
	tests := []struct {
		name                   string
		conditions             *[]metav1.Condition
		expectedConditions     []metav1.Condition
		expectDeprecatedGone   bool
		expectNewConditionType bool
	}{
		{
			name:               "nil conditions - no panic",
			conditions:         nil,
			expectedConditions: nil,
		},
		{
			name:               "empty conditions - no change",
			conditions:         &[]metav1.Condition{},
			expectedConditions: []metav1.Condition{},
		},
		{
			name: "no deprecated condition - no change",
			conditions: &[]metav1.Condition{
				{
					Type:    "SomeOtherCondition",
					Status:  metav1.ConditionTrue,
					Reason:  "SomeReason",
					Message: "Some message",
				},
			},
			expectedConditions: []metav1.Condition{
				{
					Type:    "SomeOtherCondition",
					Status:  metav1.ConditionTrue,
					Reason:  "SomeReason",
					Message: "Some message",
				},
			},
			expectDeprecatedGone:   true,
			expectNewConditionType: false,
		},
		{
			name: "deprecated condition exists - migrated to new condition type",
			conditions: &[]metav1.Condition{
				{
					Type:    getConditionTypeName(deprecatedComponent),
					Status:  metav1.ConditionTrue,
					Reason:  "DeprecatedReason",
					Message: "Deprecated message",
				},
			},
			expectDeprecatedGone:   true,
			expectNewConditionType: true,
		},
		{
			name: "deprecated condition with false status - migrated correctly",
			conditions: &[]metav1.Condition{
				{
					Type:    getConditionTypeName(deprecatedComponent),
					Status:  metav1.ConditionFalse,
					Reason:  "FailedReason",
					Message: "Failed message",
				},
			},
			expectDeprecatedGone:   true,
			expectNewConditionType: true,
		},
		{
			name: "deprecated condition alongside other conditions - only deprecated migrated",
			conditions: &[]metav1.Condition{
				{
					Type:    "SomeOtherCondition",
					Status:  metav1.ConditionTrue,
					Reason:  "SomeReason",
					Message: "Some message",
				},
				{
					Type:    getConditionTypeName(deprecatedComponent),
					Status:  metav1.ConditionTrue,
					Reason:  "DeprecatedReason",
					Message: "Deprecated message",
				},
			},
			expectDeprecatedGone:   true,
			expectNewConditionType: true,
		},
		{
			name: "new condition type already exists - deprecated condition is removed",
			conditions: &[]metav1.Condition{
				{
					Type:    getConditionTypeName(CodeModuleComponent),
					Status:  metav1.ConditionTrue,
					Reason:  "ExistingReason",
					Message: "Existing message",
				},
				{
					Type:    getConditionTypeName(deprecatedComponent),
					Status:  metav1.ConditionFalse,
					Reason:  "DeprecatedReason",
					Message: "Deprecated message",
				},
			},
			expectDeprecatedGone:   true,
			expectNewConditionType: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			migrateDeprecatedCondition(tt.conditions)

			if tt.conditions == nil {
				return
			}

			if tt.expectedConditions != nil {
				require.Len(t, *tt.conditions, len(tt.expectedConditions))

				for _, expected := range tt.expectedConditions {
					actual := meta.FindStatusCondition(*tt.conditions, expected.Type)
					require.NotNil(t, actual, "expected condition %s not found", expected.Type)
					assert.Equal(t, expected.Status, actual.Status)
					assert.Equal(t, expected.Reason, actual.Reason)
					assert.Equal(t, expected.Message, actual.Message)
				}
			}

			if tt.expectDeprecatedGone {
				deprecatedCondition := meta.FindStatusCondition(*tt.conditions, getConditionTypeName(deprecatedComponent))
				assert.Nil(t, deprecatedCondition, "deprecated condition should be removed")
			}

			if tt.expectNewConditionType {
				newCondition := meta.FindStatusCondition(*tt.conditions, getConditionTypeName(CodeModuleComponent))
				assert.NotNil(t, newCondition, "new condition type should exist after migration")
			}
		})
	}
}
