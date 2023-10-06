package image

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/util/logger"
	"path/filepath"

	dtcsi "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi"
)

var (
	CacheDir = filepath.Join(dtcsi.DataPath, "cache")
	log      = logger.Factory.GetLogger("oneagent-image")
)
