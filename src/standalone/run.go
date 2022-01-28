package standalone

import (
	"fmt"
	"path/filepath"

	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
)

const (
	noHostTenant = "-"
)

var (
	BinDirMount    = filepath.Join("mnt", "bin")
	ShareDirMount  = filepath.Join("mnt", "share")
	ConfigDirMount = filepath.Join("mnt", "config")
)

type Runner struct {
	fs         afero.Fs
	env        *environment
	client     dtclient.Client
	config     *SecretConfig
	hostTenant string
}

func NewRunner() (*Runner, error) {
	fs := afero.Afero{Fs: afero.NewOsFs()}
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
	return &Runner{
		fs:     fs,
		env:    env,
		client: client,
		config: config,
	}, nil
}

func (runner *Runner) Run() error {
	log.Info("standalone init started")
	var err error
	defer runner.consumeErrorIfNecessary(&err)
	log.Info("%+v", runner.config)
	log.Info("%+v", runner.env)

	if err = runner.setHostTenant(); err != nil {
		return err
	}

	if runner.env.mode != installerMode {
		if err = runner.installOneAgent(); err != nil {
			return err
		}
	}
	err = runner.configureInstallation()
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
	return nil
}

func (runner *Runner) installOneAgent() error {
	log.Info("downloading OneAgent zip")
	// TODO: try to use the implementation from the csi driver
	return nil
}

func (runner *Runner) configureInstallation() error {
	log.Info("configuring standalone OneAgent")

	for _, container := range runner.env.containers {
		confFilePath := filepath.Join(ShareDirMount, fmt.Sprintf("container_%s.conf", container.name))
		content := runner.getBaseConfContent()
		if runner.hostTenant != noHostTenant {
			if runner.config.TenantUUID == runner.hostTenant {
				content += runner.getK8ConfContent()
			}
			content += runner.getHostConfContent()
		}
		if err := createConfFile(confFilePath, content); err != nil {
			return err
		}
	}

	// TODO: Do dataingest/tls/processconfig stuff

	return nil
}

func (runner *Runner) getBaseConfContent() string {
	// TODO: Do stuff
	return ""
}

func (runner *Runner) getK8ConfContent() string {
	// TODO: Do stuff
	return ""
}

func (runner *Runner) getHostConfContent() string {
	// TODO: Do stuff
	return ""
}

func createConfFile(path string, content string) error {
	// TODO: create conf file
	return nil
}
