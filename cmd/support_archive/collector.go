package support_archive

import "github.com/Dynatrace/dynatrace-operator/pkg/util/logger"

type collector interface {
	Name() string
	Do() error
}

type collectorCommon struct {
	log            logger.DtLogger
	supportArchive archiver
}
