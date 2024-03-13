package image

import (
	"path/filepath"

	dtcsi "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
)

var (
	CacheDir = filepath.Join(dtcsi.DataPath, "cache")
	log      = logd.Get().WithName("oneagent-image")
)
