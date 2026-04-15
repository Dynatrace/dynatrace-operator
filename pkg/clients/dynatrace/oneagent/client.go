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

	Get(ctx context.Context, args GetParams, writer io.Writer) error
	GetLatest(ctx context.Context, args GetParams, writer io.Writer) error
	GetVersions(ctx context.Context, args GetParams) ([]string, error)
	GetViaInstallerURL(ctx context.Context, url string, writer io.Writer) error

	GetProcessModuleConfig(ctx context.Context) (*ProcessModuleConfig, error)
	GetProcessGroupingConfig(ctx context.Context, kubernetesClusterId string, etag string, writer io.Writer) (string, error)
}

type Client struct {
	apiClient core.APIClient

	hostGroup   string
	networkZone string
}

func NewClient(apiClient core.APIClient, hostGroup, networkZone string) *Client {
	return &Client{
		apiClient:   apiClient,
		hostGroup:   hostGroup,
		networkZone: networkZone,
	}
}
