package support_archive

import (
	"context"
	"fmt"
	"io"

	"github.com/Dynatrace/dynatrace-operator/pkg/util/logger"
	"golang.org/x/exp/rand"
	clientgocorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

const loadSimCollectorName = "loadSimCollector"

type loadSimCollector struct {
	context context.Context
	pods    clientgocorev1.PodInterface
	collectorCommon
	fileSize  int
	fileCount int
}

func newLoadSimCollector(ctx context.Context, log logger.DtLogger, supportArchive archiver, fileSize int, fileCount int, pods clientgocorev1.PodInterface) collector { //nolint:revive // argument-limit doesn't apply to constructors
	return loadSimCollector{
		collectorCommon: collectorCommon{
			log:            log,
			supportArchive: supportArchive,
		},
		context:   ctx,
		fileSize:  fileSize,
		fileCount: fileCount,
		pods:      pods,
	}
}

func (collector loadSimCollector) Do() error {
	if collector.fileCount <= 0 {
		return nil
	}

	logInfof(collector.log, "Starting load simulation (%d files, %d MB/file)", collector.fileCount, collector.fileSize)
	collector.createSimulatedLogFiles()

	return nil
}

func (collector loadSimCollector) Name() string {
	return loadSimCollectorName
}

func (collector loadSimCollector) createSimulatedLogFiles() {
	for i := 0; i < collector.fileCount; i++ {
		fileName := buildLoadsimFileName(i)

		lg := loadGenerator{
			fileSize: collector.fileSize,
		}

		err := collector.supportArchive.addFile(fileName, &lg)
		if err != nil {
			logErrorf(collector.log, err, "error writing simulated load to zip")

			return
		}

		logInfof(collector.log, "Successfully collected logs %s", fileName)
	}
}

func buildLoadsimFileName(i int) string {
	return fmt.Sprintf("%s/loadsim/%s-%d.log", LogsDirectoryName, "loadsim", i)
}

type loadGenerator struct {
	fileSize int
}

var letters = []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789 ")

func (lg *loadGenerator) Read(p []byte) (n int, err error) {
	if lg.fileSize > 0 {
		i := 0
		for ; i < len(p) && lg.fileSize > 0; i++ {
			p[i] = letters[rand.Intn(len(letters))]
			lg.fileSize--
		}

		return i, nil
	}

	return 0, io.EOF
}
