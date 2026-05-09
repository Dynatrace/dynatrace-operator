package image

import (
	"path/filepath"

	dtcsi "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi"
)

var (
	CacheDir = filepath.Join(dtcsi.DataPath, "cache")
)
