package csiprovisioner

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/dtclient"
	"github.com/go-logr/logr"
	"github.com/spf13/afero"
)

const agentConfPath = "agent/conf/"

type installAgentConfig struct {
	logger    logr.Logger
	dtc       dtclient.Client
	flavor    string
	arch      string
	targetDir string
	fs        afero.Fs
}

func newInstallAgentConfig(logger logr.Logger, dtc dtclient.Client,
	flavor, arch, targetDir string) *installAgentConfig {
	return &installAgentConfig{
		logger:    logger,
		dtc:       dtc,
		flavor:    flavor,
		arch:      arch,
		targetDir: targetDir,
		fs:        afero.NewOsFs(),
	}
}

func installAgent(installAgentCfg *installAgentConfig) error {
	logger := installAgentCfg.logger
	dtc := installAgentCfg.dtc
	flavor := installAgentCfg.flavor
	arch := installAgentCfg.arch
	targetDir := installAgentCfg.targetDir
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

	logger.Info("Downloading OneAgent package", "flavor", flavor, "architecture", arch)

	r, err := dtc.GetLatestAgent(dtclient.OsUnix, dtclient.InstallerTypePaaS, flavor, arch)
	if err != nil {
		return fmt.Errorf("failed to fetch latest OneAgent version: %w", err)
	}
	defer r.Close()

	logger.Info("Saving OneAgent package", "dest", tmpFile.Name())

	size, err := io.Copy(tmpFile, r)
	if err != nil {
		return fmt.Errorf("failed to save OneAgent package: %w", err)
	}

	if _, err := tmpFile.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("failed to save OneAgent package: %w", err)
	}

	zipr, err := zip.NewReader(tmpFile, size)
	if err != nil {
		return fmt.Errorf("failed to open ZIP file: %w", err)
	}

	logger.Info("Unzipping OneAgent package")
	if err := unzip(zipr, installAgentCfg); err != nil {
		return fmt.Errorf("failed to unzip file: %w", err)
	}

	logger.Info("Unzipped OneAgent package")

	for _, dir := range []string{
		filepath.Join(targetDir, "log"),
		filepath.Join(targetDir, "datastorage"),
	} {
		if err := fs.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}

func unzip(r *zip.Reader, installAgentCfg *installAgentConfig) error {
	outDir := installAgentCfg.targetDir
	logger := installAgentCfg.logger
	fs := installAgentCfg.fs

	_ = fs.MkdirAll(outDir, 0755)

	// Closure to address file descriptors issue with all the deferred .Close() methods
	extract := func(zipf *zip.File) error {
		rc, err := zipf.Open()
		if err != nil {
			return err
		}

		defer func() {
			if err := rc.Close(); err != nil {
				logger.Error(err, "Failed to close ZIP entry file", "path", zipf.Name)
			}
		}()

		path := filepath.Join(outDir, zipf.Name)

		// Check for ZipSlip (Directory traversal)
		if !strings.HasPrefix(path, filepath.Clean(outDir)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", path)
		}

		mode := zipf.Mode()

		// Mark all files inside ./agent/conf as group-writable
		if zipf.Name != agentConfPath && strings.HasPrefix(zipf.Name, agentConfPath) {
			mode |= 020
		}

		if zipf.FileInfo().IsDir() {
			return fs.MkdirAll(path, mode)
		}

		if err = fs.MkdirAll(filepath.Dir(path), mode); err != nil {
			return err
		}

		f, err := fs.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
		if err != nil {
			return err
		}

		defer func() {
			if err := f.Close(); err != nil {
				logger.Error(err, "Failed to close target file", "path", f.Name)
			}
		}()

		_, err = io.Copy(f, rc)
		return err
	}

	for _, f := range r.File {
		if err := extract(f); err != nil {
			return err
		}
	}

	return nil
}
