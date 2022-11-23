package support_archive

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/src/version"
)

const operatorVersionCollectorName = "operatorVersionCollector"
const operatorVersionFileName = "operator-version.txt"

type operatorVersionCollector struct {
	collectorCommon
}

func (vc operatorVersionCollector) Do() error {
	logInfof(vc.log, "Storing operator version into %s", operatorVersionFileName)

	versionString := fmt.Sprintf("version: %s\ngitCommit: %s\nbuildDate: %s\ngoVersion %s\nplatform %s/%s\n",
		version.Version,
		version.Commit,
		version.BuildDate,
		runtime.Version(),
		runtime.GOOS, runtime.GOARCH)
	vc.supportArchive.addFile(operatorVersionFileName, strings.NewReader(versionString))

	return nil
}

func (vc operatorVersionCollector) Name() string {
	return operatorVersionCollectorName
}
