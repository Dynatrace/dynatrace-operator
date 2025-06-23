package supportarchive

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
)

type collector interface {
	Name() string
	Do() error
}

type collectorCommon struct {
	supportArchive archiver
	log            logd.Logger
}
