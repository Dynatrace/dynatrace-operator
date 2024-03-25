package support_archive

import (
	"context"
	"fmt"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/labels"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	clientgocorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

const logCollectorName = "logCollector"

type logCollector struct {
	collectorCommon

	ctx                 context.Context
	pods                clientgocorev1.PodInterface
	appName             string
	collectManagedLogs  bool
	collectInjectedPods bool
	namespace           string
	nodeName            string
}

func newLogCollector(context context.Context, log logd.Logger, supportArchive archiver, pods clientgocorev1.PodInterface, appName string, collectManagedLogs, collectInjectedPods bool, nodeName, namespace string) collector { //nolint:revive // argument-limit doesn't apply to constructors
	return logCollector{
		collectorCommon: collectorCommon{
			log:            log,
			supportArchive: supportArchive,
		},
		ctx:                 context,
		pods:                pods,
		appName:             appName,
		collectManagedLogs:  collectManagedLogs,
		collectInjectedPods: collectInjectedPods,
		namespace:           namespace,
		nodeName:            nodeName,
	}
}

func (collector logCollector) Do() error {
	logInfof(collector.log, "Starting log collection")

	podList, err := collector.getPodList(labels.AppNameLabel)
	if err != nil {
		return err
	}

	if collector.collectManagedLogs {
		managedByOperatorPodList, err := collector.getPodList(labels.AppManagedByLabel)
		if err != nil {
			return err
		}

		podList.Items = append(podList.Items, managedByOperatorPodList.Items...)
	}

	if collector.collectInjectedPods && collector.nodeName == "" {
		injectedPodList, err := collector.getInjectedPodList(collector.namespace)
		if err != nil {
			return err
		}

		podList.Items = append(podList.Items, injectedPodList.Items...)
	}

	if collector.nodeName != "" {
		nodePodList, err := collector.getPodsFromSpecificNode()
		if err != nil {
			return err
		}

		podList.Items = append(podList.Items, nodePodList.Items...)
	}

	podGetOptions := metav1.GetOptions{}

	for _, podItem := range podList.Items {
		pod, err := collector.pods.Get(collector.ctx, podItem.Name, podGetOptions)
		if err != nil {
			logErrorf(collector.log, err, "Unable to get pod info for %s", podItem.Name)
		} else {
			collector.collectPodLogs(pod)
		}
	}

	return nil
}

func (collector logCollector) Name() string {
	return logCollectorName
}

func (collector logCollector) getPodList(labelKey string) (*corev1.PodList, error) {
	listOptions := metav1.ListOptions{
		TypeMeta: metav1.TypeMeta{
			Kind: "pod",
		},
		LabelSelector: fmt.Sprintf("%s=%s", labelKey, collector.appName),
	}

	podList, err := collector.pods.List(collector.ctx, listOptions)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return podList, nil
}

func (collector logCollector) getInjectedPodList(namespace string) (*corev1.PodList, error) {
	listOptions := metav1.ListOptions{
		TypeMeta: metav1.TypeMeta{
			Kind: "pod",
		},
		FieldSelector: fields.OneTermEqualSelector("metadata.namespace", namespace).String(),
	}

	podList, err := collector.pods.List(collector.ctx, listOptions)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return podList, nil
}

func (collector logCollector) getPodsFromSpecificNode() (*corev1.PodList, error) {
	listOptions := metav1.ListOptions{
		TypeMeta: metav1.TypeMeta{
			Kind: "pod",
		},
		FieldSelector: fields.OneTermEqualSelector("spec.nodeName", collector.nodeName).String(),
	}

	podList, err := collector.pods.List(collector.ctx, listOptions)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return podList, nil
}

func (collector logCollector) collectPodLogs(pod *corev1.Pod) {
	for _, container := range pod.Spec.Containers {
		podLogOpts := corev1.PodLogOptions{
			Container: container.Name,
			Follow:    false,
		}
		collector.collectContainerLogs(pod, container, podLogOpts)

		podLogOpts.Previous = true
		collector.collectContainerLogs(pod, container, podLogOpts)
	}

	if len(pod.Spec.InitContainers) > 0 {
		for _, initContainer := range pod.Spec.InitContainers {
			podLogOpts := corev1.PodLogOptions{
				Container: initContainer.Name,
				Follow:    false,
			}
			collector.collectContainerLogs(pod, initContainer, podLogOpts)
		}
	}
}

func (collector logCollector) collectContainerLogs(pod *corev1.Pod, container corev1.Container, logOptions corev1.PodLogOptions) {
	req := collector.pods.GetLogs(pod.Name, &logOptions)
	if req == nil {
		logErrorf(collector.log, errors.Errorf("Unable to retrieve log stream for pod %s, container %s", pod.Name, container.Name), "")

		return
	}

	podLogs, err := req.Stream(collector.ctx)
	if logOptions.Previous && err != nil {
		if k8serrors.IsBadRequest(err) { // Prevent logging of "previous terminated container not found" error
			return
		}

		logErrorf(collector.log, err, "error getting previous logs")

		return
	} else if err != nil {
		logErrorf(collector.log, err, "error in opening stream")

		return
	}

	defer podLogs.Close()

	fileName := buildLogFileName(pod, container, logOptions)

	err = collector.supportArchive.addFile(fileName, podLogs)
	if err != nil {
		logErrorf(collector.log, err, "error writing to tarball")

		return
	}

	logInfof(collector.log, "Successfully collected logs %s", fileName)
}

func buildLogFileName(pod *corev1.Pod, container corev1.Container, logOptions corev1.PodLogOptions) string {
	if logOptions.Previous {
		return fmt.Sprintf("%s/%s/%s_previous.log", LogsDirectoryName, pod.Name, container.Name)
	}

	return fmt.Sprintf("%s/%s/%s.log", LogsDirectoryName, pod.Name, container.Name)
}
