package support_archive

import "github.com/Dynatrace/dynatrace-operator/pkg/util/logd"

type collector interface {
	Name() string
	Do() error
}

type collectorCommon struct {
	supportArchive archiver
	log            logd.Logger
}
