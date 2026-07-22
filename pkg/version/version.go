// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package version

import (
	"crypto/fips140"
	"fmt"
	"runtime"
	"runtime/debug"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/arch"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
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
)

// LogVersion logs metadata about the Operator.
func LogVersion() {
	LogVersionToLogger(logd.Get().WithName("version"))
}

func LogVersionToLogger(log logd.Logger) {
	keysAndValues := []any{"version", Version,
		"gitCommit", Commit,
		"buildDate", BuildDate,
		"goVersion", runtime.Version(),
		"platform", fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)}

	if fips140.Enabled() {
		keysAndValues = append(keysAndValues, "fips140", "FIPS 140-3 Mode Enabled with version: "+fips140.Version())
	}

	log.Info(AppName, keysAndValues...)

	// SetMemoryLimit returns the previously set memory limit. A negative input does not adjust the limit, and allows for retrieval of the currently set memory limit.
	log.Debug("GOMEMLIMIT", "valueInBytes", debug.SetMemoryLimit(-1))
}

func UserAgent() string {
	return AppName + "/" + strings.TrimSuffix(Version, "-"+arch.ImageArch)
}
