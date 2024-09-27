package support_archive

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes/scheme"
	clientgocorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

const (
	diagLogCollectorName = "diagLogCollector"
	eecExtensionsPath    = "/var/lib/dynatrace/remotepluginmodule/log/extensions"
	fileNotFoundMarker   = "<NOT FOUND>"
	eecPodName           = "dynatrace-extensions-controller"
	eecContainerName     = "extensions-controller"
)

type executionResult struct {
	StdOut *bytes.Buffer
	StdErr *bytes.Buffer
}

type diagLogCollector struct {
	ctx    context.Context
	pods   clientgocorev1.PodInterface
	config *rest.Config
	collectorCommon
}

var (
	eecPodNotFoundError = errors.New("eec pod not found")
)

func newDiagLogCollector(context context.Context, config *rest.Config, log logd.Logger, supportArchive archiver, pods clientgocorev1.PodInterface) collector {
	return diagLogCollector{
		collectorCommon: collectorCommon{
			log:            log,
			supportArchive: supportArchive,
		},
		ctx:    context,
		config: config,
		pods:   pods,
	}
}

func (collector diagLogCollector) Name() string {
	return diagLogCollectorName
}

func (collector diagLogCollector) getControllerPod() (*corev1.Pod, error) {
	listOptions := metav1.ListOptions{
		TypeMeta: metav1.TypeMeta{
			Kind: "pod",
		},
		LabelSelector: fmt.Sprintf("%s=%s", "app.kubernetes.io/name", eecPodName),
	}

	podList, err := collector.pods.List(collector.ctx, listOptions)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	if len(podList.Items) == 0 {
		logInfof(collector.log, "EEC pod not found, diagnostic logs will not be collected")

		return nil, eecPodNotFoundError
	}

	if len(podList.Items) != 1 {
		err := errors.New(fmt.Sprintf("expected 1 EEC pod, got %d", len(podList.Items)))
		logErrorf(collector.log, err, "diagnostic logs will not be collected")

		return nil, err
	}

	return &podList.Items[0], nil
}

func (collector diagLogCollector) Do() error {
	eecPod, err := collector.getControllerPod()
	if errors.Is(err, eecPodNotFoundError) {
		return nil
	}

	if err != nil {
		return err
	}

	logFiles, err := collector.findLogFilesRecursively(eecPod.Name, eecPod.Namespace, eecExtensionsPath)
	if err != nil {
		logErrorf(collector.log, err, "log files lookup failed")

		return err
	}

	for _, logFilePath := range logFiles {
		if err := collector.copyDiagnosticFile(eecPod.Name, eecPod.Namespace, logFilePath); err != nil {
			logErrorf(collector.log, err, "failed to copy %s", logFilePath)

			return err
		}

		logInfof(collector.log, "Successfully collected EEC diagnostic logs logs/%s%s", eecPod.Name, logFilePath)
	}

	return nil
}

func (collector diagLogCollector) findLogFilesRecursively(podName string, podNamespace string, rootPath string) ([]string, error) {
	// command := []string{"/usr/bin/sh", "-c", "[ -d " + rootPath + " ] && ls -R1 " + rootPath + " || echo '" + fileNotFoundMarker + "'"}
	command := []string{"/usr/bin/sh", "-c", "if [ -d '" + rootPath + "' ]; then ls -R1 '" + rootPath + "' ; else echo '" + fileNotFoundMarker + "' ; fi"}

	executionResult, err := collector.executeRemoteCommand(collector.ctx, podName, podNamespace, eecContainerName, command)
	if err != nil {
		return []string{}, err
	}

	if strings.HasPrefix(executionResult.StdOut.String(), fileNotFoundMarker) {
		return []string{}, nil
	}

	zipFilePath := collector.buildZipFilePath(podName, "ls.txt")

	var buf bytes.Buffer
	tee := io.TeeReader(executionResult.StdOut, &buf)

	err = collector.supportArchive.addFile(zipFilePath, tee)
	if err != nil {
		logErrorf(collector.log, err, "error writing to tarball")

		return []string{}, err
	}

	logFiles := []string{}
	pwd := ""

	for _, line := range strings.Split(buf.String(), "\n") {
		line = strings.TrimSpace(line)
		if len(line) == 0 {
			continue
		}

		if line[0] == '/' && line[len(line)-1] == ':' {
			pwd = strings.TrimSuffix(line, ":") + "/"

			continue
		}

		if !strings.HasSuffix(line, ".log") {
			continue
		}

		logFiles = append(logFiles, pwd+line)
	}

	return logFiles, nil
}

func (collector diagLogCollector) copyDiagnosticFile(podName string, podNamespace string, eecDiagLogPath string) error {
	command := []string{"/usr/bin/sh", "-c", "[ -e " + eecDiagLogPath + " ] && cat " + eecDiagLogPath + " || echo '" + fileNotFoundMarker + "'"}

	executionResult, err := collector.executeRemoteCommand(collector.ctx, podName, podNamespace, eecContainerName, command)
	if err != nil {
		return err
	}

	if strings.HasPrefix(executionResult.StdOut.String(), fileNotFoundMarker) {
		return nil
	}

	// eecDiagLogPath is an absolute path, remove leading slash to avoid '//' in zipFilePath
	zipFilePath := collector.buildZipFilePath(podName, eecDiagLogPath[1:])

	err = collector.supportArchive.addFile(zipFilePath, executionResult.StdOut)
	if err != nil {
		logErrorf(collector.log, err, "error writing to tarball")

		return err
	}

	return nil
}

func (collector diagLogCollector) buildZipFilePath(podName string, fileName string) string {
	return fmt.Sprintf("%s/%s/%s", LogsDirectoryName, podName, fileName)
}

func (collector diagLogCollector) executeRemoteCommand(ctx context.Context, podName string, podNamespace string, containerName string, command []string) (*executionResult, error) {
	sch := scheme.Scheme
	parameterCodec := scheme.ParameterCodec

	gvk := schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "Pod",
	}

	httpClient, err := rest.HTTPClientFor(collector.config)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	restClient, err := apiutil.RESTClientForGVK(gvk, false, collector.config, serializer.NewCodecFactory(sch), httpClient)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	result := &executionResult{
		StdOut: &bytes.Buffer{},
		StdErr: &bytes.Buffer{},
	}

	req := restClient.Post().
		Resource("pods").
		Name(podName).
		Namespace(podNamespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: containerName,
			Command:   command,
			Stdin:     true,
			Stdout:    true,
			Stderr:    true,
			TTY:       false,
		}, parameterCodec)

	executor, err := remotecommand.NewSPDYExecutor(collector.config, "POST", req.URL())
	if err != nil {
		return nil, errors.WithStack(err)
	}

	err = executor.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdin:  &bytes.Buffer{},
		Stdout: result.StdOut,
		Stderr: result.StdErr,
		Tty:    false,
	})

	return result, errors.WithStack(err)
}
