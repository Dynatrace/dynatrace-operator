package support_archive

import (
	"archive/tar"
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	corev1mocks "github.com/Dynatrace/dynatrace-operator/src/mocks/k8s.io/client-go/kubernetes/typed/core/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

func TestLogCollector(t *testing.T) {
	fakeClientSet := fake.NewSimpleClientset(
		createPod("pod1"),
		createPod("pod2"),
		createPod("pod3"),
		createPod("pod4"),
		&corev1.Pod{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Pod",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "rogue-foreign-pod",
				Namespace: "dynatrace",
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{Name: "trojan-container"},
				},
			},
		})

	logBuffer := bytes.Buffer{}

	tarBuffer := bytes.Buffer{}
	supportArchive := tarball{
		tarWriter: tar.NewWriter(&tarBuffer),
	}

	logCollector := newLogCollector(context.TODO(),
		newSupportArchiveLogger(&logBuffer),
		supportArchive,
		fakeClientSet.CoreV1().Pods("dynatrace"),
		defaultOperatorAppName)

	require.NoError(t, logCollector.Do())

	supportArchive.tarWriter.Close()

	tarReader := tar.NewReader(&tarBuffer)
	tarHeader, err := tarReader.Next()
	require.NoError(t, err)
	assert.Equal(t, "logs/pod1/container1.log", tarHeader.Name)

	tarHeader, err = tarReader.Next()
	require.NoError(t, err)
	assert.Equal(t, "logs/pod1/container1_previous.log", tarHeader.Name)

	tarHeader, err = tarReader.Next()
	require.NoError(t, err)
	assert.Equal(t, "logs/pod1/container2.log", tarHeader.Name)

	tarHeader, err = tarReader.Next()
	require.NoError(t, err)
	assert.Equal(t, "logs/pod1/container2_previous.log", tarHeader.Name)

	tarHeader, err = tarReader.Next()
	require.NoError(t, err)
	assert.Equal(t, "logs/pod2/container1.log", tarHeader.Name)

	tarHeader, err = tarReader.Next()
	require.NoError(t, err)
	assert.Equal(t, "logs/pod2/container1_previous.log", tarHeader.Name)

	tarHeader, err = tarReader.Next()
	require.NoError(t, err)
	assert.Equal(t, "logs/pod2/container2.log", tarHeader.Name)

	tarHeader, err = tarReader.Next()
	require.NoError(t, err)
	assert.Equal(t, "logs/pod2/container2_previous.log", tarHeader.Name)
}

//go:generate mockery --case=snake --srcpkg=k8s.io/client-go/kubernetes/typed/core/v1 --with-expecter --name=PodInterface --output ../../mocks/k8s.io/client-go/kubernetes/typed/core/v1
func TestLogCollectorPodListError(t *testing.T) {
	context := context.Background()
	logBuffer := bytes.Buffer{}
	tarBuffer := bytes.Buffer{}
	supportArchive := tarball{
		tarWriter: tar.NewWriter(&tarBuffer),
	}
	defer supportArchive.tarWriter.Close()

	mockedPods := corev1mocks.NewPodInterface(t)
	mockedPods.EXPECT().
		List(context, createPodListOptions()).
		Return(nil, assert.AnError)
	logCollector := newLogCollector(context,
		newSupportArchiveLogger(&logBuffer),
		supportArchive,
		mockedPods,
		defaultOperatorAppName)
	require.Error(t, logCollector.Do())
}

