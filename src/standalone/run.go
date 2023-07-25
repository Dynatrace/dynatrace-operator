package standalone

import (
	"fmt"
	"path"
	"path/filepath"

	"github.com/Dynatrace/dynatrace-operator/src/arch"
	"github.com/Dynatrace/dynatrace-operator/src/config"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/csi/metadata"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/installer"
	"github.com/Dynatrace/dynatrace-operator/src/installer/url"
	"github.com/Dynatrace/dynatrace-operator/src/processmoduleconfig"
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

		client, err = newDTClientBuilder(secretConfig).createClient()
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
				PathResolver:  metadata.PathResolver{RootDir: config.AgentBinDirMount},
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

func (runner *Runner) Run() (resultedError error) {
	log.Info("standalone agent init started")
	defer runner.consumeErrorIfNecessary(&resultedError)

	if runner.env.OneAgentInjected {
		if err := runner.setHostTenant(); err != nil {
			return err
		}

		if runner.env.Mode == config.AgentInstallerMode {
			if err := runner.installOneAgent(); err != nil {
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
	if !runner.env.FailurePolicy && *resultedError != nil {
		log.Error(*resultedError, "This error has been masked to not fail the container.")
		*resultedError = nil
	}
}

func (runner *Runner) setHostTenant() error {
	log.Info("setting host tenant")
	runner.hostTenant = config.AgentNoHostTenant
	if runner.config.HasHost {
		hostTenant, ok := runner.config.MonitoringNodes[runner.env.K8NodeName]
		if !ok {
			return errors.Errorf("host tenant info is missing for %s", runner.env.K8NodeName)
		}
		runner.hostTenant = hostTenant
	}
	log.Info("successfully set host tenant", "hostTenant", runner.hostTenant)
	return nil
}

func (runner *Runner) installOneAgent() error {
	log.Info("downloading OneAgent")
	_, err := runner.installer.InstallAgent(config.AgentBinDirMount)
	if err != nil {
		return err
	}
	processModuleConfig, err := runner.getProcessModuleConfig()
	if err != nil {
		return err
	}
	err = processmoduleconfig.UpdateProcessModuleConfigInPlace(runner.fs, config.AgentBinDirMount, processModuleConfig)
	if err != nil {
		return err
	}
	return nil
}

func (runner *Runner) getProcessModuleConfig() (*dtclient.ProcessModuleConfig, error) {
	processModuleConfig, err := runner.dtclient.GetProcessModuleConfig(0)
	if err != nil {
		return nil, err
	}

	if runner.config.Proxy != "" {
		processModuleConfig.AddProxy(runner.config.Proxy)
	}
	processModuleConfig = processModuleConfig.AddConnectionInfo(runner.config.ConnectionInfo, runner.config.TenantToken)
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

	if runner.config.TlsCert != "" {
		log.Info("propagating tls cert to agent")
		if err := runner.propagateTLSCert(); err != nil {
			return err
		}
	}

	if runner.config.InitialConnectRetry > -1 {
		log.Info("creating curl options file")
		if err := runner.createCurlOptionsFile(); err != nil {
			return err
		}
	}
	if runner.env.IsReadOnlyCSI {
		log.Info("readOnly CSI detected, copying agent conf to empty-dir")
		err := copyFolder(runner.fs, getReadOnlyAgentConfMountPath(), config.AgentConfInitDirMount)
		if err != nil {
			return err
		}
	}
	return nil
}

func (runner *Runner) setLDPreload() error {
	return runner.createConfFile(filepath.Join(config.AgentShareDirMount, config.LdPreloadFilename), filepath.Join(runner.env.InstallPath, config.LibAgentProcPath))
}

func (runner *Runner) createContainerConfigurationFiles() error {
	for _, container := range runner.env.Containers {
		log.Info("creating conf file for container", "container", container)
		confFilePath := filepath.Join(config.AgentShareDirMount, fmt.Sprintf(config.AgentContainerConfFilenameTemplate, container.Name))
		content := runner.getBaseConfContent(container)
		if runner.hostTenant != config.AgentNoHostTenant {
			if runner.config.TenantUUID == runner.hostTenant {
				log.Info("adding k8s fields")
				content += runner.getK8ConfContent()
			}
			log.Info("adding hostTenant field")
			content += runner.getHostConfContent()
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

func (runner *Runner) propagateTLSCert() error {
	return runner.createConfFile(filepath.Join(config.AgentShareDirMount, "custom.pem"), runner.config.TlsCert)
}

func getReadOnlyAgentConfMountPath() string {
	return path.Join(config.AgentBinDirMount, "agent/conf")
}
