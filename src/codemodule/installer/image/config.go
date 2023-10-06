package image

import (
	"github.com/Dynatrace/dynatrace-operator/src/util/logger"
	"path/filepath"

	dtcsi "github.com/Dynatrace/dynatrace-operator/src/controllers/csi"
)

var (
	CacheDir = filepath.Join(dtcsi.DataPath, "cache")
	log      = logger.Factory.GetLogger("oneagent-image")
)
