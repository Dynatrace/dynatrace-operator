package version

import (
	"fmt"
	"runtime"

	"github.com/Dynatrace/dynatrace-operator/pkg/util/logd"
)

var (

	// AppName contains the name of the application
	AppName = "dynatrace-operator"

	// Version contains the version of the Operator. Assigned externally.
	Version = "snapshot"

	// Commit indicates the Git commit hash the binary was build from. Assigned externally.
	Commit = ""

	// BuildDate is the date when the binary was build. Assigned externally.
	BuildDate = ""

	log = logd.Get().WithName("version")
)

// LogVersion logs metadata about the Operator.
func LogVersion() {
	LogVersionToLogger(log)
}

func LogVersionToLogger(log logd.Logger) {
	log.Info(AppName,
		"version", Version,
		"gitCommit", Commit,
		"buildDate", BuildDate,
		"goVersion", runtime.Version(),
		"platform", fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	)
}
