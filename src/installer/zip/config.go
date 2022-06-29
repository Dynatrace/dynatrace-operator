package zip

import (
	"strings"

	"github.com/Dynatrace/dynatrace-operator/src/installer/common"
	"github.com/Dynatrace/dynatrace-operator/src/logger"
)

var (
	log = logger.NewDTLogger().WithName("oneagent-installer-zip")
)

func isRuxitConfFile(fileName string) bool {
	return strings.HasSuffix(fileName, common.RuxitConfFileName)
}
