package startup

import (
	"context"
	"fmt"
	"path"
	"path/filepath"

	"github.com/Dynatrace/dynatrace-operator/pkg/arch"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/metadata"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer/url"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/processmoduleconfig"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
)

type Runner struct {
	fs         afero.Fs
	env        *environment
	config     *SecretConfig
	dtclient   dtclient.Client
	installer  installer.Installer
	hostTenant string
}

func NewRunner(fs afero.Fs) (*Runner, error) {
	log.Info("creating standalone runner")

	env, err := newEnv()
	if err != nil {
		return nil, err
	}

	var secretConfig *SecretConfig

	var client dtclient.Client

	var oneAgentInstaller installer.Installer

	if env.OneAgentInjected {
		secretConfig, err = newSecretConfigViaFs(fs)
		if err != nil {
			return nil, err
		}

		trustedCAs, err := newSecretTrustedCAsViaFs(fs)
		if err != nil {
			return nil, err
		}

		client, err = newDTClientBuilder(secretConfig, trustedCAs).createClient()
		if err != nil {
			return nil, err
		}

		targetVersion := url.VersionLatest
		if env.InstallVersion != "" {
			targetVersion = env.InstallVersion
		}

		oneAgentInstaller = url.NewUrlInstaller(
			fs,
			client,
			&url.Properties{
				Os:            dtclient.OsUnix,
				Type:          dtclient.InstallerTypePaaS,
				Flavor:        env.InstallerFlavor,
				Arch:          arch.Arch,
				Technologies:  env.InstallerTech,
				TargetVersion: targetVersion,
				Url:           env.InstallerUrl,
				SkipMetadata:  false,
				PathResolver:  metadata.PathResolver{RootDir: consts.AgentBinDirMount},
			},
		)
	}

	log.Info("standalone runner created successfully")

	return &Runner{
		fs:        fs,
		env:       env,
		config:    secretConfig,
		dtclient:  client,
		installer: oneAgentInstaller,
	}, nil
}

func (runner *Runner) Run(ctx context.Context) (resultedError error) {
	log.Info("standalone agent init started")

	defer runner.consumeErrorIfNecessary(&resultedError)

	if runner.env.OneAgentInjected {
		if err := runner.setHostTenant(); err != nil {
			return err
		}

		if !runner.config.CSIMode {
			if err := runner.installOneAgent(ctx); err != nil {
				return err
			}

			log.Info("OneAgent download finished")
		}
	}

	err := runner.configureInstallation()
	if err == nil {
		log.Info("standalone agent init completed")
	}

	return err
}

func (runner *Runner) consumeErrorIfNecessary(resultedError *error) {
	if runner.env.FailurePolicy == silentPhrase && *resultedError != nil {
		log.Error(*resultedError, "This error has been masked to not fail the container.")
		*resultedError = nil
	}
}

func (runner *Runner) setHostTenant() error {
	log.Info("setting host tenant")

	runner.hostTenant = consts.AgentNoHostTenant
	if runner.config.HasHost {
		if runner.config.EnforcementMode {
			runner.hostTenant = runner.config.TenantUUID

			log.Info("host tenant set to TenantUUID")
		} else {
			hostTenant, ok := runner.config.MonitoringNodes[runner.env.K8NodeName]
			if !ok {
				return errors.Errorf("host tenant info is missing for %s", runner.env.K8NodeName)
			}

			runner.hostTenant = hostTenant
		}
	}

	log.Info("successfully set host tenant", "hostTenant", runner.hostTenant)

	return nil
}

func (runner *Runner) installOneAgent(ctx context.Context) error {
	log.Info("downloading OneAgent")

	_, err := runner.installer.InstallAgent(ctx, consts.AgentBinDirMount)
	if err != nil {
		return err
	}

	processModuleConfig, err := runner.getProcessModuleConfig(ctx)
	if err != nil {
		return err
	}

	err = processmoduleconfig.UpdateProcessModuleConfigInPlace(runner.fs, consts.AgentBinDirMount, processModuleConfig)
	if err != nil {
		return err
	}

	return nil
}

func (runner *Runner) getProcessModuleConfig(ctx context.Context) (*dtclient.ProcessModuleConfig, error) {
	processModuleConfig, err := runner.dtclient.GetProcessModuleConfig(ctx, 0)
	if err != nil {
		return nil, err
	}

	if runner.config.Proxy != "" {
		processModuleConfig = processModuleConfig.AddProxy(runner.config.Proxy)
	}

	if runner.config.OneAgentNoProxy != "" {
		processModuleConfig = processModuleConfig.AddNoProxy(runner.config.OneAgentNoProxy)
	}

	if runner.config.HostGroup != "" {
		processModuleConfig.AddHostGroup(runner.config.HostGroup)
	}

	return processModuleConfig, nil
}

func (runner *Runner) configureInstallation() error {
	log.Info("configuring standalone OneAgent")

	if runner.env.OneAgentInjected {
		if err := runner.configureOneAgent(); err != nil {
			return err
		}
	}

	if runner.env.DataIngestInjected {
		log.Info("creating enrichment files")

		if err := runner.enrichMetadata(); err != nil {
			return err
		}
	}

	return nil
}

func (runner *Runner) configureOneAgent() error {
	log.Info("setting ld.so.preload")

	if err := runner.setLDPreload(); err != nil {
		return err
	}

	log.Info("creating container configuration files")

	if err := runner.createContainerConfigurationFiles(); err != nil {
		return err
	}

	if runner.config.InitialConnectRetry > -1 {
		log.Info("creating curl options file")

		if err := runner.createCurlOptionsFile(); err != nil {
			return err
		}
	}

	if runner.config.ReadOnlyCSIDriver {
		log.Info("readOnly CSI detected, copying agent conf to empty-dir")

		err := copyFolder(runner.fs, getReadOnlyAgentConfMountPath(), consts.AgentConfInitDirMount)
		if err != nil {
			return err
		}
	}

	return nil
}

func (runner *Runner) setLDPreload() error {
	return runner.createConfFile(filepath.Join(consts.AgentShareDirMount, consts.LdPreloadFilename), filepath.Join(runner.env.InstallPath, consts.LibAgentProcPath))
}

func (runner *Runner) createContainerConfigurationFiles() error {
	for _, container := range runner.env.Containers {
		log.Info("creating conf file for container", "container", container)
		confFilePath := filepath.Join(consts.AgentShareDirMount, fmt.Sprintf(consts.AgentContainerConfFilenameTemplate, container.Name))
		content := runner.getBaseConfContent(container)

		log.Info("adding k8s cluster id")

		content += runner.getK8SClusterID()

		if runner.hostTenant != consts.AgentNoHostTenant {
			if runner.config.TenantUUID == runner.hostTenant {
				log.Info("adding k8s node name")

				content += runner.getK8SHostInfo()
			}
		}

		if err := runner.createConfFile(confFilePath, content); err != nil {
			return err
		}
	}

	return nil
}

func (runner *Runner) enrichMetadata() error {
	if err := runner.createPropsEnrichmentFile(); err != nil {
		return err
	}

	if err := runner.createJsonEnrichmentFile(); err != nil {
		return err
	}

	return nil
}

func getReadOnlyAgentConfMountPath() string {
	return path.Join(consts.AgentBinDirMount, "agent/conf")
}
