package installer

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/processmoduleconfig"
	"github.com/klauspost/compress/zip"
	"github.com/spf13/afero"
)

const (
	agentConfPath = "agent/conf/"
	VersionLatest = "latest"
)

var (
	ruxitAgentProcPath       = filepath.Join("agent", "conf", "ruxitagentproc.conf")
	sourceRuxitAgentProcPath = filepath.Join("agent", "conf", "_ruxitagentproc.conf")
)

type InstallerProperties struct {
	Os           string
	Arch         string
	Type         string
	Flavor       string
	Version      string
	Technologies []string
	Url          string // if this is set all others will be ignored
}

func (props *InstallerProperties) fillEmptyWithDefaults() {
	if props.Technologies == nil || len(props.Technologies) == 0 {
		props.Technologies = []string{"all"}
	}
}

type Installer interface {
	InstallAgent(targetDir string) error
	UpdateProcessModuleConfig(targetDir string, processModuleConfig *dtclient.ProcessModuleConfig) error
	SetVersion(version string)
}

var _ Installer = &OneAgentInstaller{}

type OneAgentInstaller struct {
	fs    afero.Fs
	dtc   dtclient.Client
	props InstallerProperties
}

func NewOneAgentInstaller(
	fs afero.Fs,
	dtc dtclient.Client,
	props InstallerProperties,
) *OneAgentInstaller {
	return &OneAgentInstaller{
		fs:    fs,
		dtc:   dtc,
		props: props,
	}
}

func (installer *OneAgentInstaller) InstallAgent(targetDir string) error {
	log.Info("installing agent", "target dir", targetDir)
	installer.props.fillEmptyWithDefaults()
	if err := installer.installAgent(targetDir); err != nil {
		_ = installer.fs.RemoveAll(targetDir)

		return fmt.Errorf("failed to install agent: %w", err)
	}

	return nil
}

func (installer *OneAgentInstaller) SetVersion(version string) {
	installer.props.Version = version
}

func (installer *OneAgentInstaller) installAgent(targetDir string) error {
	fs := installer.fs
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

	if installer.props.Url != "" {
		if err := installer.downloadOneAgentViaInstallerUrl(tmpFile); err != nil {
			return err
		}
	} else if installer.props.Version == VersionLatest {
		if err := installer.downloadLatestOneAgent(tmpFile); err != nil {
			return err
		}
	} else {
		if err := installer.downloadOneAgentWithVersion(tmpFile); err != nil {
			return err
		}
	}

	var fileSize int64
	if stat, err := tmpFile.Stat(); err == nil {
		fileSize = stat.Size()
	}

	log.Info("saved OneAgent package", "dest", tmpFile.Name(), "size", fileSize)
	log.Info("unzipping OneAgent package")
	if err := installer.unzip(tmpFile, targetDir); err != nil {
		return fmt.Errorf("failed to unzip file: %w", err)
	}
	log.Info("unzipped OneAgent package")

	if err = installer.createSymlinkIfNotExists(targetDir); err != nil {
		return err
	}

	return nil
}

func (installer *OneAgentInstaller) downloadLatestOneAgent(tmpFile afero.File) error {
	log.Info("downloading latest OneAgent package", "props", installer.props)
	return installer.dtc.GetLatestAgent(
		installer.props.Os,
		installer.props.Type,
		installer.props.Flavor,
		installer.props.Arch,
		installer.props.Technologies,
		tmpFile,
	)
}

func (installer *OneAgentInstaller) downloadOneAgentWithVersion(tmpFile afero.File) error {
	log.Info("downloading specific OneAgent package", "version", installer.props.Version)
	err := installer.dtc.GetAgent(
		installer.props.Os,
		installer.props.Type,
		installer.props.Flavor,
		installer.props.Arch,
		installer.props.Version,
		installer.props.Technologies,
		tmpFile,
	)

	if err != nil {
		availableVersions, getVersionsError := installer.dtc.GetAgentVersions(
			installer.props.Os,
			installer.props.Type,
			installer.props.Flavor,
			installer.props.Arch,
		)
		if getVersionsError != nil {
			return fmt.Errorf("failed to fetch OneAgent version: %w", err)
		}
		return fmt.Errorf("failed to fetch OneAgent version: %w, available versions are: %s", err, "[ "+strings.Join(availableVersions, " , ")+" ]")
	}
	return nil
}

