package support_archive

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/src/version"
)

const versionFileName = "operator-version.txt"

func collectOperatorVersion(ctx supportArchiveContext) error {
	logInfof(ctx.log, "Storing operator version into %s", versionFileName)

	versionString := fmt.Sprintf("version: %s\ngitCommit: %s\nbuildDate: %s\ngoVersion %s\nplatform %s/%s\n",
		version.Version,
		version.Commit,
		version.BuildDate,
		runtime.Version(),
		runtime.GOOS, runtime.GOARCH)
	ctx.supportArchive.addFile(versionFileName, strings.NewReader(versionString))

	return nil
}
