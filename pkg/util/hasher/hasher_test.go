// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package hasher

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

func TestGenerateHash(t *testing.T) {
	type testCase struct {
		title string
		in    any
	}

	cases := []testCase{
		{
			title: "nil",
			in:    nil,
		},
		{
			title: "empty",
			in:    "",
		},
		{
			title: "int",
			in:    1,
		},
		{
			title: "string",
			in:    "password123",
		},
		{
			title: "[]string",
			in:    []string{"password123", "password321"},
		},
		{
			title: "map[string][]byte",
			in:    map[string][]byte{"tokenField": []byte("password321")},
		},
		{
			title: "PodTemplate",
			in: corev1.PodTemplate{
				Name:     "my-pod",
				Template: corev1.PodTemplateSpec{Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "my-container"}}}},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			initialHash, err := GenerateHash(c.in)
			require.NoError(t, err)
			require.NotEmpty(t, initialHash)

			rerunHash, err := GenerateHash(c.in)
			require.NoError(t, err)

			assert.Equal(t, initialHash, rerunHash)
		})
	}
}

func TestGenerateSecureHash(t *testing.T) {
	type testCase struct {
		title string
		in    any
	}

	cases := []testCase{
		{
			title: "nil",
			in:    nil,
		},
		{
			title: "empty",
			in:    "",
		},
		{
			title: "int",
			in:    1,
		},
		{
			title: "string",
			in:    "password123",
		},
		{
			title: "[]string",
			in:    []string{"password123", "password321"},
		},
		{
			title: "map[string][]byte",
			in:    map[string][]byte{"tokenField": []byte("password321")},
		},
		{
			title: "PodTemplate",
			in: corev1.PodTemplate{
				Name:     "my-pod",
				Template: corev1.PodTemplateSpec{Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "my-container"}}}},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			initialHash, err := GenerateSecureHash(c.in)
			require.NoError(t, err)
			require.NotEmpty(t, initialHash)
			assert.Len(t, initialHash, 16)

			rerunHash, err := GenerateSecureHash(c.in)
			require.NoError(t, err)

			assert.Equal(t, initialHash, rerunHash)
		})
	}
}

func TestGenerateHashDifferentInputs(t *testing.T) {
	t.Run("GenerateHash", func(t *testing.T) {
		hash1, err := GenerateHash("password123")
		require.NoError(t, err)

		hash2, err := GenerateHash("password321")
		require.NoError(t, err)

		assert.NotEqual(t, hash1, hash2)
	})
	t.Run("GenerateSecureHash", func(t *testing.T) {
		hash1, err := GenerateSecureHash("password123")
		require.NoError(t, err)

		hash2, err := GenerateSecureHash("password321")
		require.NoError(t, err)

		assert.NotEqual(t, hash1, hash2)
	})
}

func TestGenerateHashUnmarshalableInput(t *testing.T) {
	// channels can't be marshaled to JSON
	unmarshalable := make(chan int)

	t.Run("GenerateHash", func(t *testing.T) {
		hash, err := GenerateHash(unmarshalable)
		require.Error(t, err)
		assert.Empty(t, hash)
	})
	t.Run("GenerateSecureHash", func(t *testing.T) {
		hash, err := GenerateSecureHash(unmarshalable)
		require.Error(t, err)
		assert.Empty(t, hash)
	})
	t.Run("IsDifferent", func(t *testing.T) {
		_, err := IsDifferent(unmarshalable, "valid")
		require.Error(t, err)

		_, err = IsDifferent("valid", unmarshalable)
		require.Error(t, err)
	})
}

func TestIsDifferent(t *testing.T) {
	testDeployment := appsv1.Deployment{
		Name: "deployment"}
	testDaemonSet := appsv1.DaemonSet{
		Name: "daemonset"}

	t.Run("different", func(t *testing.T) {
		isDifferent, err := IsDifferent(testDeployment, testDaemonSet)
		require.NoError(t, err)
		assert.True(t, isDifferent)
	})
	t.Run("same", func(t *testing.T) {
		isDifferent, err := IsDifferent(testDeployment, testDeployment)
		require.NoError(t, err)
		assert.False(t, isDifferent)
	})
}

func TestIsAnnotationDifferent(t *testing.T) {
	testDeployment := &appsv1.Deployment{
		Annotations: map[string]string{
			AnnotationHash: "hash1",
		}}
	testDaemonSet := &appsv1.DaemonSet{
		Annotations: map[string]string{
			AnnotationHash: "hash2",
		}}

	t.Run("different", func(t *testing.T) {
		isDifferent := IsAnnotationDifferent(&testDeployment.ObjectMeta, &testDaemonSet.ObjectMeta)
		assert.True(t, isDifferent)
	})
	t.Run("same", func(t *testing.T) {
		isDifferent := IsAnnotationDifferent(&testDeployment.ObjectMeta, &testDeployment.ObjectMeta)
		assert.False(t, isDifferent)
	})
	t.Run("missing annotation => different from present one", func(t *testing.T) {
		withoutAnnotations := &appsv1.DaemonSet{}

		assert.True(t, IsAnnotationDifferent(&testDeployment.ObjectMeta, &withoutAnnotations.ObjectMeta))
		assert.True(t, IsAnnotationDifferent(&withoutAnnotations.ObjectMeta, &testDeployment.ObjectMeta))
	})
	t.Run("both without annotation => same", func(t *testing.T) {
		assert.False(t, IsAnnotationDifferent(&appsv1.Deployment{}, &appsv1.DaemonSet{}))
	})
}

func TestAddAnnotation(t *testing.T) {
	t.Run("nil => error", func(t *testing.T) {
		err := AddAnnotation(nil)
		require.Error(t, err)
	})
	t.Run("typed nil => error", func(t *testing.T) {
		var nilDaemonSet *appsv1.DaemonSet

		err := AddAnnotation(nilDaemonSet)
		require.Error(t, err)
	})
	t.Run("existing hash annotation is not overwritten", func(t *testing.T) {
		testDaemonSet := &appsv1.DaemonSet{
			Annotations: map[string]string{
				AnnotationHash: "preset-hash",
			}}
		err := AddAnnotation(testDaemonSet)
		require.NoError(t, err)
		assert.Len(t, testDaemonSet.Annotations, 1)
		assert.Equal(t, "preset-hash", testDaemonSet.Annotations[AnnotationHash])
	})
	t.Run("append to annotations", func(t *testing.T) {
		testDaemonSet := &appsv1.DaemonSet{
			Annotations: map[string]string{
				"something": "else",
			}}
		err := AddAnnotation(testDaemonSet)
		require.NoError(t, err)
		assert.Len(t, testDaemonSet.Annotations, 2)
		assert.NotEmpty(t, testDaemonSet.Annotations[AnnotationHash])
	})
	t.Run("create annotation map, if not there", func(t *testing.T) {
		testDaemonSet := &appsv1.DaemonSet{}
		err := AddAnnotation(testDaemonSet)
		require.NoError(t, err)
		assert.Len(t, testDaemonSet.Annotations, 1)
		assert.NotEmpty(t, testDaemonSet.Annotations[AnnotationHash])
	})
}
