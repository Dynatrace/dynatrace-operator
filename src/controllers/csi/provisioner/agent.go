package csiprovisioner

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/csi/metadata"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/csi/provisioner/arch"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/klauspost/compress/zip"
	"github.com/spf13/afero"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
)

const agentConfPath = "agent/conf/"

type installAgentConfig struct {
	dtc      dtclient.Client
	fs       afero.Fs
	path     metadata.PathResolver
	dk       *dynatracev1beta1.DynaKube
	recorder record.EventRecorder
}

func newInstallAgentConfig(
	dtc dtclient.Client,
	path metadata.PathResolver,
	fs afero.Fs,
	recorder record.EventRecorder,
	dk *dynatracev1beta1.DynaKube,
) *installAgentConfig {
	return &installAgentConfig{
		dtc:      dtc,
		path:     path,
		fs:       fs,
		recorder: recorder,
		dk:       dk,
	}
}

func (installAgentCfg *installAgentConfig) updateAgent(installedVersion, tenantUUID string, previousHash string, latestProcessModuleConfigCache *processModuleConfigCache) (string, error) {
	dk := installAgentCfg.dk
	targetVersion := installAgentCfg.getOneAgentVersionFromInstance()
	targetDir := installAgentCfg.path.AgentBinaryDirForVersion(tenantUUID, targetVersion)

	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		log.Info("updating agent",
			"target version", targetVersion,
			"installed version", installedVersion,
			"target directory", targetDir)

		if err := installAgentCfg.installAgentVersion(targetVersion, tenantUUID); err != nil {
			installAgentCfg.recorder.Eventf(dk,
				corev1.EventTypeWarning,
				failedInstallAgentVersionEvent,
				"Failed to install agent version: %s to tenant: %s, err: %s", targetVersion, tenantUUID, err)
			return "", err
		}
		log.Info("updating ruxitagentproc.conf on new version")
		if err := installAgentCfg.updateProcessModuleConfig(targetVersion, tenantUUID, latestProcessModuleConfigCache.ProcessModuleConfig); err != nil {
			return "", err
		}
		installAgentCfg.recorder.Eventf(dk,
			corev1.EventTypeNormal,
			installAgentVersionEvent,
			"Installed agent version: %s to tenant: %s", targetVersion, tenantUUID)
		return targetVersion, nil
	}
	if targetVersion != installedVersion {
		log.Info("updating agent, installer was already present",
			"target version", targetVersion,
			"installed version", installedVersion,
			"target directory", targetDir)
		installAgentCfg.recorder.Eventf(dk,
			corev1.EventTypeNormal,
			installAgentVersionEvent,
			"Set new agent version: %s to tenant: %s", targetVersion, tenantUUID)
		log.Info("updating ruxitagentproc.conf on new set version")
		if err := installAgentCfg.updateProcessModuleConfig(targetVersion, tenantUUID, latestProcessModuleConfigCache.ProcessModuleConfig); err != nil {
			return "", err
		}
		return targetVersion, nil
	}
	if latestProcessModuleConfigCache != nil && previousHash != latestProcessModuleConfigCache.Hash {
		log.Info("updating ruxitagentproc.conf on latest installed version")
		if err := installAgentCfg.updateProcessModuleConfig(targetVersion, tenantUUID, latestProcessModuleConfigCache.ProcessModuleConfig); err != nil {
			return "", err
		}
	}

	return "", nil
}

func (installAgentCfg *installAgentConfig) getOneAgentVersionFromInstance() string {
	dk := installAgentCfg.dk
	currentVersion := dk.Status.LatestAgentVersionUnixPaas
	if dk.Version() != "" {
		currentVersion = dk.Version()
	}
	return currentVersion
}

