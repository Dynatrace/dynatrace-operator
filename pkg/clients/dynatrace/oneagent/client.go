package oneagent

import (
	"context"
	"io"

	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
)

var log = logd.Get().WithName("dtclient-oneagent")

type Client interface {
	GetConnectionInfo(ctx context.Context) (ConnectionInfo, error)

	Get(ctx context.Context, args GetParams, writer io.Writer) error
	GetLatest(ctx context.Context, args GetParams, writer io.Writer) error
	GetVersions(ctx context.Context, args GetParams) ([]string, error)

	GetProcessModuleConfig(ctx context.Context) (*ProcessModuleConfig, error)
	GetProcessGroupingConfig(ctx context.Context, kubernetesClusterID string, etag string, writer io.Writer) (string, error)
}

type client struct {
	apiClient core.Client

	hostGroup   string
	networkZone string
}

func NewClient(apiClient core.Client, hostGroup, networkZone string) *client {
	return &client{
		apiClient:   apiClient,
		hostGroup:   hostGroup,
		networkZone: networkZone,
	}
}