func TestLogCollectorGetPodFail(t *testing.T) {
	context := context.Background()

	fakeClientSet := fake.NewSimpleClientset(
		createPod("pod1"),
		createPod("pod2"))

	logBuffer := bytes.Buffer{}

	tarBuffer := bytes.Buffer{}
	supportArchive := tarball{
		tarWriter: tar.NewWriter(&tarBuffer),
	}
	defer supportArchive.tarWriter.Close()

	mockedPods := corev1mocks.NewPodInterface(t)
	listOptions := createPodListOptions()
	mockedPods.EXPECT().
		List(context, listOptions).
		Return(fakeClientSet.CoreV1().Pods("dynatrace").List(context, listOptions))
	mockedPods.EXPECT().
		Get(context, "pod1", metav1.GetOptions{}).
		Return(nil, assert.AnError)
	mockedPods.EXPECT().
		Get(context, "pod2", metav1.GetOptions{}).
		Return(nil, assert.AnError)

	logCollector := newLogCollector(context, newSupportArchiveLogger(&logBuffer), supportArchive, mockedPods, defaultOperatorAppName)
	require.NoError(t, logCollector.Do())
}

func TestLogCollectorGetLogsFail(t *testing.T) {
	context := context.Background()

	fakeClientSet := fake.NewSimpleClientset(
		createPod("pod1"),
		createPod("pod2"))

	logBuffer := bytes.Buffer{}

	tarBuffer := bytes.Buffer{}
	supportArchive := tarball{
		tarWriter: tar.NewWriter(&tarBuffer),
	}
	defer supportArchive.tarWriter.Close()

	mockedPods := corev1mocks.NewPodInterface(t)
	listOptions := createPodListOptions()
	getOptions := metav1.GetOptions{}

	listCall := mockedPods.EXPECT().
		List(context, listOptions).
		Return(fakeClientSet.CoreV1().Pods("dynatrace").List(context, listOptions))
	getPod1Call := mockedPods.EXPECT().
		Get(context, "pod1", getOptions).
		NotBefore(listCall.Call).
		Return(fakeClientSet.CoreV1().Pods("dynatrace").Get(context, "pod1", getOptions))
	getLogsPod1Container1Call := mockedPods.EXPECT().
		GetLogs("pod1", createGetPodLogOptions("container1", false)).
		NotBefore(getPod1Call).
		Return(nil, assert.AnError)
	getPreviousLogsPod1Container1Call := mockedPods.EXPECT().
		GetLogs("pod1", createGetPodLogOptions("container1", true)).
		NotBefore(getLogsPod1Container1Call).
		Return(nil, assert.AnError)
	getLogsPod1Container2Call := mockedPods.EXPECT().
		GetLogs("pod1", createGetPodLogOptions("container2", false)).
		NotBefore(getPreviousLogsPod1Container1Call).
		Return(nil, assert.AnError)
	getPreviousLogsPod1Container2Call := mockedPods.EXPECT().
		GetLogs("pod1", createGetPodLogOptions("container2", true)).
		NotBefore(getLogsPod1Container2Call).
		Return(nil, assert.AnError)
	getPod2Call := mockedPods.EXPECT().
		Get(context, "pod2", getOptions).
		NotBefore(getPreviousLogsPod1Container2Call).
		Return(fakeClientSet.CoreV1().Pods("dynatrace").Get(context, "pod2", getOptions))
	getLogsPod2Container1Call := mockedPods.EXPECT().
		GetLogs("pod2", createGetPodLogOptions("container1", false)).
		NotBefore(getPod2Call).
		Return(nil, assert.AnError)
	getPreviousLogsPod2Container1Call := mockedPods.EXPECT().
		GetLogs("pod2", createGetPodLogOptions("container1", true)).
		NotBefore(getLogsPod2Container1Call).
		Return(nil, assert.AnError)
	getLogsPod2Container2Call := mockedPods.EXPECT().
		GetLogs("pod2", createGetPodLogOptions("container2", false)).
		NotBefore(getPreviousLogsPod2Container1Call).
		Return(nil, assert.AnError)
	mockedPods.EXPECT().
		GetLogs("pod2", createGetPodLogOptions("container2", true)).
		NotBefore(getLogsPod2Container2Call).
		Return(nil, assert.AnError)

	logCollector := newLogCollector(context, newSupportArchiveLogger(&logBuffer), supportArchive, mockedPods, defaultOperatorAppName)
	require.NoError(t, logCollector.Do())

	assert.Contains(t, logBuffer.String(), "Unable to retrieve log stream for pod pod1, container container1")
	assert.Contains(t, logBuffer.String(), "Unable to retrieve log stream for pod pod1, container container2")
	assert.Contains(t, logBuffer.String(), "Unable to retrieve log stream for pod pod2, container container1")
	assert.Contains(t, logBuffer.String(), "Unable to retrieve log stream for pod pod2, container container2")
}

