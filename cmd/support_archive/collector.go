package support_archive

import "github.com/go-logr/logr"

type collector interface {
	Name() string
	Do() error
}

type collectorCommon struct {
	log            logr.Logger
	supportArchive archiver
}
