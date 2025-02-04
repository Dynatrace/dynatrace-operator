package zip

import (
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer/common"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
)

var (
	log = logd.Get().WithName("oneagent-zip")
)

// Checks if a file is under /agent/conf
// Different archive file implementations return different output, this function handles the differences
// tar.Header.Name == "./path/to/file"
// zip.File.Name == "path/to/file"
func isAgentConfFile(fileName string) bool {
	return strings.HasPrefix(fileName, "./"+common.AgentConfDirPath) || strings.HasPrefix(fileName, common.AgentConfDirPath)
}
