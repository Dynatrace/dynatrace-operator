package zip

import (
	"github.com/Dynatrace/dynatrace-operator/src/controllers/csi/metadata"
	"github.com/spf13/afero"
)

type Extractor interface {
	ExtractZip(sourceFile afero.File, targetDir string) error
	ExtractGzip(sourceFilePath, targetDir string) error
}

func NewOnAgentExtractor(fs afero.Fs, pathResolver metadata.PathResolver) Extractor {
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
	extractor.fs.RemoveAll(extractor.pathResolver.AgentTempUnzipDir())
}

func (extractor OneAgentExtractor) moveToTargetDir(targetDir string) error {
	log.Info("moving unpacked archive to target", "targetDir", targetDir)
	return extractor.fs.Rename(extractor.pathResolver.AgentTempUnzipDir(), targetDir)
}
