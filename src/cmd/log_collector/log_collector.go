package log_collector

import (
	"bytes"
	"fmt"
	"io"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const namespace = "dynatrace"

func collectLogs(ctx *logCollectorContext) error {

	listOptions := metav1.ListOptions{
		TypeMeta: metav1.TypeMeta{
			Kind: "pod",
		},
	}
	podList, err := ctx.clientSet.CoreV1().Pods(namespace).List(ctx.ctx, listOptions)
	if err != nil {
		return err
	}

	for _, pod := range podList.Items {
		podGetOptions := metav1.GetOptions{}
		pod, err := ctx.clientSet.CoreV1().Pods(namespace).Get(ctx.ctx, pod.Name, podGetOptions)
		if err != nil {
			return err
		}
		getPodLogs(ctx, pod)
	}

	return nil
}

func getPodLogs(ctx *logCollectorContext, pod *corev1.Pod) {

	for _, container := range pod.Spec.Containers {
		fmt.Printf("\nPod: %s/%s\n", pod.Name, container.Name)
		getContainerLogs(ctx, pod, container)
	}
}

func getContainerLogs(ctx *logCollectorContext, pod *corev1.Pod, container corev1.Container) {
	podLogOpts := corev1.PodLogOptions{
		Container: container.Name,
	}
	req := ctx.clientSet.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &podLogOpts)

	podLogs, err := req.Stream(ctx.ctx)
	if err != nil {
		fmt.Printf("error in opening stream: %v\n", err)
		return
	}
	defer podLogs.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		fmt.Printf("error in copy information from podLogs to buf: %v\n", err)
		return
	}

	fmt.Printf("%s\n", buf.String())
}
