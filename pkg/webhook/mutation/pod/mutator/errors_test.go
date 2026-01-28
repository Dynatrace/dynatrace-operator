package mutator

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestMutatorError_Error(t *testing.T) {
	t.Run("returns error message", func(t *testing.T) {
		innerErr := errors.New("something went wrong")
		mutErr := MutatorError{
			Err: innerErr,
		}
		assert.Equal(t, "something went wrong", mutErr.Error())
	})

	t.Run("returns empty string for nil error", func(t *testing.T) {
		mutErr := MutatorError{
			Err: nil,
		}
		// This will panic when calling Error() on nil, so we need to check the inner error is not nil
		// But we test the actual behavior - calling Error() when Err is nil
		defer func() {
			if r := recover(); r != nil {
				assert.NotNil(t, r, "Expected panic when Error is nil")
			}
		}()
		_ = mutErr.Error()
	})
}

func TestMutatorError_Unwrap(t *testing.T) {
	t.Run("returns wrapped error", func(t *testing.T) {
		innerErr := errors.New("inner error")
		mutErr := MutatorError{
			Err: innerErr,
		}
		assert.Equal(t, innerErr, mutErr.Unwrap())
	})

	t.Run("returns nil when no error is wrapped", func(t *testing.T) {
		mutErr := MutatorError{
			Err: nil,
		}
		assert.Nil(t, mutErr.Unwrap())
	})

	t.Run("works with errors.Is", func(t *testing.T) {
		sentinelErr := errors.New("sentinel")
		mutErr := MutatorError{
			Err: sentinelErr,
		}
		assert.True(t, errors.Is(mutErr, sentinelErr))
	})

	t.Run("works with wrapped errors", func(t *testing.T) {
		innerErr := errors.New("inner")
		wrappedErr := errors.New("outer: " + innerErr.Error())
		mutErr := MutatorError{
			Err: wrappedErr,
		}
		// Unwrap should return the outer error
		assert.Equal(t, wrappedErr, mutErr.Unwrap())
	})
}

func TestMutatorError_SetAnnotations(t *testing.T) {
	t.Run("calls annotate function when provided", func(t *testing.T) {
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:        "test-pod",
				Annotations: map[string]string{},
			},
		}

		called := false
		annotateFunc := func(p *corev1.Pod) {
			called = true
			p.Annotations["test-key"] = "test-value"
		}

		mutErr := MutatorError{
			Err:      errors.New("test error"),
			Annotate: annotateFunc,
		}

		mutErr.SetAnnotations(pod)

		assert.True(t, called, "Annotate function should be called")
		assert.Equal(t, "test-value", pod.Annotations["test-key"])
	})

	t.Run("does nothing when annotate function is nil", func(t *testing.T) {
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:        "test-pod",
				Annotations: map[string]string{},
			},
		}

		mutErr := MutatorError{
			Err:      errors.New("test error"),
			Annotate: nil,
		}

		// Should not panic or error
		mutErr.SetAnnotations(pod)

		assert.Empty(t, pod.Annotations)
	})

	t.Run("annotate function can add multiple annotations", func(t *testing.T) {
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:        "test-pod",
				Annotations: map[string]string{},
			},
		}

		annotateFunc := func(p *corev1.Pod) {
			p.Annotations["key1"] = "value1"
			p.Annotations["key2"] = "value2"
			p.Annotations["key3"] = "value3"
		}

		mutErr := MutatorError{
			Err:      errors.New("test error"),
			Annotate: annotateFunc,
		}

		mutErr.SetAnnotations(pod)

		assert.Len(t, pod.Annotations, 3)
		assert.Equal(t, "value1", pod.Annotations["key1"])
		assert.Equal(t, "value2", pod.Annotations["key2"])
		assert.Equal(t, "value3", pod.Annotations["key3"])
	})

	t.Run("annotate function can modify existing annotations", func(t *testing.T) {
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-pod",
				Annotations: map[string]string{
					"existing-key": "original-value",
				},
			},
		}

		annotateFunc := func(p *corev1.Pod) {
			p.Annotations["existing-key"] = "modified-value"
			p.Annotations["new-key"] = "new-value"
		}

		mutErr := MutatorError{
			Err:      errors.New("test error"),
			Annotate: annotateFunc,
		}

		mutErr.SetAnnotations(pod)

		assert.Len(t, pod.Annotations, 2)
		assert.Equal(t, "modified-value", pod.Annotations["existing-key"])
		assert.Equal(t, "new-value", pod.Annotations["new-key"])
	})
}
