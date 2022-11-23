package support_archive

import (
	"context"

	"github.com/go-logr/logr"
)

type supportArchiveContext struct {
	ctx            context.Context
	namespaceName  string
	log            logr.Logger
	supportArchive tarball
}
