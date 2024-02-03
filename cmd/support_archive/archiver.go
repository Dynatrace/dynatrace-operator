package support_archive

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/klauspost/compress/zip"
	"github.com/pkg/errors"
)

const zipArchiveFileName = "%s/operator-support-archive-%s.zip"

type archiver interface {
	addFile(fileName string, reader io.Reader) error
}

type archiveCloser interface {
	archiver
	io.Closer
}

func newZipArchive(target io.Writer) archiveCloser {
	newZipArchive := zipArchive{writer: zip.NewWriter(target)}

	return newZipArchive
}

type zipArchive struct {
	writer *zip.Writer
}

func (z zipArchive) addFile(fileName string, reader io.Reader) error {
	w, err := z.writer.Create(fileName)
	if err != nil {
		return errors.WithMessagef(err, "could not create file '%s' in zip archive", fileName)
	}

	_, err = io.Copy(w, reader)
	if err != nil {
		return errors.WithMessagef(err, "could not copy the file '%s' data to the zip archive", fileName)
	}

	err = z.writer.Flush()
	if err != nil {
		return err
	}

	return nil
}

func (z zipArchive) Close() error {
	if z.writer != nil {
		err := z.writer.Close()
		if err != nil {
			return err
		}
	}

	return nil
}

func createZipArchiveTargetFile(useStdout bool, targetDir string) (*os.File, error) {
	if useStdout {
		return os.Stdout, nil
	} else {
		archiveFile, err := createZipArchiveFile(targetDir)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		return archiveFile, nil
	}
}

func createZipArchiveFile(targetDir string) (*os.File, error) {
	archiveFilePath := fmt.Sprintf(zipArchiveFileName, targetDir, time.Now().Format(time.RFC3339))
	archiveFilePath = strings.ReplaceAll(archiveFilePath, ":", "_")

	tarFile, err := os.Create(archiveFilePath)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return tarFile, nil
}
