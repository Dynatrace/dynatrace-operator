package support_archive

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/installconfig"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/labels"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apilabels "k8s.io/apimachinery/pkg/labels"
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
	EecPodName           = "dynatrace-extensions-controller"
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
	appName            string
	collectManagedLogs bool
}

var (
	eecPodNotFoundError = errors.New("eec pod not found")
)

func newDiagLogCollector(context context.Context, config *rest.Config, log logd.Logger, supportArchive archiver, pods clientgocorev1.PodInterface, appName string, collectManagedLogs bool) collector { //nolint:revive
	return diagLogCollector{
		collectorCommon: collectorCommon{
			log:            log,
			supportArchive: supportArchive,
		},
		ctx:                context,
		config:             config,
		pods:               pods,
		appName:            appName,
		collectManagedLogs: collectManagedLogs,
	}
}

func (collector diagLogCollector) Name() string {
	return diagLogCollectorName
}

func (collector diagLogCollector) getControllerPodList() (*corev1.PodList, error) {
	ls := metav1.LabelSelector{
		MatchLabels: map[string]string{
			labels.AppNameLabel:      EecPodName,
			labels.AppManagedByLabel: collector.appName,
		},
	}

	listOptions := metav1.ListOptions{
		TypeMeta: metav1.TypeMeta{
			Kind: "pod",
		},
		LabelSelector: apilabels.Set(ls.MatchLabels).String(),
	}

	podList, err := collector.pods.List(collector.ctx, listOptions)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	if len(podList.Items) == 0 {
		logInfof(collector.log, "EEC pod not found, diagnostic logs will not be collected")

		return nil, eecPodNotFoundError
	}

	if !collector.collectManagedLogs {
		logInfof(collector.log, "%s", "EEC diagnostic logs will not be collected")

		return nil, eecPodNotFoundError
	}

	return podList, nil
}

func (collector diagLogCollector) Do() error {
	if !installconfig.GetModules().Supportability {
		logInfof(collector.log, "%s", installconfig.GetModuleValidationErrorMessage("EEC Diagnostic Log Collection"))

		return nil
	}

	eecPodList, err := collector.getControllerPodList()
	if errors.Is(err, eecPodNotFoundError) {
		return nil
	}

	if err != nil {
		return err
	}

	for _, eecPod := range eecPodList.Items {
		logFiles, err := collector.findLogFilesRecursively(eecPod.Name, eecPod.Namespace, eecExtensionsPath)
		if err != nil {
			logErrorf(collector.log, err, "log files lookup failed, podName: %s", eecPod.Name)

			continue
		}

		for _, logFilePath := range logFiles {
			if err := collector.copyDiagnosticFile(eecPod.Name, eecPod.Namespace, logFilePath); err != nil {
				logErrorf(collector.log, err, "failed to copy %s from pod: %s", logFilePath, eecPod.Name)
			} else {
				logInfof(collector.log, "Successfully collected EEC diagnostic logs logs/%s%s", eecPod.Name, logFilePath)
			}
		}
	}

	return nil
}

func (collector diagLogCollector) findLogFilesRecursively(podName string, podNamespace string, rootPath string) ([]string, error) {
	command := []string{"/usr/bin/sh", "-c", "if [ -d '" + rootPath + "' ]; then ls -R1 '" + rootPath + "' ; else echo '" + fileNotFoundMarker + "' ; fi"}

	executionResult, err := collector.executeRemoteCommand(collector.ctx, podName, podNamespace, eecContainerName, command)
	if err != nil {
		return []string{}, err
	}

	if strings.HasPrefix(executionResult.StdOut.String(), fileNotFoundMarker) {
		return []string{}, nil
	}

	zipFilePath := BuildZipFilePath(podName, "ls.txt")

	var buf bytes.Buffer
	tee := io.TeeReader(executionResult.StdOut, &buf)

	err = collector.supportArchive.addFile(zipFilePath, tee)
	if err != nil {
		logErrorf(collector.log, err, "error writing to tarball")

		return []string{}, err
	}

	/*
		Output of ls command is used to find all *.log files recursively
		in the /var/lib/dynatrace/remotepluginmodule/log/extensions directory.

		$ ls -R1 /var/lib/dynatrace/remotepluginmodule/log/extensions

		/var/lib/dynatrace/remotepluginmodule/log/extensions/:
		datasources
		diagnostics
		oneagent-logmon-detailed.log
		oneagent-logmon-general.log
		ruxitagent_extensionsmodule_14.0.log

		/var/lib/dynatrace/remotepluginmodule/log/extensions/datasources:

		/var/lib/dynatrace/remotepluginmodule/log/extensions/diagnostics:
		diag_executor.log
	*/

	logFiles := []string{}
	pwd := ""

	for _, line := range strings.Split(buf.String(), "\n") {
		line = strings.TrimSpace(line)
		if len(line) == 0 {
			continue
		}

		// new subdirectory
		if line[0] == '/' && line[len(line)-1] == ':' {
			pwd = strings.TrimSuffix(line, ":") + "/"

			continue
		}

		if !strings.HasSuffix(line, ".log") {
			continue
		}

		// add absolute path of the file to the list
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
	zipFilePath := BuildZipFilePath(podName, eecDiagLogPath[1:])

	err = collector.supportArchive.addFile(zipFilePath, executionResult.StdOut)
	if err != nil {
		logErrorf(collector.log, err, "error writing to tarball")

		return err
	}

	return nil
}

func BuildZipFilePath(podName string, fileName string) string {
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
