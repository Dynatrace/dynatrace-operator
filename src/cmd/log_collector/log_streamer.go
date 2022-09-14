package log_collector

import (
	"fmt"
	"io"
	"os"
	"sync"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const logFileName = "/tmp/dynatrace-operator/%s_%s.gz"

func streamLogs(ctx *logCollectorContext) error {
	ctx.wg = sync.WaitGroup{}

	listOptions := metav1.ListOptions{
		TypeMeta: metav1.TypeMeta{
			Kind: "pod",
		},
	}
	podList, err := ctx.clientSet.CoreV1().Pods(ctx.namespaceName).List(ctx.ctx, listOptions)
	if err != nil {
		return err
	}

	for _, pod := range podList.Items {
		podGetOptions := metav1.GetOptions{}
		pod, err := ctx.clientSet.CoreV1().Pods(ctx.namespaceName).Get(ctx.ctx, pod.Name, podGetOptions)
		if err != nil {
			return err
		}
		streamPodLogs(ctx, pod)
	}

	ctx.wg.Wait()
	return nil
}

func streamPodLogs(ctx *logCollectorContext, pod *corev1.Pod) {
	for _, container := range pod.Spec.Containers {
		streamContainerLogs(ctx, pod, container)
	}
}

func streamContainerLogs(ctx *logCollectorContext, pod *corev1.Pod, container corev1.Container) {

	fileName := fmt.Sprintf(logFileName, pod.Name, container.Name)

	zipFile, err := os.Create(fileName)
	if err != nil {
		logErrorf("could not create file for %s: %v", fileName, err)
		return
	}
	writer := zipFile
	//	writer := gzip.NewWriter(zipFile)

	ctx.wg.Add(1)
	go func() {
		logInfof("Start log collection for %s", fileName)

		defer writer.Close()
		//	defer zipFile.Close()

		podLogOpts := corev1.PodLogOptions{
			Container: container.Name,
			Follow:    true,
		}
		req := ctx.clientSet.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &podLogOpts)

		podLogs, err := req.Stream(ctx.ctx)
		if err != nil {
			logErrorf("Error in opening stream: %v", err)
			return
		}
		defer podLogs.Close()

		_, err = io.Copy(writer, podLogs)
		if err != nil {
			logErrorf("Error writing to tarball: %v", err)
			return
		}

		logInfof("Successfully collected logs %s", fileName)
	}()
}
