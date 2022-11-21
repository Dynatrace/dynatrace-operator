package support_archive

import (
	"context"

	"github.com/go-logr/logr"
)

type supportArchiveContext struct {
	ctx              context.Context
	namespaceName    string
	toStdout         bool
	log              logr.Logger
	tarballTargetDir string
}
