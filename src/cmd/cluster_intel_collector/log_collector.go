package cluster_intel_collector

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const tarFileName = "%s/operator-cic-%s.tgz"

func collectLogs(cicCtx *intelCollectorContext, tarball *intelTarball) error {
	podList, err := getPodList(cicCtx)
	if err != nil {
		return err
	}

	for _, pod := range podList.Items {
		podGetOptions := metav1.GetOptions{}
		pod, err := cicCtx.clientSet.CoreV1().Pods(cicCtx.namespaceName).Get(cicCtx.ctx, pod.Name, podGetOptions)
		if err != nil {
			return err
		}
		getPodLogs(cicCtx, tarball, pod)
	}
	return nil
}

func getPodList(cicCtx *intelCollectorContext) (*corev1.PodList, error) {
	listOptions := metav1.ListOptions{
		TypeMeta: metav1.TypeMeta{
			Kind: "pod",
		},
	}

	podList, err := cicCtx.clientSet.CoreV1().Pods(cicCtx.namespaceName).List(cicCtx.ctx, listOptions)
	if err != nil {
		return nil, err
	}
	return podList, nil
}

func getPodLogs(cicCtx *intelCollectorContext, tarball *intelTarball, pod *corev1.Pod) {
	for _, container := range pod.Spec.Containers {
		podLogOpts := corev1.PodLogOptions{
			Container: container.Name,
			Follow:    false,
		}
		getContainerLogs(cicCtx, tarball, pod, container, podLogOpts)

		podLogOpts.Previous = true
		getContainerLogs(cicCtx, tarball, pod, container, podLogOpts)
	}
}

func getContainerLogs(cicCtx *intelCollectorContext, tarball *intelTarball, pod *corev1.Pod, container corev1.Container, logOptions corev1.PodLogOptions) {
	req := cicCtx.clientSet.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &logOptions)

	podLogs, err := req.Stream(cicCtx.ctx)

	switch {
	case logOptions.Previous && err != nil:
		// Soften error message for previous pods a bit, so users don't get nervous. Previous pods often just didn't exist.
		logInfof("logs for previous pod not found: %v", err)
		return
	case err != nil:
		logErrorf("error in opening stream: %v", err)
		return
	}
	defer podLogs.Close()

	fileName := buildLogFileName(pod, container, logOptions)
	err = tarball.addFile(fileName, podLogs)

	if err != nil {
		logErrorf("error writing to tarball: %v", err)
		return
	}

	logInfof("Successfully collected logs %s", fileName)
}

func buildLogFileName(pod *corev1.Pod, container corev1.Container, logOptions corev1.PodLogOptions) string {
	prevPostFix := ""
	if logOptions.Previous {
		prevPostFix = "_previous"
	}
	return fmt.Sprintf("%s_%s%s.log", pod.Name, container.Name, prevPostFix)
}
