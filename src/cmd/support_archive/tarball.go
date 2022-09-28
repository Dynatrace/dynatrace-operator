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
)

const tarFileName = "%s/operator-support-archive-%s.tgz"

type tarball struct {
	tarWriter  *tar.Writer
	gzipWriter *gzip.Writer
	tarFile    *os.File
}

func newTarball(ctx *supportArchiveContext) (*tarball, error) {
	var err error
	tarball := tarball{}
	if tarball.tarFile, err = selectAndCreateTargetFile(ctx); err != nil {
		return nil, err
	}
	tarball.gzipWriter = gzip.NewWriter(tarball.tarFile)
	tarball.tarWriter = tar.NewWriter(tarball.gzipWriter)
	return &tarball, nil
}

func (t *tarball) close() {
	if t.tarWriter != nil {
		t.tarWriter.Close()
	}
	if t.gzipWriter != nil {
		t.gzipWriter.Close()
	}
	if t.tarFile != nil {
		t.tarFile.Close()
	}
}

func (t *tarball) addFile(fileName string, file io.Reader) error {
	buffer := &bytes.Buffer{}
	io.Copy(buffer, file)

	header := &tar.Header{
		Name: fileName,
		Size: int64(buffer.Len()),
		Mode: int64(fs.ModePerm),
	}

	err := t.tarWriter.WriteHeader(header)
	if err != nil {
		return fmt.Errorf("could not write header for file '%s', got error '%w'", fileName, err)
	}

	_, err = io.Copy(t.tarWriter, buffer)
	if err != nil {
		return fmt.Errorf("could not copy the file '%s' data to the tarball, got error '%w'", fileName, err)
	}
	return nil
}

func selectAndCreateTargetFile(ctx *supportArchiveContext) (*os.File, error) {
	if ctx.toStdout {
		return os.Stdout, nil
	} else {
		tarFile, err := createTarFile(ctx)
		if err != nil {
			return nil, err
		}
		return tarFile, nil
	}
}

func createTarFile(ctx *supportArchiveContext) (*os.File, error) {
	tarballFilePath := fmt.Sprintf(tarFileName, ctx.targetDir, time.Now().Format(time.RFC3339))
	tarballFilePath = strings.Replace(tarballFilePath, ":", "_", -1)

	tarFile, err := os.Create(tarballFilePath)
	if err != nil {
		return nil, err
	}
	return tarFile, nil
}
