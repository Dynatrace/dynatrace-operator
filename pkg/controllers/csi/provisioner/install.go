package csiprovisioner

import (
	"context"
	"encoding/base64"
	"errors"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/arch"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer/image"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer/job"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer/job/helmconfig"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer/symlink"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer/url"
)

const (
	notReadyRequeueDuration = 30 * time.Second
)

var errNotReady = errors.New("download job is not ready yet")

func (provisioner *OneAgentProvisioner) installAgent(ctx context.Context, dk dynakube.DynaKube) error {
	agentInstaller, err := provisioner.getInstaller(ctx, dk)
	if err != nil {
		log.Info("failed to create CodeModule installer", "dk", dk.GetName())

		return err
	}

	targetDir := provisioner.getTargetDir(dk)

	ready, err := agentInstaller.InstallAgent(ctx, targetDir)
	if err != nil {
		return err
	}

	if !ready {
		return errNotReady
	}

	err = provisioner.createLatestVersionSymlink(dk, targetDir)
	if err != nil {
		return err
	}

	return nil
}

func (provisioner *OneAgentProvisioner) getInstaller(ctx context.Context, dk dynakube.DynaKube) (installer.Installer, error) {
	switch {
	case dk.FF().IsNodeImagePull():
		return provisioner.getJobInstaller(ctx, dk), nil
	case dk.OneAgent().GetCustomCodeModulesImage() != "":
		props := &image.Properties{
			ImageURI:     dk.OneAgent().GetCodeModulesImage(),
			APIReader:    provisioner.apiReader,
			Dynakube:     &dk,
			PathResolver: provisioner.path,
		}

		imageInstaller, err := provisioner.imageInstallerBuilder(ctx, props)
		if err != nil {
			return nil, err
		}

		return imageInstaller, nil
	default:
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
			TargetVersion: dk.OneAgent().GetCodeModulesVersion(),
			SkipMetadata:  true,
			PathResolver:  provisioner.path,
		}

		urlInstaller := provisioner.urlInstallerBuilder(dtc, props)

		return urlInstaller, nil
	}
}

func (provisioner *OneAgentProvisioner) getJobInstaller(ctx context.Context, dk dynakube.DynaKube) installer.Installer {
	imageURI := dk.OneAgent().GetCustomCodeModulesImage()
	if imageURI == "" {
		imageURI = "public.ecr.aws/dynatrace/dynatrace-codemodules:" + dk.OneAgent().GetCodeModulesVersion()
	}

	pullSecrets := []string{}
	if dk.Spec.CustomPullSecret != "" {
		pullSecrets = append(pullSecrets, dk.Spec.CustomPullSecret)
	}

	props := &job.Properties{
		ImageURI:     imageURI,
		Owner:        &dk,
		PullSecrets:  pullSecrets,
		APIReader:    provisioner.apiReader,
		Client:       provisioner.kubeClient,
		PathResolver: provisioner.path,
		CSIJob:       helmconfig.Get(),
	}

	return provisioner.jobInstallerBuilder(ctx, props)
}

func (provisioner *OneAgentProvisioner) getTargetDir(dk dynakube.DynaKube) string {
	var dirName string

	if dk.OneAgent().GetCustomCodeModulesImage() != "" {
		// An image URI often contains one or several slashes, which is problematic when trying to use it as a folder name.
		// Easiest to just base64 encode it
		dirName = base64.StdEncoding.EncodeToString([]byte(dk.OneAgent().GetCodeModulesImage()))
	} else {
		dirName = dk.OneAgent().GetCodeModulesVersion()
	}

	return provisioner.path.AgentSharedBinaryDirForAgent(dirName)
}

func (provisioner *OneAgentProvisioner) createLatestVersionSymlink(dk dynakube.DynaKube, targetDir string) error {
	symlinkPath := provisioner.path.LatestAgentBinaryForDynaKube(dk.GetName())
	if err := symlink.Remove(symlinkPath); err != nil {
		return err
	}

	err := symlink.Create(targetDir, symlinkPath)
	if err != nil {
		return err
	}

	return err
}
