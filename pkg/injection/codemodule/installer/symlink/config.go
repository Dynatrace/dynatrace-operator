package symlink

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
)

const (
	// example match: 1.239.14.20220325-164521
	versionRegexp = `^(\d+)\.(\d+)\.(\d+)\.(\d+)-(\d+)$`
	binDir        = "/agent/bin"
)

var (
	log = logd.Get().WithName("oneagent-symlink")
)
