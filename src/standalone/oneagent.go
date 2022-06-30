package standalone

import (
	"github.com/Dynatrace/dynatrace-operator/src/arch"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/installer"
	"github.com/Dynatrace/dynatrace-operator/src/installer/url"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
)

type oneAgentSetup struct {
	fs         afero.Fs
	env        *environment
	config     *SecretConfig
	dtclient   dtclient.Client
	installer  installer.Installer
	hostTenant string
}

func newOneagentSetup(fs afero.Fs, env *environment) (*oneAgentSetup, error) {
	config, err := newSecretConfigViaFs(fs)
	if err != nil {
		return nil, err
	}

	client, err := newDTClientBuilder(config).createClient()
	if err != nil {
		return nil, err
	}

	oneAgentInstaller := url.NewUrlInstaller(
		fs,
		client,
		&url.Properties{
			Os:           dtclient.OsUnix,
			Type:         dtclient.InstallerTypePaaS,
			Flavor:       env.InstallerFlavor,
			Arch:         arch.Arch,
			Technologies: env.InstallerTech,
			Version:      url.VersionLatest,
			Url:          env.InstallerUrl,
		},
	)
	return &oneAgentSetup{
		fs:        fs,
		env:       env,
		config:    config,
		dtclient:  client,
		installer: oneAgentInstaller,
	}, nil
}

func (setup *oneAgentSetup) setup() error {
	log.Info("standalone oneagent init started")

	if setup.env.OneAgentInjected {
		if err := setup.setHostTenant(); err != nil {
			return err
		}

		if setup.env.Mode == InstallerMode {
			if err := setup.installOneAgent(); err != nil {
				return err
			}
			log.Info("OneAgent download finished")
		}
	}

	err := setup.configureInstallation()
	if err == nil {
		log.Info("standalone agent init completed")
	}
	return err
}

func (setup *oneAgentSetup) setHostTenant() error {
	log.Info("setting host tenant")
	setup.hostTenant = NoHostTenant
	if setup.config.HasHost {
		hostTenant, ok := setup.config.MonitoringNodes[setup.env.K8NodeName]
		if !ok {
			return errors.Errorf("host tenant info is missing for %s", setup.env.K8NodeName)
		}
		setup.hostTenant = hostTenant
	}
	log.Info("successfully set host tenant", "hostTenant", setup.hostTenant)
	return nil
}

func (setup *oneAgentSetup) installOneAgent() error {
	log.Info("downloading OneAgent")
	_, err := setup.installer.InstallAgent(BinDirMount)
	if err != nil {
		return err
	}
	processModuleConfig, err := setup.dtclient.GetProcessModuleConfig(0)
	if err != nil {
		return err
	}
	if err := setup.installer.UpdateProcessModuleConfig(BinDirMount, processModuleConfig); err != nil {
		return err
	}
	return nil
}

func (setup *oneAgentSetup) configureInstallation() error {
	log.Info("configuring standalone OneAgent")

	log.Info("setting ld.so.preload")
	if err := setup.setLDPreload(); err != nil {
		return errors.WithStack(err)
	}

	log.Info("creating container configuration files")
	if err := setup.createContainerConfigurationFiles(); err != nil {
		return errors.WithStack(err)
	}

	if setup.config.TlsCert != "" {
		log.Info("propagating tls cert to agent")
		if err := setup.propagateTLSCert(); err != nil {
			return errors.WithStack(err)
		}
	}

	if setup.config.InitialConnectRetry > -1 {
		log.Info("creating curl options file")
		if err := setup.createCurlOptionsFile(); err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}
