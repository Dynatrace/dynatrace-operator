package support_archive

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/env"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/labels"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestGetAppName(t *testing.T) {
	const alternativeOperatorName = "renamed-operator"

	const alternativeNamespace = "weirednamespacename"

	fakeClientSet := fake.NewSimpleClientset(
		&corev1.Pod{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Pod",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      alternativeOperatorName,
				Namespace: alternativeNamespace,
				Labels: map[string]string{
					labels.AppNameLabel: alternativeOperatorName,
				},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{Name: "container1"},
					{Name: "container2"},
				},
			},
		})

	t.Setenv(env.PodName, alternativeOperatorName)
	assert.Equal(t, alternativeOperatorName, getAppNameLabel(context.TODO(), fakeClientSet.CoreV1().Pods(alternativeNamespace)))
}

func TestStdoutArgMissing(t *testing.T) {
	cmd := New()
	// should fail when invoked without --stdout
	err := cmd.Execute()
	require.Error(t, err)
}
