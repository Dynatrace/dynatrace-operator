package standalone

import (
	"fmt"
	"path/filepath"

	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/installer"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
)

const (
	noHostTenant = "-"

	enrichmentFilenameTemplate    = "dt_metadata.%s"
	containerConfFilenameTemplate = "container_%s.conf"
	ldPreloadFilename             = "ld.so.preload"
)

var (
	BinDirMount    = filepath.Join("mnt", "bin")
	ShareDirMount  = filepath.Join("mnt", "share")
	ConfigDirMount = filepath.Join("mnt", "config")
	EnrichmentPath = filepath.Join("var", "lib", "dynatrace", "enrichment")
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
	config, err := newSecretConfigViaFs(fs)
	if err != nil {
		return nil, err
	}
	env, err := newEnv()
	if err != nil {
		return nil, err
	}
	client, err := newDTClientBuilder(config).createClient()
	if err != nil {
		return nil, err
	}
	oneAgentInstaller := installer.NewOneAgentInstaller(
		fs,
		client,
		installer.InstallerProperties{
			Os:           dtclient.OsUnix,
			Type:         dtclient.InstallerTypePaaS,
			Flavor:       env.installerFlavor,
			Arch:         env.installerArch,
			Technologies: env.installerTech,
			Version:      installer.VersionLatest,
			Url:          env.installerUrl,
		},
	)
	return &Runner{
		fs:        fs,
		env:       env,
		config:    config,
		dtclient:  client,
		installer: oneAgentInstaller,
	}, nil
}

func (runner *Runner) Run() error {
	log.Info("standalone agent init started")
	var err error
	defer runner.consumeErrorIfNecessary(&err)

	if err = runner.setHostTenant(); err != nil {
		return err
	}

	if runner.env.mode == InstallerMode && runner.env.oneAgentInjected {
		if err = runner.installOneAgent(); err != nil {
			return err
		}
	}
	err = runner.configureInstallation()
	if err == nil {
		log.Info("standalone agent init completed")
	}
	return err
}

func (runner *Runner) consumeErrorIfNecessary(err *error) {
	if !runner.env.canFail && err != nil {
		log.Error(*err, "This error has been masked to not fail the container.")
		*err = nil
	}
}

func (runner *Runner) setHostTenant() error {
	log.Info("setting host tenant")
	runner.hostTenant = noHostTenant
	if runner.config.HasHost {
		hostTenant, ok := runner.config.MonitoringNodes[runner.env.k8NodeName]
		if !ok {
			return errors.Errorf("host tenant info is missing for %s", runner.env.k8NodeName)
		}
		runner.hostTenant = hostTenant
	}
	log.Info("successfully set host tenant", "hostTenant", runner.hostTenant)
	return nil
}

func (runner *Runner) installOneAgent() error {
	log.Info("downloading OneAgent")
	return runner.installer.InstallAgent(BinDirMount)
}

func (runner *Runner) configureInstallation() error {
	log.Info("configuring standalone OneAgent")

	if runner.env.oneAgentInjected {
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
		processModuleConfig, err := runner.dtclient.GetProcessModuleConfig(0)
		if err != nil {
			return err
		}
		if err := runner.installer.UpdateProcessModuleConfig(BinDirMount, processModuleConfig); err != nil {
			return err
		}
	}
	if runner.env.dataIngestInjected {
		log.Info("creating enrichment files")
		if err := runner.enrichMetadata(); err != nil {
			return err
		}
	}
	return nil
}

func (runner *Runner) setLDPreload() error {
	return runner.createConfFile(filepath.Join(ShareDirMount, ldPreloadFilename), fmt.Sprintf("%s/agent/lib64/liboneagentproc.so", runner.env.installPath))
}

func (runner *Runner) createContainerConfigurationFiles() error {
	for _, container := range runner.env.containers {
		confFilePath := filepath.Join(ShareDirMount, fmt.Sprintf(containerConfFilenameTemplate, container.name))
		content := runner.getBaseConfContent(container)
		if runner.hostTenant != noHostTenant {
			if runner.config.TenantUUID == runner.hostTenant {
				content += runner.getK8ConfContent()
			}
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
	return runner.createConfFile(filepath.Join(ShareDirMount, "custom.pem"), runner.config.TlsCert)
}
