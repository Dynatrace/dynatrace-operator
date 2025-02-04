package support_archive

import (
	"archive/zip"
	"bufio"
	"bytes"
	"io"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/labels"
	mocks "github.com/Dynatrace/dynatrace-operator/test/mocks/cmd/remote_command"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

const (
	eecPodName   = "dynakube-extensions-controller-0"
	eecNamespace = "dynatrace"

	zipLsFileName           = "logs/dynakube-extensions-controller-0/ls.txt"
	zipDiagExecutorFileName = "logs/dynakube-extensions-controller-0/var/lib/dynatrace/remotepluginmodule/log/extensions/diagnostics/diag_executor.log"

	lsOutput = `/var/lib/dynatrace/remotepluginmodule/log/extensions/:
datasources
diagnostics

/var/lib/dynatrace/remotepluginmodule/log/extensions/datasources:

/var/lib/dynatrace/remotepluginmodule/log/extensions/diagnostics:
diag_executor.log
`
	diagExecutorOutput = "lorem ipsum"
)

func TestFsLog(t *testing.T) {
	fakeClientSet := fake.NewSimpleClientset(
		&corev1.Pod{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Pod",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					labels.AppNameLabel:      LabelEecPodName,
					labels.AppManagedByLabel: "dynatrace-operator",
				},
				Name:      eecPodName,
				Namespace: eecNamespace,
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{Name: eecContainerName},
				},
			},
		})

	logBuffer := bytes.Buffer{}

	buffer := bytes.Buffer{}
	supportArchive := newZipArchive(bufio.NewWriter(&buffer))

	rce := mocks.NewExecutor(t)
	stdErr := &bytes.Buffer{}

	lsStdOut := &bytes.Buffer{}
	lsStdOut.WriteString(lsOutput)
	rce.On("Exec", mock.Anything, mock.Anything, eecPodName, eecNamespace, eecContainerName, []string{
		"/usr/bin/sh",
		"-c",
		"if [ -d '" + eecExtensionsPath + "' ]; then ls -R1 '" + eecExtensionsPath + "' ; else echo '<NOT FOUND>' ; fi",
	}).Return(lsStdOut, stdErr, nil)

	diagStdOut := &bytes.Buffer{}
	diagStdOut.WriteString(diagExecutorOutput)
	rce.On("Exec", mock.Anything, mock.Anything, eecPodName, eecNamespace, eecContainerName, []string{
		"/usr/bin/sh",
		"-c",
		"[ -e " + eecExtensionsPath + "/diagnostics/diag_executor.log ] && cat " + eecExtensionsPath + "/diagnostics/diag_executor.log || echo '<NOT FOUND>'",
	}).Return(diagStdOut, stdErr, nil)

	logCollector := newFsLogCollector(context.Background(),
		nil,
		rce,
		newSupportArchiveLogger(&logBuffer),
		supportArchive,
		fakeClientSet.CoreV1().Pods("dynatrace"),
		defaultOperatorAppName,
		true)

	require.NoError(t, logCollector.Do())

	require.NoError(t, supportArchive.Close())

	zipReader, err := zip.NewReader(bytes.NewReader(buffer.Bytes()), int64(buffer.Len()))
	require.NoError(t, err)

	require.Len(t, zipReader.File, 2)
	assert.Equal(t, zipLsFileName, zipReader.File[0].Name)
	assert.Equal(t, zipDiagExecutorFileName, zipReader.File[1].Name)

	f, err := zipReader.Open(zipLsFileName)
	require.NoError(t, err)
	contents, err := io.ReadAll(f)
	require.NoError(t, err)
	f.Close()
	assert.Equal(t, lsOutput, string(contents))

	f, err = zipReader.Open(zipDiagExecutorFileName)
	require.NoError(t, err)
	contents, err = io.ReadAll(f)
	require.NoError(t, err)
	f.Close()
	assert.Equal(t, diagExecutorOutput, string(contents))
}
