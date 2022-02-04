package v1beta1

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetInternalFlags(t *testing.T) {
	t.Run("Empty pod should have no internal flags", func(t *testing.T) {
		annotatedObject := &corev1.Pod{}
		expectedMapContents := "map[]"

		assert.Equal(t, expectedMapContents, fmt.Sprint(GetInternalFlags(annotatedObject)))
	})

	t.Run("Only internal flags should be returned (1)", func(t *testing.T) {
		annotatedObject := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					"not-a-flag": "oh, no",
				},
			},
		}
		expectedMapContents := "map[]"

		assert.Equal(t, expectedMapContents, fmt.Sprint(GetInternalFlags(annotatedObject)))
	})

	t.Run("Only internal flags should be returned (2)", func(t *testing.T) {
		annotatedObject := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					InternalFlagPrefix + "some-flag":                  "something",
					InternalFlagPrefix + "other-flag":                 "nothing",
					"unexpected." + InternalFlagPrefix + "not-a-flag": "oh, no",
				},
			},
		}
		expectedMapContents := "map[internal.operator.dynatrace.com/other-flag:nothing internal.operator.dynatrace.com/some-flag:something]"

		assert.Equal(t, expectedMapContents, fmt.Sprint(GetInternalFlags(annotatedObject)))
	})
}

func TestIsInternalFlagsEqual(t *testing.T) {
	t.Run("Expected that no-flag objects are equal internal flags-wise", func(t *testing.T) {
		assert.True(t, IsInternalFlagsEqual(&corev1.Pod{}, &corev1.Pod{}))
		assert.True(t, IsInternalFlagsEqual(&corev1.Pod{}, &corev1.Service{}))
		assert.True(t, IsInternalFlagsEqual(&corev1.Namespace{}, &corev1.Service{}))
	})

	t.Run("Expected that objects without internal operator flags compare as equal", func(t *testing.T) {
		assert.True(t, IsInternalFlagsEqual(
			&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{
				"": "", "space": " ", "dot": ".",
			}}},
			&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{
				"comma": ",",
			}}},
		))
		assert.True(t, IsInternalFlagsEqual(
			&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{
				"": "", "space": " ", "dot": ".",
			}}},
			&corev1.Service{ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{
				"comma": ",",
			}}},
		))
		assert.True(t, IsInternalFlagsEqual(
			&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{
				"": "", "space": " ", "dot": ".",
			}}},
			&corev1.Service{ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{
				"comma": ",",
			}}},
		))
	})

	t.Run("Expected that objects with internal operator flags compare as equal if the flags are identical", func(t *testing.T) {
		assert.True(t, IsInternalFlagsEqual(
			&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{
				"dyna": "trace", InternalFlagPrefix + "flag": "value",
			}}},
			&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{
				"dyna": "truce", InternalFlagPrefix + "flag": "value",
			}}},
		))
		assert.True(t, IsInternalFlagsEqual(
			&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{
				"dyna": "trace", InternalFlagPrefix + "flag": "value",
			}}},
			&corev1.Service{ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{
				"dyna": "tarce", InternalFlagPrefix + "flag": "value",
			}}},
		))
		assert.True(t, IsInternalFlagsEqual(
			&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{
				"dyna": "trace", InternalFlagPrefix + "flag": "value",
			}}},
			&corev1.Service{ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{
				"dyna": "trcue", InternalFlagPrefix + "flag": "value",
			}}},
		))
	})

	t.Run("Expected that objects with internal operator flags compare as different if the flags have different values or are missing", func(t *testing.T) {
		assert.False(t, IsInternalFlagsEqual(
			&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{
				"dyna": "trace", InternalFlagPrefix + "flag": "value",
			}}},
			&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{
				"dyna": "trace", InternalFlagPrefix + "flag": "other value",
			}}},
		))
		assert.False(t, IsInternalFlagsEqual(
			&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{
				"dyna": "trace", InternalFlagPrefix + "flag": "value",
			}}},
			&corev1.Service{ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{
				"dyna": "trace", InternalFlagPrefix + "flag": "other value",
			}}},
		))
		assert.False(t, IsInternalFlagsEqual(
			&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{
				"dyna": "trace", InternalFlagPrefix + "flag": "value",
			}}},
			&corev1.Service{ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{
				"dyna": "trace", InternalFlagPrefix + "flag": "other value",
			}}},
		))
	})
}
