package standalone

import (
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/spf13/afero"
)

type Runner struct {
	fs     afero.Fs
	env    *environment
	client dtclient.Client
	config *SecretConfig
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
	client, err := NewDTClientBuilder(config).CreateClient()
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

func (runner Runner) Run() error {
	log.Info("%+v", runner.config)
	log.Info("%+v", runner.env)
	return nil
}
