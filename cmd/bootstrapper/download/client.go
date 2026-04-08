package download

import (
	"context"
	"os"
	"path/filepath"

	"github.com/Dynatrace/dynatrace-bootstrapper/pkg/configure/oneagent/ca"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer/url"
)

type Client struct {
	newInstaller url.NewFunc
}

type Option func(*Client)

func WithInstaller(builder url.NewFunc) Option {
	return func(cl *Client) {
		cl.newInstaller = builder
	}
}

func New(options ...Option) *Client {
	cl := &Client{
		newInstaller: url.NewURLInstaller,
	}

	for _, opt := range options {
		opt(cl)
	}

	return cl
}

func (cl *Client) Do(ctx context.Context, inputDir string, targetDir string, props url.Properties) error {
	client, err := cl.createDTClientFromFs(inputDir)
	if err != nil {
		return err
	}

	oneAgentInstaller := cl.newInstaller(
		client,
		&props,
	)

	_, err = oneAgentInstaller.InstallAgent(ctx, targetDir)

	return err
}

func (cl *Client) createDTClientFromFs(inputDir string) (oneagent.APIClient, error) {
	config, err := configFromFs(inputDir)
	if err != nil {
		return nil, err
	}

	caFile := filepath.Join(inputDir, ca.TrustedCertsInputFile) // TODO: Replace with ca.GetFromFs

	certs, err := os.ReadFile(caFile)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	options := config.toDTClientOptionsV2()

	if len(certs) > 0 {
		options = append(options, dtclient.WithV2HTTPOptions(dtclient.WithCerts(certs)))
	}

	client, err := dtclient.NewClientV2(config.URL, options...)
	if err != nil {
		return nil, err
	}

	return client.OneAgent, nil
}
