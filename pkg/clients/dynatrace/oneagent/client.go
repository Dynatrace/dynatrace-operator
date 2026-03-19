package oneagent

import (
	"context"
	"io"

	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
)

var log = logd.Get().WithName("dtclient-oneagent")

type APIClient interface {
	GetConnectionInfo(ctx context.Context) (ConnectionInfo, error)

	Get(ctx context.Context, os, installerType, flavor, arch, version string, technologies []string, skipMetadata bool, writer io.Writer) error
	GetLatest(ctx context.Context, os, installerType, flavor, arch string, technologies []string, skipMetadata bool, writer io.Writer) error
	GetVersions(ctx context.Context, os, installerType, flavor string) ([]string, error)
	GetViaInstallerURL(ctx context.Context, url string, writer io.Writer) error
}

type Client struct {
	apiClient   core.APIClient
	networkZone string
}

func NewClient(apiClient core.APIClient, networkZone string) *Client {
	return &Client{
		apiClient:   apiClient,
		networkZone: networkZone,
	}
}
