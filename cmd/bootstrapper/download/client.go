package download

import (
	"context"
	"os"
	"path/filepath"

	"github.com/Dynatrace/dynatrace-bootstrapper/pkg/configure/oneagent/ca"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer/binary"
)

type Client struct {
	newInstaller binary.NewFunc
}

type Option func(*Client)

func WithInstaller(builder binary.NewFunc) Option {
	return func(cl *Client) {
		cl.newInstaller = builder
	}
}

func New(options ...Option) *Client {
	cl := &Client{
		newInstaller: binary.NewInstaller,
	}

	for _, opt := range options {
		opt(cl)
	}

	return cl
}

func (cl *Client) Do(ctx context.Context, inputDir string, targetDir string, props binary.Properties) error {
	dtClient, err := cl.createDTClientFromFs(inputDir)
	if err != nil {
		return err
	}

	oneAgentInstaller := cl.newInstaller(
		dtClient,
		&props,
	)

	_, err = oneAgentInstaller.InstallAgent(ctx, targetDir)

	return err
}

func (cl *Client) createDTClientFromFs(inputDir string) (oneagent.Client, error) {
	config, err := configFromFs(inputDir)
	if err != nil {
		return nil, err
	}

	caFile := filepath.Join(inputDir, ca.TrustedCertsInputFile) // TODO: Replace with ca.GetFromFs

	certs, err := os.ReadFile(caFile)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	options := config.toDTClientOptions()

	if len(certs) > 0 {
		options = append(options, dynatrace.WithCerts(certs))
	}

	options = append(options, dynatrace.WithBaseURL(config.URL))

	dtClient, err := dynatrace.NewClient(options...)
	if err != nil {
		return nil, err
	}

	return dtClient.OneAgent, nil
}
