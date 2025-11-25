package supportarchive

import (
	"archive/zip"
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8slabel"
	corev1mock "github.com/Dynatrace/dynatrace-operator/test/mocks/k8s.io/client-go/kubernetes/typed/core/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

func TestLogCollector(t *testing.T) {
	testLogCollection(t, true)
}

func TestManagedByLogsIgnored(t *testing.T) {
	testLogCollection(t, false)
}

func testLogCollection(t *testing.T, collectManagedLogs bool) {
	fakeClientSet := fake.NewSimpleClientset(
		createPod("pod1", k8slabel.AppNameLabel),
		createPod("pod2", k8slabel.AppNameLabel),
		createPod("pod3", k8slabel.AppManagedByLabel),
		createPod("pod4", k8slabel.AppManagedByLabel),
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

	buffer := bytes.Buffer{}
	supportArchive := newZipArchive(bufio.NewWriter(&buffer))

	logCollector := newLogCollector(context.TODO(),
		newSupportArchiveLogger(&logBuffer),
		supportArchive,
		fakeClientSet.CoreV1().Pods("dynatrace"),
		defaultOperatorAppName,
		collectManagedLogs)

	require.NoError(t, logCollector.Do())

	require.NoError(t, supportArchive.Close())

	zipReader, err := zip.NewReader(bytes.NewReader(buffer.Bytes()), int64(buffer.Len()))
	require.NoError(t, err)

	podNumbers := []int{1, 2}
	if collectManagedLogs {
		podNumbers = []int{1, 2, 3, 4}
	}

	containerNumber := []int{1, 2}
	fileIndex := 0

	for _, podNumber := range podNumbers {
		for _, containerNumber := range containerNumber {
			assert.Equal(t, fmt.Sprintf("logs/pod%d/container%d.log", podNumber, containerNumber),
				zipReader.File[fileIndex].Name)

			fileIndex++
			assert.Equal(t, fmt.Sprintf("logs/pod%d/container%d_previous.log", podNumber, containerNumber),
				zipReader.File[fileIndex].Name)

			fileIndex++
		}
	}

	if collectManagedLogs {
		assert.Len(t, zipReader.File, 16)
	} else {
		assert.Len(t, zipReader.File, 8)
	}
}

func TestLogCollectorPodListError(t *testing.T) {
	ctx := context.Background()
	logBuffer := bytes.Buffer{}
	buffer := bytes.Buffer{}

	supportArchive := newZipArchive(bufio.NewWriter(&buffer))
	defer assertNoErrorOnClose(t, supportArchive)

	mockedPods := corev1mock.NewPodInterface(t)
	mockedPods.EXPECT().
		List(ctx, createPodListOptions(k8slabel.AppNameLabel)).
		Return(nil, assert.AnError)

	logCollector := newLogCollector(ctx,
		newSupportArchiveLogger(&logBuffer),
		supportArchive,
		mockedPods,
		defaultOperatorAppName,
		true)
	require.Error(t, logCollector.Do())
}

func assertNoErrorOnClose(t *testing.T, closer io.Closer) {
	require.NoError(t, closer.Close())
}

func TestLogCollectorGetPodFail(t *testing.T) {
	ctx := context.Background()

	fakeClientSet := fake.NewSimpleClientset(
		createPod("pod1", k8slabel.AppNameLabel),
		createPod("pod2", k8slabel.AppNameLabel),
		createPod("oneagent", k8slabel.AppManagedByLabel))

	logBuffer := bytes.Buffer{}

	buffer := bytes.Buffer{}

	supportArchive := newZipArchive(bufio.NewWriter(&buffer))
	defer assertNoErrorOnClose(t, supportArchive)

	mockedPods := corev1mock.NewPodInterface(t)
	listOptionsAppName := createPodListOptions(k8slabel.AppNameLabel)
	listOptionsManagedByOperator := createPodListOptions(k8slabel.AppManagedByLabel)

	mockedPods.EXPECT().
		List(ctx, listOptionsAppName).
		Return(fakeClientSet.CoreV1().Pods("dynatrace").List(ctx, listOptionsAppName))
	mockedPods.EXPECT().
		List(ctx, listOptionsManagedByOperator).
		Return(fakeClientSet.CoreV1().Pods("dynatrace").List(ctx, listOptionsManagedByOperator))
	mockedPods.EXPECT().
		Get(ctx, "pod1", metav1.GetOptions{}).
		Return(nil, assert.AnError)
	mockedPods.EXPECT().
		Get(ctx, "pod2", metav1.GetOptions{}).
		Return(nil, assert.AnError)
	mockedPods.EXPECT().
		Get(ctx, "oneagent", metav1.GetOptions{}).
		Return(nil, assert.AnError)

	logCollector := newLogCollector(ctx, newSupportArchiveLogger(&logBuffer), supportArchive, mockedPods, defaultOperatorAppName, true)
	require.NoError(t, logCollector.Do())
}

func TestLogCollectorGetLogsFail(t *testing.T) {
	ctx := context.Background()

	fakeClientSet := fake.NewSimpleClientset(
		createPod("pod1", k8slabel.AppNameLabel),
		createPod("pod2", k8slabel.AppNameLabel))

	logBuffer := bytes.Buffer{}

	buffer := bytes.Buffer{}

	supportArchive := newZipArchive(bufio.NewWriter(&buffer))
	defer assertNoErrorOnClose(t, supportArchive)

	mockedPods := corev1mock.NewPodInterface(t)
	listOptionsAppName := createPodListOptions(k8slabel.AppNameLabel)
	listOptionsManagedByOperator := createPodListOptions(k8slabel.AppManagedByLabel)

	getOptions := metav1.GetOptions{}

	listAppNameCall := mockedPods.EXPECT().
		List(ctx, listOptionsAppName).
		Return(fakeClientSet.CoreV1().Pods("dynatrace").List(ctx, listOptionsAppName))
	mockedPods.EXPECT().
		List(ctx, listOptionsManagedByOperator).
		NotBefore(listAppNameCall.Call).
		Return(fakeClientSet.CoreV1().Pods("dynatrace").List(ctx, listOptionsManagedByOperator))

	getPod1Call := mockedPods.EXPECT().
		Get(ctx, "pod1", getOptions).
		NotBefore(listAppNameCall.Call).
		Return(fakeClientSet.CoreV1().Pods("dynatrace").Get(ctx, "pod1", getOptions))
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
		Get(ctx, "pod2", getOptions).
		NotBefore(getPreviousLogsPod1Container2Call).
		Return(fakeClientSet.CoreV1().Pods("dynatrace").Get(ctx, "pod2", getOptions))
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

	logCollector := newLogCollector(ctx, newSupportArchiveLogger(&logBuffer), supportArchive, mockedPods, defaultOperatorAppName, true)
	require.NoError(t, logCollector.Do())

	assert.Contains(t, logBuffer.String(), "Unable to retrieve log stream for pod pod1, container container1")
	assert.Contains(t, logBuffer.String(), "Unable to retrieve log stream for pod pod1, container container2")
	assert.Contains(t, logBuffer.String(), "Unable to retrieve log stream for pod pod2, container container1")
	assert.Contains(t, logBuffer.String(), "Unable to retrieve log stream for pod pod2, container container2")
}

func TestLogCollectorNoAbortOnError(t *testing.T) {
	ctx := context.Background()

	fakeClientSet := fake.NewSimpleClientset(
		createPod("pod1", k8slabel.AppNameLabel),
		createPod("pod2", k8slabel.AppNameLabel))

	logBuffer := bytes.Buffer{}

	buffer := bytes.Buffer{}
	supportArchive := newZipArchive(bufio.NewWriter(&buffer))

	mockedPods := corev1mock.NewPodInterface(t)
	listOptionsAppName := createPodListOptions(k8slabel.AppNameLabel)
	listOptionsManagedByOperator := createPodListOptions(k8slabel.AppManagedByLabel)
	getOptions := metav1.GetOptions{}

	listCall := mockedPods.EXPECT().
		List(ctx, listOptionsAppName).
		Return(fakeClientSet.CoreV1().Pods("dynatrace").List(ctx, listOptionsAppName))
	mockedPods.EXPECT().
		List(ctx, listOptionsManagedByOperator).
		NotBefore(listCall.Call).
		Return(fakeClientSet.CoreV1().Pods("dynatrace").List(ctx, listOptionsManagedByOperator))

	getPod1Call := mockedPods.EXPECT().
		Get(ctx, "pod1", getOptions).
		NotBefore(listCall.Call).
		Return(nil, assert.AnError)

	getPod2Call := mockedPods.EXPECT().
		Get(ctx, "pod2", mock.Anything).
		NotBefore(getPod1Call).
		Return(fakeClientSet.CoreV1().Pods("dynatrace").Get(ctx, "pod2", getOptions))

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

	logCollector := newLogCollector(ctx, newSupportArchiveLogger(&logBuffer), supportArchive, mockedPods, defaultOperatorAppName, true)
	require.NoError(t, logCollector.Do())

	assertNoErrorOnClose(t, supportArchive)

	zipReader, err := zip.NewReader(bytes.NewReader(buffer.Bytes()), int64(buffer.Len()))
	require.NoError(t, err)
	assert.Equal(t, "logs/pod2/container1_previous.log", zipReader.File[0].Name)
	assert.Equal(t, "logs/pod2/container2.log", zipReader.File[1].Name)
}

func createPod(name string, labelKey string) *corev1.Pod {
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
				labelKey: defaultOperatorAppName,
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

func createPodListOptions(labelKey string) metav1.ListOptions {
	return metav1.ListOptions{
		TypeMeta: metav1.TypeMeta{
			Kind: "pod",
		},
		LabelSelector: fmt.Sprintf("%s=dynatrace-operator", labelKey),
	}
}

func createGetPodLogOptions(container string, previous bool) *corev1.PodLogOptions {
	return &corev1.PodLogOptions{
		Container: container,
		Follow:    false,
		Previous:  previous,
	}
}
