package zip

import (
	"strings"

	"github.com/Dynatrace/dynatrace-operator/src/installer/common"
	"github.com/Dynatrace/dynatrace-operator/src/logger"
)

var (
	log = logger.NewDTLogger().WithName("oneagent-installer-zip")
)

// Checks if a file is under /agent/conf
// Different archive file implementations return different output, this function handles the differences
// tar.Header.Name == "./path/to/file"
// zip.File.Name == "path/to/file"
func isAgentConfFile(fileName string) bool {
	return strings.HasPrefix(fileName, "./"+common.AgentConfDirPath) || strings.HasPrefix(fileName, common.AgentConfDirPath)
}
