package support_archive

import (
	"io"

	"github.com/go-logr/logr"
)

const (
	supportArchiveCollectorName = "supportarchiveoutput"
)

type supportArchiveOutputCollector struct {
	collectorCommon
	output io.Reader
}

func newSupportArchiveOutputCollector(log logr.Logger, supportArchive archiver, logBuffer io.Reader) collector {
	return supportArchiveOutputCollector{
		collectorCommon: collectorCommon{
			log:            log,
			supportArchive: supportArchive,
		},
		output: logBuffer,
	}
}

func (vc supportArchiveOutputCollector) Do() error {
	logInfof(vc.log, "Storing support archive output into %s", SupportArchiveOutputFileName)
	vc.supportArchive.addFile(SupportArchiveOutputFileName, vc.output)

	return nil
}
func (vc supportArchiveOutputCollector) Name() string {
	return supportArchiveCollectorName
}
