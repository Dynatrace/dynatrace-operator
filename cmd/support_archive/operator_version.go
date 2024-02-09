package support_archive

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/util/logger"
	"github.com/Dynatrace/dynatrace-operator/pkg/version"
)

const operatorVersionCollectorName = "operatorVersionCollector"

type operatorVersionCollector struct {
	collectorCommon
}

func newOperatorVersionCollector(log logger.DtLogger, supportArchive archiver) collector {
	return operatorVersionCollector{
		collectorCommon{
			log:            log,
			supportArchive: supportArchive,
		},
	}
}

func (vc operatorVersionCollector) Do() error {
	logInfof(vc.log, "Storing operator version into %s", OperatorVersionFileName)

	versionString := fmt.Sprintf("version: %s\ngitCommit: %s\nbuildDate: %s\ngoVersion %s\nplatform %s/%s\n",
		version.Version,
		version.Commit,
		version.BuildDate,
		runtime.Version(),
		runtime.GOOS, runtime.GOARCH)

	err := vc.supportArchive.addFile(OperatorVersionFileName, strings.NewReader(versionString))
	if err != nil {
		return err
	}

	return nil
}

func (vc operatorVersionCollector) Name() string {
	return operatorVersionCollectorName
}
