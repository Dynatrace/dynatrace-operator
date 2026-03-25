package oneagent

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
)

var log = logd.Get().WithName("dtclient-oneagent")

type APIClient interface {
	GetConnectionInfo(ctx context.Context) (ConnectionInfo, error)

	GetProcessModuleConfig(ctx context.Context) (*ProcessModuleConfig, error)
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
