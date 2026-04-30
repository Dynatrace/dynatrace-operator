package zip

import (
	"context"
	"os"

	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/metadata"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
)

type Extractor interface {
	ExtractZip(ctx context.Context, sourceFile *os.File, targetDir string) error
	ExtractGzip(ctx context.Context, sourceFilePath, targetDir string) error
}

func NewOneAgentExtractor(pathResolver metadata.PathResolver) Extractor {
	return &OneAgentExtractor{
		pathResolver: pathResolver,
	}
}

type OneAgentExtractor struct {
	pathResolver metadata.PathResolver
}

func (extractor OneAgentExtractor) cleanTempZipDir() {
	os.RemoveAll(extractor.pathResolver.AgentTempUnzipRootDir())
}

func (extractor OneAgentExtractor) moveToTargetDir(ctx context.Context, targetDir string) error {
	log := logd.FromContext(ctx)

	defer extractor.cleanTempZipDir()

	log.Info("moving unpacked archive to target", "targetDir", targetDir)

	_, err := os.Stat(extractor.pathResolver.AgentTempUnzipDir())
	if err == nil {
		return os.Rename(extractor.pathResolver.AgentTempUnzipDir(), targetDir)
	}

	if !os.IsNotExist(err) {
		return err
	}

	return os.Rename(extractor.pathResolver.AgentTempUnzipRootDir(), targetDir)
}