func TestLogCollectorNoAbortOnError(t *testing.T) {
	context := context.Background()

	fakeClientSet := fake.NewSimpleClientset(
		createPod("pod1"),
		createPod("pod2"))

	logBuffer := bytes.Buffer{}

	tarBuffer := bytes.Buffer{}
	supportArchive := tarball{
		tarWriter: tar.NewWriter(&tarBuffer),
	}
	defer supportArchive.tarWriter.Close()

	mockedPods := corev1mocks.NewPodInterface(t)
	listOptions := createPodListOptions()
	getOptions := metav1.GetOptions{}

	listCall := mockedPods.EXPECT().
		List(context, listOptions).
		Return(fakeClientSet.CoreV1().Pods("dynatrace").List(context, listOptions))
	getPod1Call := mockedPods.EXPECT().
		Get(context, "pod1", getOptions).
		NotBefore(listCall.Call).
		Return(nil, assert.AnError)

	getPod2Call := mockedPods.EXPECT().
		Get(context, "pod2", mock.Anything).
		NotBefore(getPod1Call).
		Return(fakeClientSet.CoreV1().Pods("dynatrace").Get(context, "pod2", getOptions))

	getLogsPod2Container1Call := mockedPods.EXPECT().
		GetLogs("pod2", createGetPodLogOptions("container1", false)).
		NotBefore(getPod2Call).
		Return(nil, assert.AnError)
	getPreviousLogsPod2Container1Call := mockedPods.EXPECT().
		GetLogs("pod2", createGetPodLogOptions("container1", true)).
		NotBefore(getLogsPod2Container1Call).
		Return(fakeClientSet.CoreV1().Pods("dynatrace").GetLogs("pod2", createGetPodLogOptions("container1", true)))
	getLogsPod2Container2Call := mockedPods.EXPECT().
		GetLogs("pod2", createGetPodLogOptions("container2", false)).
		NotBefore(getPreviousLogsPod2Container1Call).
		Return(fakeClientSet.CoreV1().Pods("dynatrace").GetLogs("pod2", createGetPodLogOptions("container2", false)))
	mockedPods.EXPECT().
		GetLogs("pod2", createGetPodLogOptions("container2", true)).
		NotBefore(getLogsPod2Container2Call).
		Return(nil, assert.AnError)

	logCollector := newLogCollector(context, newSupportArchiveLogger(&logBuffer), supportArchive, mockedPods, defaultOperatorAppName)
	require.NoError(t, logCollector.Do())

	supportArchive.tarWriter.Close()

	tarReader := tar.NewReader(&tarBuffer)

	tarHeader, err := tarReader.Next()
	require.NoError(t, err)
	assert.Equal(t, "logs/pod2/container1_previous.log", tarHeader.Name)

	tarHeader, err = tarReader.Next()
	require.NoError(t, err)
	assert.Equal(t, "logs/pod2/container2.log", tarHeader.Name)

	_, err = tarReader.Next()
	require.ErrorIs(t, err, io.EOF)
}

func createPod(name string) *corev1.Pod {
	const namespace = "dynatrace"
	return &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				kubeobjects.AppNameLabel: defaultOperatorAppName,
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "container1"},
				{Name: "container2"},
			},
		},
	}
}

func createPodListOptions() metav1.ListOptions {
	return metav1.ListOptions{
		TypeMeta: metav1.TypeMeta{
			Kind: "pod",
		},
		LabelSelector: "app.kubernetes.io/name=dynatrace-operator",
	}
}

func createGetPodLogOptions(container string, previous bool) *corev1.PodLogOptions {
	return &corev1.PodLogOptions{
		Container: container,
		Follow:    false,
		Previous:  previous,
	}
}
