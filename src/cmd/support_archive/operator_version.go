package support_archive

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/src/version"
)

const versionFileName = "operator-version.txt"

func collectOperatorVersion(_ *supportArchiveContext, tarball *tarball) error {
	versionString := fmt.Sprintf("version: %s\ngitCommit: %s\nbuildDate: %s\ngoVersion %s\nplatform %s/%s\n",
		version.Version,
		version.Commit,
		version.BuildDate,
		runtime.Version(),
		runtime.GOOS, runtime.GOARCH)
	tarball.addFile(versionFileName, strings.NewReader(versionString))
	return nil
}