func (installer *OneAgentInstaller) downloadOneAgentViaInstallerUrl(tmpFile afero.File) error {
	log.Info("downloading OneAgent package using provided url", "url", installer.props.Url)
	return installer.dtc.GetAgentViaInstallerUrl(installer.props.Url, tmpFile)
}

func (installer *OneAgentInstaller) unzip(file afero.File, targetDir string) error {
	fs := installer.fs

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

	_ = fs.MkdirAll(targetDir, 0755)

	for _, file := range reader.File {
		err := func() error {
			path := filepath.Join(targetDir, file.Name)

			// Check for ZipSlip: https://snyk.io/research/zip-slip-vulnerability
			if !strings.HasPrefix(path, filepath.Clean(targetDir)+string(os.PathSeparator)) {
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

func (installer *OneAgentInstaller) createSymlinkIfNotExists(targetDir string) error {
	fs := installer.fs

	// MemMapFs (used for testing) doesn't comply with the Linker interface
	linker, ok := fs.(afero.Linker)
	if !ok {
		log.Info("symlinking not possible", "version", installer.props.Version, "fs", installer.fs)
		return nil
	}

	relativeSymlinkPath := installer.props.Version
	symlinkTargetPath := filepath.Join(targetDir, "agent", "bin", "current")
	if fileInfo, _ := fs.Stat(symlinkTargetPath); fileInfo != nil {
		log.Info("symlink already exists", "location", symlinkTargetPath)
		return nil
	}

	log.Info("creating symlink", "points-to(relative)", relativeSymlinkPath, "location", symlinkTargetPath)
	if err := linker.SymlinkIfPossible(relativeSymlinkPath, symlinkTargetPath); err != nil {
		log.Info("symlinking failed", "version", installer.props.Version)
		return err
	}
	return nil
}

func (installer *OneAgentInstaller) UpdateProcessModuleConfig(targetDir string, processModuleConfig *dtclient.ProcessModuleConfig) error {
	if processModuleConfig != nil {
		log.Info("updating ruxitagentproc.conf", "agentVersion", installer.props.Version)
		usedProcessModuleConfigPath := filepath.Join(targetDir, ruxitAgentProcPath)
		sourceProcessModuleConfigPath := filepath.Join(targetDir, sourceRuxitAgentProcPath)
		if err := installer.checkProcessModuleConfigCopy(sourceProcessModuleConfigPath, usedProcessModuleConfigPath); err != nil {
			return err
		}
		return processmoduleconfig.Update(installer.fs, sourceProcessModuleConfigPath, usedProcessModuleConfigPath, processModuleConfig.ToMap())
	}
	log.Info("no changes to ruxitagentproc.conf, skipping update")
	return nil
}

// checkProcessModuleConfigCopy checks if we already made a copy of the original ruxitagentproc.conf file.
// After the initial install of a version we copy the ruxitagentproc.conf to _ruxitagentproc.conf and we use the _ruxitagentproc.conf + the api response to re-create the ruxitagentproc.conf
// so its easier to update
func (installer *OneAgentInstaller) checkProcessModuleConfigCopy(sourcePath, destPath string) error {
	if _, err := installer.fs.Open(sourcePath); os.IsNotExist(err) {
		log.Info("saving original ruxitagentproc.conf to _ruxitagentproc.conf")
		fileInfo, err := installer.fs.Stat(destPath)
		if err != nil {
			return err
		}

		sourceProcessModuleConfigFile, err := installer.fs.OpenFile(sourcePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, fileInfo.Mode())
		if err != nil {
			return err
		}

		usedProcessModuleConfigFile, err := installer.fs.Open(destPath)
		if err != nil {
			return err
		}
		_, err = io.Copy(sourceProcessModuleConfigFile, usedProcessModuleConfigFile)
		if err != nil {
			sourceProcessModuleConfigFile.Close()
			usedProcessModuleConfigFile.Close()
			return err
		}
		if err = sourceProcessModuleConfigFile.Close(); err != nil {
			return err
		}
		return usedProcessModuleConfigFile.Close()
	}
	return nil
}
