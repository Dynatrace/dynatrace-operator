//go:build e2e

package logs

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubernetes/objects/k8sdeployment"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

func FetchOperatorLog(ctx context.Context, envConfig *envconf.Config, t *testing.T) string {
	resources := envConfig.Client().Resources()

	var operatorLog string

	err := k8sdeployment.NewQuery(ctx, resources, client.ObjectKey{Name: "dynatrace-operator", Namespace: "dynatrace"}).ForEachPod(func(pod corev1.Pod) {
		operatorLog = ReadLog(ctx, t, envConfig, pod.Namespace, pod.Name, "operator")
	})
	require.NoError(t, err)

	return operatorLog
}

func ReadLog(ctx context.Context, t *testing.T, envConfig *envconf.Config, namespace, podName, containerName string) string { //nolint:revive
	resources := envConfig.Client().Resources()

	var pod corev1.Pod
	require.NoError(t, resources.WithNamespace(namespace).Get(ctx, podName, namespace, &pod))

	clientset, err := kubernetes.NewForConfig(resources.GetConfig())
	require.NoError(t, err)

	logStream, err := clientset.CoreV1().Pods(namespace).GetLogs(podName, &corev1.PodLogOptions{
		Container: containerName,
	}).Stream(ctx)
	require.NoError(t, err)

	defer func() {
		require.NoError(t, logStream.Close())
	}()

	buffer := new(bytes.Buffer)
	_, err = io.Copy(buffer, logStream)
	require.NoError(t, err)

	return buffer.String()
}

func AssertContains(t *testing.T, logStream io.ReadCloser, contains string) {
	buffer := new(bytes.Buffer)
	copied, err := io.Copy(buffer, logStream)

	require.NoError(t, err)
	require.Equal(t, int64(buffer.Len()), copied)
	assert.Contains(t, buffer.String(), contains)
}

func FindLineContainingText(log, searchText string) string {
	for line := range strings.SplitSeq(log, "\n") {
		if strings.Contains(line, searchText) {
			return line
		}
	}

	return ""
}
