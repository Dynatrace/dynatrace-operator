package hasher

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-pod",
				},
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
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-pod",
				},
				Template: corev1.PodTemplateSpec{Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "my-container"}}}},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			initialHash, err := GenerateSecureHash(c.in)
			require.NoError(t, err)
			require.NotEmpty(t, initialHash)

			rerunHash, err := GenerateSecureHash(c.in)
			require.NoError(t, err)

			assert.Equal(t, initialHash, rerunHash)
		})
	}
}

func TestIsDifferent(t *testing.T) {
	testDeployment := appsv1.Deployment{}
	testDeployment.Name = "deployment"
	testDaemonSet := appsv1.DaemonSet{}
	testDaemonSet.Name = "daemonset"

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

func TestIsHashAnnotationDifferent(t *testing.T) {
	testDeployment := appsv1.Deployment{}
	testDeployment.Annotations = map[string]string{
		AnnotationHash: "hash1",
	}
	testDaemonSet := appsv1.DaemonSet{}
	testDaemonSet.Annotations = map[string]string{
		AnnotationHash: "hash2",
	}

	t.Run("different", func(t *testing.T) {
		isDifferent := IsAnnotationDifferent(&testDeployment.ObjectMeta, &testDaemonSet.ObjectMeta)
		assert.True(t, isDifferent)
	})
	t.Run("same", func(t *testing.T) {
		isDifferent := IsAnnotationDifferent(&testDeployment.ObjectMeta, &testDeployment.ObjectMeta)
		assert.False(t, isDifferent)
	})
}

func TestAddHashAnnotation(t *testing.T) {
	t.Run("nil => error", func(t *testing.T) {
		err := AddAnnotation(nil)
		require.Error(t, err)
	})
	t.Run("append to annotations", func(t *testing.T) {
		testDaemonSet := appsv1.DaemonSet{}
		testDaemonSet.Annotations = map[string]string{
			"something": "else",
		}
		err := AddAnnotation(&testDaemonSet)
		require.NoError(t, err)
		assert.Len(t, testDaemonSet.Annotations, 2)
		assert.NotEmpty(t, testDaemonSet.Annotations[AnnotationHash])
	})
	t.Run("create annotation map, if not there", func(t *testing.T) {
		testDaemonSet := appsv1.DaemonSet{}
		err := AddAnnotation(&testDaemonSet)
		require.NoError(t, err)
		assert.Len(t, testDaemonSet.Annotations, 1)
		assert.NotEmpty(t, testDaemonSet.Annotations[AnnotationHash])
	})
}
