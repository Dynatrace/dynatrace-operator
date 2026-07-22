// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package zip

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer/common"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/klauspost/compress/zip"
	"github.com/pkg/errors"
	"golang.org/x/sys/unix"
)

// hardenedOpenFileFlags is used whenever a regular file is created during code
// module extraction.
//
// Attack vector: the CSI driver extracts code module images as root onto a
// shared hostPath (the CSI data dir) that is reused across DynaKubes. A
// malicious image layer can contain symlink entries that, once created, form a
// chain redirecting a subsequent regular-file write out of the extraction
// directory - for example onto another DynaKube's liboneagentproc.so, which is
// later loaded via LD_PRELOAD into injected workloads, yielding code execution
// as that workload.
//
// unix.O_NOFOLLOW makes os.OpenFile refuse (ELOOP) to follow a symlink at the
// final path component, so a pre-existing or concurrently-planted symlink at the
// destination cannot redirect the written bytes. It is applied unconditionally -
// on top of skipping symlink/hardlink archive entries by default - so the
// protection still holds when link extraction is explicitly opted in.
//
// Note: O_NOFOLLOW only guards the final path component, not intermediate
// directory components. Fully confining resolution would require openat2 with
// RESOLVE_NO_SYMLINKS (or filepath-securejoin).
const hardenedOpenFileFlags = os.O_CREATE | os.O_WRONLY | os.O_TRUNC | unix.O_NOFOLLOW

func (extractor OneAgentExtractor) ExtractZip(ctx context.Context, sourceFile *os.File, targetDir string) error {
	ctx, log := logd.NewFromContext(ctx, "zip")

	extractor.cleanTempZipDir()

	if sourceFile == nil {
		return errors.New("file is nil")
	}

	fileInfo, err := sourceFile.Stat()
	if err != nil {
		log.Info("failed to get file info", "err", err)

		return errors.WithStack(err)
	}

	reader, err := zip.NewReader(sourceFile, fileInfo.Size())
	if err != nil {
		log.Info("failed to create zip reader", "err", err)

		return errors.WithStack(err)
	}

	extractDest := extractor.pathResolver.AgentTempUnzipRootDir()
	if extractor.pathResolver.RootDir == consts.AgentInitBinDirMount {
		extractDest = targetDir
	}

	err = extractFilesFromZip(extractDest, reader)
	if err != nil {
		log.Info("failed to extract files from zip", "err", err)

		return err
	}

	if extractDest != targetDir {
		err := extractor.moveToTargetDir(ctx, targetDir)
		if err != nil {
			log.Info("failed to move file to final destination", "err", err)

			return err
		}
	}

	return nil
}

func extractFilesFromZip(targetDir string, reader *zip.Reader) error {
	if err := os.MkdirAll(targetDir, common.MkDirFileMode); err != nil {
		return errors.WithStack(err)
	}

	for _, file := range reader.File {
		if err := extractFileFromZip(targetDir, file); err != nil {
			return err
		}
	}

	return nil
}

func extractFileFromZip(targetDir string, file *zip.File) (reterr error) {
	path := filepath.Join(targetDir, file.Name)

	// Check for ZipSlip: https://snyk.io/research/zip-slip-vulnerability
	if !strings.HasPrefix(path, filepath.Clean(targetDir)+string(os.PathSeparator)) {
		return errors.Errorf("illegal file path: %s", path)
	}

	mode := file.Mode()

	if file.FileInfo().IsDir() {
		err := os.MkdirAll(path, mode)
		if err != nil {
			return errors.WithStack(err)
		}

		return nil
	}

	if isAgentConfFile(file.Name) {
		mode = common.ReadWriteAllFileMode
	}

	if err := os.MkdirAll(filepath.Dir(path), mode); err != nil {
		return errors.WithStack(err)
	}

	dstFile, err := os.OpenFile(path, hardenedOpenFileFlags, mode)
	if err != nil {
		return errors.WithStack(err)
	}

	defer func() {
		// failed close has lower prio than existing error
		if closeErr := dstFile.Close(); reterr == nil {
			reterr = closeErr
		}
	}()

	srcFile, err := file.Open()
	if err != nil {
		return errors.WithStack(err)
	}

	defer func() {
		// failed close has lower prio than existing error
		if closeErr := srcFile.Close(); reterr == nil {
			reterr = closeErr
		}
	}()

	_, err = io.Copy(dstFile, srcFile)

	return errors.WithStack(err)
}
