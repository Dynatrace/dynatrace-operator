package zip

import (
	"os"

	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/metadata"
	"github.com/spf13/afero"
)

type Extractor interface {
	ExtractZip(sourceFile afero.File, targetDir string) error
	ExtractGzip(sourceFilePath, targetDir string) error
}

func NewOneAgentExtractor(fs afero.Fs, pathResolver metadata.PathResolver) Extractor {
	return &OneAgentExtractor{
		fs:           fs,
		pathResolver: pathResolver,
	}
}

type OneAgentExtractor struct {
	fs           afero.Fs
	pathResolver metadata.PathResolver
}

func (extractor OneAgentExtractor) cleanTempZipDir() {
	extractor.fs.RemoveAll(extractor.pathResolver.AgentTempUnzipRootDir())
}

func (extractor OneAgentExtractor) moveToTargetDir(targetDir string) error {
	defer extractor.cleanTempZipDir()

	log.Info("moving unpacked archive to target", "targetDir", targetDir)

	_, err := extractor.fs.Stat(extractor.pathResolver.AgentTempUnzipDir())
	if err == nil {
		return extractor.fs.Rename(extractor.pathResolver.AgentTempUnzipDir(), targetDir)
	}

	if !os.IsNotExist(err) {
		return err
	}

	return extractor.fs.Rename(extractor.pathResolver.AgentTempUnzipRootDir(), targetDir)
}
