//go:build e2e

package logs

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/operator"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubernetes/objects/k8sdeployment"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

const operatorLogFileSuffix = "_operator.log"

// WriteOperatorLogToFile fetches the operator pod logs and writes them to a file in the test/e2e directory.
// The filename is generated from the test name and has the suffix "_operator.log",
func WriteOperatorLogToFile(ctx context.Context, envConfig *envconf.Config, t *testing.T) {
	resources := envConfig.Client().Resources()

	clientset, err := kubernetes.NewForConfig(resources.GetConfig())
	require.NoError(t, err)

	resultsDir := filepath.Join(findProjectRoot(t), "results")
	require.NoError(t, os.MkdirAll(resultsDir, 0755))
	logFile, err := os.Create(filepath.Join(resultsDir, t.Name()+operatorLogFileSuffix))
	require.NoError(t, err)

	err = k8sdeployment.NewQuery(ctx, resources, client.ObjectKey{Name: operator.DeploymentName, Namespace: operator.DefaultNamespace}).ForEachPod(func(pod corev1.Pod) {
		err = copyLogStream(ctx, clientset, logFile, logParams{
			namespace:     pod.Namespace,
			podName:       pod.Name,
			containerName: operator.ContainerName,
		})
		require.NoError(t, err)
	})
	require.NoError(t, err)
}

func ReadLog(ctx context.Context, t *testing.T, envConfig *envconf.Config, namespace, podName, containerName string) string { //nolint:revive
	resources := envConfig.Client().Resources()

	var pod corev1.Pod
	require.NoError(t, resources.WithNamespace(namespace).Get(ctx, podName, namespace, &pod))

	clientset, err := kubernetes.NewForConfig(resources.GetConfig())
	require.NoError(t, err)

	buffer := new(bytes.Buffer)
	err = copyLogStream(ctx, clientset, buffer, logParams{
		namespace:     namespace,
		podName:       podName,
		containerName: containerName,
	})
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

type logParams struct {
	namespace     string
	podName       string
	containerName string
}

func copyLogStream(ctx context.Context, clientset kubernetes.Interface, w io.Writer, params logParams) (err error) {
	logStream, err := clientset.CoreV1().Pods(params.namespace).GetLogs(params.podName, &corev1.PodLogOptions{
		Container: params.containerName,
	}).Stream(ctx)
	if err != nil {
		return err
	}

	defer func() {
		errClose := logStream.Close()
		if errClose != nil {
			err = errors.Join(err, errClose)
		}
	}()

	_, err = io.Copy(w, logStream)

	return err
}

func findProjectRoot(t *testing.T) string {
	path := "."

	for {
		var err error
		path, err = filepath.Abs(path)
		require.NoError(t, err)

		require.NotEqual(t, "/", path, "reached filesystem root")
		require.NotEqual(t, os.Getenv("HOME"), path, "reached user home directory")

		if _, err := os.Stat(filepath.Join(path, "PROJECT")); err == nil {
			return path
		}

		path = filepath.Join(path, "..")
	}
}