func (installAgentCfg *installAgentConfig) installAgentVersion(version, tenantUUID string) error {
	targetDir := installAgentCfg.path.AgentBinaryDirForVersion(tenantUUID, version)

	log.Info("installing agent", "target dir", targetDir)
	if err := installAgentCfg.installAgent(version, tenantUUID); err != nil {
		_ = installAgentCfg.fs.RemoveAll(targetDir)

		return fmt.Errorf("failed to install agent: %w", err)
	}

	return nil
}

func (installAgentCfg *installAgentConfig) installAgent(version, tenantUUID string) error {
	dtc := installAgentCfg.dtc
	fs := installAgentCfg.fs

	tmpFile, err := afero.TempFile(fs, "", "download")
	if err != nil {
		return fmt.Errorf("failed to create temporary file for download: %w", err)
	}
	defer func() {
		_ = tmpFile.Close()
		if err := fs.Remove(tmpFile.Name()); err != nil {
			log.Error(err, "failed to delete downloaded file", "path", tmpFile.Name())
		}
	}()

	log.Info("downloading OneAgent package", "architecture", arch.Arch, "flavor", arch.Flavor)
	err = dtc.GetAgent(dtclient.OsUnix, dtclient.InstallerTypePaaS, arch.Flavor, arch.Arch, version, tmpFile)

	if err != nil {
		availableVersions, getVersionsError := dtc.GetAgentVersions(dtclient.OsUnix, dtclient.InstallerTypePaaS, arch.Flavor, arch.Arch)
		if getVersionsError != nil {
			return fmt.Errorf("failed to fetch OneAgent version: %w", err)
		}
		return fmt.Errorf("failed to fetch OneAgent version: %w, available versions are: %s", err, "[ "+strings.Join(availableVersions, " , ")+" ]")
	}

	var fileSize int64
	if stat, err := tmpFile.Stat(); err == nil {
		fileSize = stat.Size()
	}

	log.Info("saved OneAgent package", "dest", tmpFile.Name(), "size", fileSize)
	log.Info("unzipping OneAgent package")
	if err := installAgentCfg.unzip(tmpFile, version, tenantUUID); err != nil {
		return fmt.Errorf("failed to unzip file: %w", err)
	}
	log.Info("unzipped OneAgent package")

	if err = installAgentCfg.createSymlinkIfNotExists(version, tenantUUID); err != nil {
		return err
	}

	return nil
}

func (installAgentCfg *installAgentConfig) unzip(file afero.File, version, tenantUUID string) error {
	target := installAgentCfg.path.AgentBinaryDirForVersion(tenantUUID, version)
	fs := installAgentCfg.fs

	if file == nil {
		return fmt.Errorf("file is nil")
	}

	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("unable to determine file info: %w", err)
	}

	reader, err := zip.NewReader(file, fileInfo.Size())
	if err != nil {
		return fmt.Errorf("failed to open ZIP file: %w", err)
	}

	_ = fs.MkdirAll(target, 0755)

	for _, file := range reader.File {
		err := func() error {
			path := filepath.Join(target, file.Name)

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

func (installAgentCfg *installAgentConfig) createSymlinkIfNotExists(version, tenantUUID string) error {
	fs := installAgentCfg.fs

	// MemMapFs (used for testing) doesn't comply with the Linker interface
	linker, ok := fs.(afero.Linker)
	if !ok {
		log.Info("symlinking not possible", "version", version, "fs", installAgentCfg.fs)
		return nil
	}

	relativeSymlinkPath := version
	symlinkTargetPath := installAgentCfg.path.InnerAgentBinaryDirForSymlinkForVersion(tenantUUID, version)
	if fileInfo, _ := fs.Stat(symlinkTargetPath); fileInfo != nil {
		log.Info("symlink already exists", "location", symlinkTargetPath)
		return nil
	}

	log.Info("creating symlink", "points-to(relative)", relativeSymlinkPath, "location", symlinkTargetPath)
	if err := linker.SymlinkIfPossible(relativeSymlinkPath, symlinkTargetPath); err != nil {
		log.Info("symlinking failed", "version", version)
		return err
	}
	return nil
}
