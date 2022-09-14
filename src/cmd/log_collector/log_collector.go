package log_collector

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const tarFileName = "/tmp/dynatrace-operator/operator-logs-%s.tgz"

func createTarball() (*tar.Writer, func(), error) {
	tarballFilePath := fmt.Sprintf(tarFileName, time.Now().Format(time.RFC3339))
	tarballFilePath = strings.Replace(tarballFilePath, ":", "_", -1)

	tarball, err := os.Create(tarballFilePath)
	if err != nil {
		return nil, nil, fmt.Errorf("could not create tarball file '%s', got error '%w'", tarballFilePath, err)
	}
	gzipWriter := gzip.NewWriter(tarball)
	tarWriter := tar.NewWriter(gzipWriter)

	logInfof("Created log tarball %s", tarballFilePath)
	return tarWriter, func() {
		tarWriter.Close()
		gzipWriter.Close()
		tarball.Close()
	}, nil
}

func collectLogs(ctx *logCollectorContext) error {

	tarball, closer, err := createTarball()
	if err != nil {
		return err
	}
	defer closer()

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
		getPodLogs(ctx, tarball, pod)
	}
	return nil
}

func getPodLogs(ctx *logCollectorContext, tarball *tar.Writer, pod *corev1.Pod) {

	for _, container := range pod.Spec.Containers {
		getContainerLogs(ctx, tarball, pod, container)
	}
}

func getContainerLogs(ctx *logCollectorContext, tarball *tar.Writer, pod *corev1.Pod, container corev1.Container) {
	podLogOpts := corev1.PodLogOptions{
		Container: container.Name,
	}
	req := ctx.clientSet.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &podLogOpts)

	podLogs, err := req.Stream(ctx.ctx)
	if err != nil {
		logErrorf("error in opening stream: %v", err)
		return
	}
	defer podLogs.Close()

	fileName := fmt.Sprintf("%s_%s.log", pod.Name, container.Name)

	err = addFileToTarWriter(fileName, tarball, podLogs)

	if err != nil {
		logErrorf("error writing to tarball: %v", err)
		return
	}

	logInfof("Successfully collected logs %s", fileName)
}

func addFileToTarWriter(fileName string, tarWriter *tar.Writer, logFile io.ReadCloser) error {

	logBuffer := &bytes.Buffer{}
	io.Copy(logBuffer, logFile)

	header := &tar.Header{
		Name: fileName,
		Size: int64(logBuffer.Len()),
		Mode: int64(fs.ModePerm),
	}

	err := tarWriter.WriteHeader(header)
	if err != nil {
		return fmt.Errorf("could not write header for file '%s', got error '%w'", fileName, err)
	}

	_, err = io.Copy(tarWriter, logBuffer)
	if err != nil {
		return fmt.Errorf("could not copy the file '%s' data to the tarball, got error '%w'", fileName, err)
	}

	return nil
}
