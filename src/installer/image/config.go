package image

import (
	"path/filepath"

	dtcsi "github.com/Dynatrace/dynatrace-operator/src/controllers/csi"
	"github.com/Dynatrace/dynatrace-operator/src/logger"
)

const (
	rawPolicy = `{
		"default": [
			{
				"type": "insecureAcceptAnything"
			}
		]
	}
	`
)

var (
	CacheDir = filepath.Join(dtcsi.DataPath, "cache")
	log      = logger.NewDTLogger().WithName("oneagent-image-installer")
)
