package download

import (
	"context"
	"os"
	"path/filepath"

	"github.com/Dynatrace/dynatrace-bootstrapper/pkg/configure/oneagent/ca"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer/url"
	"github.com/spf13/afero"
)

type Client struct {
	newInstaller url.NewFunc
	newDTClient  dtclient.NewFunc
}

type Option func(*Client)

func WithInstaller(builder url.NewFunc) Option {
	return func(cl *Client) {
		cl.newInstaller = builder
	}
}

func WithDTClient(builder dtclient.NewFunc) Option {
	return func(cl *Client) {
		cl.newDTClient = builder
	}
}

func New(options ...Option) *Client {
	cl := &Client{
		newInstaller: url.NewURLInstaller,
		newDTClient:  dtclient.NewClient,
	}

	for _, opt := range options {
		opt(cl)
	}

	return cl
}

func (cl *Client) Do(ctx context.Context, fs afero.Afero, inputDir string, targetDir string, props url.Properties) error {
	client, err := cl.createDTClientFromFs(fs, inputDir)
	if err != nil {
		return err
	}

	oneAgentInstaller := cl.newInstaller(
		fs.Fs,
		client,
		&props,
	)

	_, err = oneAgentInstaller.InstallAgent(ctx, targetDir)

	return err
}

func (cl *Client) createDTClientFromFs(fs afero.Afero, inputDir string) (dtclient.Client, error) {
	config, err := configFromFs(fs, inputDir)
	if err != nil {
		return nil, err
	}

	caFile := filepath.Join(inputDir, ca.TrustedCertsInputFile) // TODO: Replace with ca.GetFromFs

	certs, err := fs.ReadFile(caFile)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	options := config.toDTClientOptions()

	if len(certs) > 0 {
		options = append(options, dtclient.Certs(certs))
	}

	client, err := cl.newDTClient(
		config.URL,
		config.APIToken,
		config.APIToken,
		options...,
	)
	if err != nil {
		return nil, err
	}

	return client, nil
}
