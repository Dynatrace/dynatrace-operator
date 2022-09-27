package support_archive

import (
	"archive/tar"
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

func TestLogCollector(t *testing.T) {
	const namespace = "dynatrace"
	fakeClientSet := fake.NewSimpleClientset(
		&corev1.Pod{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Pod",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pod1",
				Namespace: namespace,
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{Name: "container1"},
					{Name: "container2"},
				},
			},
		},
		&corev1.Pod{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Pod",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pod2",
				Namespace: namespace,
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{Name: "container1"},
					{Name: "container2"},
				},
			},
		},
		&corev1.Pod{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Pod",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pod3",
				Namespace: namespace,
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{Name: "container1"},
					{Name: "container2"},
				},
			},
		},
		&corev1.Pod{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Pod",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pod4",
				Namespace: namespace,
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{Name: "container1"},
					{Name: "container2"},
				},
			},
		})

	ctx := supportArchiveContext{
		ctx:           context.TODO(),
		clientSet:     fakeClientSet,
		apiReader:     nil,
		namespaceName: "dynatrace",
		toStdout:      false,
		targetDir:     "",
	}

	tarBuffer := bytes.Buffer{}
	tarball := tarball{
		tarWriter: tar.NewWriter(&tarBuffer),
	}

	require.NoError(t, collectLogs(&ctx, &tarball))
	tarball.tarWriter.Close()

	tarReader := tar.NewReader(&tarBuffer)

	tarHeader, err := tarReader.Next()
	require.NoError(t, err)
	assert.Equal(t, "pod1_container1.log", tarHeader.Name)

	tarHeader, err = tarReader.Next()
	require.NoError(t, err)
	assert.Equal(t, "pod1_container1_previous.log", tarHeader.Name)

	tarHeader, err = tarReader.Next()
	require.NoError(t, err)
	assert.Equal(t, "pod1_container2.log", tarHeader.Name)

	tarHeader, err = tarReader.Next()
	require.NoError(t, err)
	assert.Equal(t, "pod1_container2_previous.log", tarHeader.Name)

	tarHeader, err = tarReader.Next()
	require.NoError(t, err)
	assert.Equal(t, "pod2_container1.log", tarHeader.Name)

	tarHeader, err = tarReader.Next()
	require.NoError(t, err)
	assert.Equal(t, "pod2_container1_previous.log", tarHeader.Name)

	tarHeader, err = tarReader.Next()
	require.NoError(t, err)
	assert.Equal(t, "pod2_container2.log", tarHeader.Name)

	tarHeader, err = tarReader.Next()
	require.NoError(t, err)
	assert.Equal(t, "pod2_container2_previous.log", tarHeader.Name)
}
