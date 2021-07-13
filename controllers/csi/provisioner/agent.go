package csiprovisioner

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/dtclient"
	"github.com/go-logr/logr"
	"github.com/klauspost/compress/zip"
	"github.com/spf13/afero"
)

const agentConfPath = "agent/conf/"

type installAgentConfig struct {
	logger    logr.Logger
	dtc       dtclient.Client
	arch      string
	targetDir string
	fs        afero.Fs
}

func newInstallAgentConfig(logger logr.Logger, dtc dtclient.Client, arch, targetDir string) *installAgentConfig {
	return &installAgentConfig{
		logger:    logger,
		dtc:       dtc,
		arch:      arch,
		targetDir: targetDir,
		fs:        afero.NewOsFs(),
	}
}

func installAgent(installAgentCfg *installAgentConfig) error {
	logger := installAgentCfg.logger
	dtc := installAgentCfg.dtc
	arch := installAgentCfg.arch
	fs := installAgentCfg.fs

	tmpFile, err := afero.TempFile(fs, "", "download")
	if err != nil {
		return fmt.Errorf("failed to create temporary file for download: %w", err)
	}
	defer func() {
		_ = tmpFile.Close()
		if err := fs.Remove(tmpFile.Name()); err != nil {
			logger.Error(err, "Failed to delete downloaded file", "path", tmpFile.Name())
		}
	}()

	logger.Info("Downloading OneAgent package", "architecture", arch)
	err = dtc.GetLatestAgent(dtclient.OsUnix, dtclient.InstallerTypePaaS, dtclient.FlavorMultidistro, arch, tmpFile)
	if err != nil {
		return fmt.Errorf("failed to fetch latest OneAgent version: %w", err)
	}
	logger.Info("Saved OneAgent package", "dest", tmpFile.Name())

	newFile, err := os.Create("/tmp/copied-paas.zip")
	if err != nil {
		logger.Error(err, "error creating new file")
	}
	defer newFile.Close()

	bytesCopied, err := io.Copy(newFile, tmpFile)
	if err != nil {
		logger.Error(err, "error copying file")
	}
	logger.Info("Copied %d bytes.", bytesCopied)

	fileInfo, err := tmpFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to determine agent archive file size: %w", err)
	}

	logger.Info("determined file info",
		"name", fileInfo.Name(),
		"size", fileInfo.Size(),
		"mode", fileInfo.Mode(),
		"file name", tmpFile.Name())
	reader, err := zip.OpenReader(tmpFile.Name())
	if err != nil {
		return fmt.Errorf("failed to open ZIP file: %w", err)
	}
	defer func() { _ = reader.Close() }()

	logger.Info("Unzipping OneAgent package")
	if err := unzip(reader, installAgentCfg); err != nil {
		return fmt.Errorf("failed to unzip file: %w", err)
	}

	logger.Info("Unzipped OneAgent package")

	return nil
}

func unzip(reader *zip.ReadCloser, installAgentCfg *installAgentConfig) error {
	target := installAgentCfg.targetDir
	logger := installAgentCfg.logger
	fs := installAgentCfg.fs

	_ = fs.MkdirAll(target, 0755)

	for _, file := range reader.File {
		err := func() error {
			path := filepath.Join(target, file.Name)
			logger.Info("reading file", "path", path)

			// Check for ZipSlip: https://snyk.io/research/zip-slip-vulnerability
			if !strings.HasPrefix(path, filepath.Clean(target)+string(os.PathSeparator)) {
				return fmt.Errorf("illegal file path: %s", path)
			}

			mode := file.Mode()

			// Mark all files inside ./agent/conf as group-writable
			if file.Name != agentConfPath && strings.HasPrefix(file.Name, agentConfPath) {
				mode |= 020
			}

			if file.FileInfo().IsDir() {
				return fs.MkdirAll(path, mode)
			}

			if err := fs.MkdirAll(filepath.Dir(path), mode); err != nil {
				return err
			}

			dstFile, err := fs.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
			if err != nil {
				return err
			}
			defer func() { _ = dstFile.Close() }()

			srcFile, err := file.Open()
			if err != nil {
				return err
			}
			defer func() { _ = srcFile.Close() }()

			_, err = io.Copy(dstFile, srcFile)
			return err
		}()
		if err != nil {
			return err
		}
	}

	return nil
}
