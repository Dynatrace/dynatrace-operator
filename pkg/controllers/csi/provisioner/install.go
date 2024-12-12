package csiprovisioner

import (
	"context"
	"encoding/base64"
	"errors"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/arch"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/processmoduleconfigsecret"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer/image"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer/symlink"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer/url"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/processmoduleconfig"
)

func (provisioner *OneAgentProvisioner) installAgent(ctx context.Context, dk dynakube.DynaKube) error {
	agentInstaller, err := provisioner.getInstaller(ctx, dk)
	if err != nil {
		log.Info("failed to create CodeModule installer", "dk", dk.GetName())

		return err
	}

	targetDir := provisioner.getTargetDir(dk)

	_, err = agentInstaller.InstallAgent(ctx, targetDir)
	if err != nil {
		return err
	}

	err = provisioner.createLatestVersionSymlink(dk, targetDir)
	if err != nil {
		return err
	}

	return provisioner.setupAgentConfigDir(ctx, dk, targetDir)
}

func (provisioner *OneAgentProvisioner) getInstaller(ctx context.Context, dk dynakube.DynaKube) (installer.Installer, error) {
	switch {
	case dk.CodeModulesVersion() != "":
		dtc, err := buildDtc(provisioner, ctx, dk)
		if err != nil {
			return nil, err
		}

		props := &url.Properties{
			Os:            dtclient.OsUnix,
			Type:          dtclient.InstallerTypePaaS,
			Arch:          arch.Arch,
			Flavor:        arch.Flavor,
			Technologies:  []string{"all"},
			TargetVersion: dk.CodeModulesVersion(),
			SkipMetadata:  true,
			PathResolver:  provisioner.path,
		}

		urlInstaller := provisioner.urlInstallerBuilder(provisioner.fs, dtc, props)

		return urlInstaller, nil
	case dk.CodeModulesImage() != "":
		props := &image.Properties{
			ImageUri:     dk.CodeModulesImage(),
			ApiReader:    provisioner.apiReader,
			Dynakube:     &dk,
			PathResolver: provisioner.path,
		}

		imageInstaller, err := provisioner.imageInstallerBuilder(ctx, provisioner.fs, props)
		if err != nil {
			return nil, err
		}

		return imageInstaller, nil
	default:
		return nil, errors.New("missing version/image information to download CodeModule")
	}
}

func (provisioner *OneAgentProvisioner) getTargetDir(dk dynakube.DynaKube) string {
	var dirName string

	switch {
	case dk.CodeModulesImage() != "":
		// An image URI often contains one or several slashes, which is problematic when trying to use it as a folder name.
		// Easiest to just base64 encode it
		dirName = base64.StdEncoding.EncodeToString([]byte(dk.CodeModulesImage()))
	case dk.CodeModulesVersion() != "":
		dirName = dk.CodeModulesVersion()
	default:
		dirName = "unknown"
	}

	return provisioner.path.AgentSharedBinaryDirForAgent(dirName)
}

func (provisioner *OneAgentProvisioner) createLatestVersionSymlink(dk dynakube.DynaKube, targetDir string) error {
	symlinkPath := provisioner.path.LatestAgentBinaryForDynaKube(dk.GetName())
	if err := symlink.Remove(provisioner.fs, symlinkPath); err != nil {
		return err
	}

	err := symlink.Create(provisioner.fs, targetDir, symlinkPath)
	if err != nil {
		return err
	}

	return err
}

func (provisioner *OneAgentProvisioner) setupAgentConfigDir(ctx context.Context, dk dynakube.DynaKube, targetDir string) error {
	latestProcessModuleConfig, err := processmoduleconfigsecret.GetSecretData(ctx, provisioner.apiReader, dk.Name, dk.Namespace)
	if err != nil {
		return err
	}

	return processmoduleconfig.UpdateFromDir(provisioner.fs, provisioner.path.AgentConfigDir(dk.GetName()), targetDir, latestProcessModuleConfig)
}
