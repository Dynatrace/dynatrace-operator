package image

import (
	"path/filepath"

	dtcsi "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/logger"
)

var (
	CacheDir = filepath.Join(dtcsi.DataPath, "cache")
	log      = logger.Factory.GetLogger("oneagent-image")
)
