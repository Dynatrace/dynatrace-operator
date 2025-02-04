package support_archive

import (
	"context"
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/installconfig"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/labels"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientgocorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

const logCollectorName = "logCollector"

type logCollector struct {
	collectorCommon

	ctx                context.Context
	pods               clientgocorev1.PodInterface
	appName            string
	collectManagedLogs bool
}

func newLogCollector(context context.Context, log logd.Logger, supportArchive archiver, pods clientgocorev1.PodInterface, appName string, collectManagedLogs bool) collector { //nolint:revive // argument-limit doesn't apply to constructors
	return logCollector{
		collectorCommon: collectorCommon{
			log:            log,
			supportArchive: supportArchive,
		},
		ctx:                context,
		pods:               pods,
		appName:            appName,
		collectManagedLogs: collectManagedLogs,
	}
}

func (lc logCollector) Do() error {
	if !installconfig.GetModules().Supportability {
		logInfof(lc.log, "%s", installconfig.GetModuleValidationErrorMessage("Log Collection"))

		return nil
	}

	logInfof(lc.log, "Starting log collection")

	podList, err := lc.getPodList(labels.AppNameLabel)
	if err != nil {
		return err
	}

	if lc.collectManagedLogs {
		managedByOperatorPodList, err := lc.getPodList(labels.AppManagedByLabel)
		if err != nil {
			return err
		}

		podList.Items = append(podList.Items, managedByOperatorPodList.Items...)
	}

	podGetOptions := metav1.GetOptions{}

	for _, podItem := range podList.Items {
		pod, err := lc.pods.Get(lc.ctx, podItem.Name, podGetOptions)
		if err != nil {
			logErrorf(lc.log, err, "Unable to get pod info for %s", podItem.Name)
		} else {
			lc.collectPodLogs(pod)
		}
	}

	return nil
}

func (lc logCollector) Name() string {
	return logCollectorName
}

func (lc logCollector) getPodList(labelKey string) (*corev1.PodList, error) {
	listOptions := metav1.ListOptions{
		TypeMeta: metav1.TypeMeta{
			Kind: "pod",
		},
		LabelSelector: fmt.Sprintf("%s=%s", labelKey, lc.appName),
	}

	podList, err := lc.pods.List(lc.ctx, listOptions)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return podList, nil
}

func (lc logCollector) collectPodLogs(pod *corev1.Pod) {
	for _, container := range pod.Spec.Containers {
		podLogOpts := corev1.PodLogOptions{
			Container: container.Name,
			Follow:    false,
		}
		lc.collectContainerLogs(pod, container, podLogOpts)

		podLogOpts.Previous = true
		lc.collectContainerLogs(pod, container, podLogOpts)
	}
}

func (lc logCollector) collectContainerLogs(pod *corev1.Pod, container corev1.Container, logOptions corev1.PodLogOptions) {
	req := lc.pods.GetLogs(pod.Name, &logOptions)
	if req == nil {
		logErrorf(lc.log, errors.Errorf("Unable to retrieve log stream for pod %s, container %s", pod.Name, container.Name), "")

		return
	}

	podLogs, err := req.Stream(lc.ctx)
	if logOptions.Previous && err != nil {
		if k8serrors.IsBadRequest(err) { // Prevent logging of "previous terminated container not found" error
			return
		}

		logErrorf(lc.log, err, "error getting previous logs")

		return
	} else if err != nil {
		logErrorf(lc.log, err, "error in opening stream")

		return
	}

	defer podLogs.Close()

	fileName := buildLogFileName(pod, container, logOptions)

	err = lc.supportArchive.addFile(fileName, podLogs)
	if err != nil {
		logErrorf(lc.log, err, "error writing to tarball")

		return
	}

	logInfof(lc.log, "Successfully collected logs %s", fileName)
}

func buildLogFileName(pod *corev1.Pod, container corev1.Container, logOptions corev1.PodLogOptions) string {
	if logOptions.Previous {
		return fmt.Sprintf("%s/%s/%s_previous.log", LogsDirectoryName, pod.Name, container.Name)
	}

	return fmt.Sprintf("%s/%s/%s.log", LogsDirectoryName, pod.Name, container.Name)
}
