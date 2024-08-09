package support_archive

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/functional"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
)

const (
	diagLogCollectorName = "diagLogCollector"
	eecDiagnosticPath    = "/var/lib/dynatrace/remotepluginmodule/log/extensions/diagnostics"
	fileNotFoundMarker   = "<NOT FOUND>"
)

type diagLogCollector struct {
	ctx    context.Context
	config *rest.Config
	collectorCommon
}

func newDiagLogCollector(context context.Context, config *rest.Config, log logd.Logger, supportArchive archiver) collector {
	return diagLogCollector{
		collectorCommon: collectorCommon{
			log:            log,
			supportArchive: supportArchive,
		},
		ctx:    context,
		config: config,
	}
}

func (collector diagLogCollector) Name() string {
	return diagLogCollectorName
}

func (collector diagLogCollector) Do() error {
	environmentResources, err := resources.New(collector.config)
	if err != nil {
		logErrorf(collector.log, err, "error creating resources")

		return err
	}

	var pods corev1.PodList
	err = environmentResources.WithNamespace("dynatrace").List(collector.ctx, &pods)

	if err != nil {
		logErrorf(collector.log, err, "error listing pods")

		return err
	}

	eecPods := functional.Filter(pods.Items, func(podItem corev1.Pod) bool {
		return strings.HasPrefix(podItem.Name, "dynatrace-extensions-controller-")
	})

	if len(eecPods) == 0 {
		logInfof(collector.log, "EEC pod not found, diagnostic logs will not be collected")

		return nil
	}

	if len(eecPods) != 1 {
		err := errors.New("unexpected number of eec pods")
		logErrorf(collector.log, err, "unexpected number of eec pods %d", len(eecPods))

		return err
	}

	fileName := "diag_executor.log"
	if err := collector.copyDiagnosticFile(environmentResources, eecPods[0], fileName); err != nil {
		logErrorf(collector.log, err, "failed to copy %s", fileName)

		return err
	}

	for i := range 10 {
		fileName := fmt.Sprintf("diag_executor.%d.log", i)
		if err := collector.copyDiagnosticFile(environmentResources, eecPods[0], fileName); err != nil {
			logErrorf(collector.log, err, "failed to copy %s", fileName)

			return err
		}
	}

	logInfof(collector.log, "Successfully collected EEC diagnostic logs")

	return nil
}

func (collector diagLogCollector) copyDiagnosticFile(environmentResources *resources.Resources, eecPod corev1.Pod, fileName string) error {
	path := filepath.Join(eecDiagnosticPath, fileName)

	command := []string{"/usr/bin/sh", "-c", "[ -e " + path + " ] && cat " + path + " || echo '" + fileNotFoundMarker + "'"}

	executionResult, err := execute(collector.ctx, environmentResources,
		eecPod,
		"extensions-controller",
		command...,
	)

	if err != nil {
		return err
	}

	if executionResult.StdOut.Len() > 0 && executionResult.StdOut.Len() < len(fileNotFoundMarker)+2 {
		if strings.TrimSpace(executionResult.StdOut.String()) == fileNotFoundMarker {
			return nil
		}
	}

	zipFileName := collector.buildLogFileName(&eecPod, fileName)

	err = collector.supportArchive.addFile(zipFileName, executionResult.StdOut)
	if err != nil {
		logErrorf(collector.log, err, "error writing to tarball")

		return err
	}

	return nil
}

func (collector diagLogCollector) buildLogFileName(pod *corev1.Pod, fileName string) string {
	return fmt.Sprintf("%s/%s/%s.log", LogsDirectoryName, pod.Name, fileName)
}

type ExecutionResult struct {
	StdOut *bytes.Buffer
	StdErr *bytes.Buffer
}

func execute(ctx context.Context, resource *resources.Resources, pod corev1.Pod, container string, command ...string) (*ExecutionResult, error) {
	result := &ExecutionResult{
		StdOut: &bytes.Buffer{},
		StdErr: &bytes.Buffer{},
	}

	err := resource.ExecInPod(
		ctx,
		pod.Namespace,
		pod.Name,
		container,
		command,
		result.StdOut,
		result.StdErr,
	)

	if err != nil {
		return result, errors.WithMessagef(errors.WithStack(err),
			"stdout:\n%s\nstderr:\n%s", result.StdOut.String(), result.StdErr.String())
	}

	return result, nil
}
