package symlink

import (
	"github.com/Dynatrace/dynatrace-operator/src/logger"
)

const (
	// example match: 1.239.14.20220325-164521
	versionRegexp = `^(\d+)\.(\d+)\.(\d+)\.(\d+)-(\d+)$`
	binDir        = "/agent/bin"
)

var (
	log = logger.NewDTLogger().WithName("oneagent-installer-symlink")
)
