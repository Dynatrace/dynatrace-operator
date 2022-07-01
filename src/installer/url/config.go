package url

import (
	"path/filepath"

	"github.com/Dynatrace/dynatrace-operator/src/logger"
)

var (
	log              = logger.NewDTLogger().WithName("oneagent-url-installer")
	standaloneBinDir = filepath.Join("mnt", "bin")
)

const (
	VersionLatest = "latest"
)
