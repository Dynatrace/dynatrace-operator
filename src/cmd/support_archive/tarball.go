package support_archive

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"
)

const tarFileName = "%s/operator-support-archive-%s.tgz"

type tarball struct {
	tarWriter  *tar.Writer
	gzipWriter *gzip.Writer
}

func newTarball(target io.Writer) tarball {
	newTarball := tarball{}

	newTarball.gzipWriter = gzip.NewWriter(target)
	newTarball.tarWriter = tar.NewWriter(newTarball.gzipWriter)
	return newTarball
}

func (t tarball) close() {
	if t.tarWriter != nil {
		t.tarWriter.Close()
	}
	if t.gzipWriter != nil {
		t.gzipWriter.Close()
	}
}

func (t tarball) addFile(fileName string, file io.Reader) error {
	buffer := &bytes.Buffer{}
	_, err := io.Copy(buffer, file)
	if err != nil {
		return errors.WithMessagef(err, "could not copy data from source for '%s'", fileName)
	}

	header := &tar.Header{
		Name: fileName,
		Size: int64(buffer.Len()),
		Mode: int64(fs.ModePerm),
	}

	err = t.tarWriter.WriteHeader(header)
	if err != nil {
		return errors.WithMessagef(err, "could not write header for file '%s'", fileName)
	}

	_, err = io.Copy(t.tarWriter, buffer)
	if err != nil {
		return errors.WithMessagef(err, "could not copy the file '%s' data to the tarball", fileName)
	}
	return nil
}

func createTarballTargetFile(useStdout bool, targetDir string) (*os.File, error) {
	if useStdout {
		return os.Stdout, nil
	} else {
		tarFile, err := createTarFile(targetDir)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return tarFile, nil
	}
}

func createTarFile(targetDir string) (*os.File, error) {
	tarballFilePath := fmt.Sprintf(tarFileName, targetDir, time.Now().Format(time.RFC3339))
	tarballFilePath = strings.Replace(tarballFilePath, ":", "_", -1)

	tarFile, err := os.Create(tarballFilePath)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return tarFile, nil
}
