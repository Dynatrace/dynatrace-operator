package zip

import (
	"os"

	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/metadata"
)

type Extractor interface {
	ExtractZip(sourceFile *os.File, targetDir string) error
	ExtractGzip(sourceFilePath, targetDir string) error
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

func (extractor OneAgentExtractor) moveToTargetDir(targetDir string) error {
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
