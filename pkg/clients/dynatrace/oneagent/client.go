package oneagent

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
)

var log = logd.Get().WithName("dtclient-oneagent")

type APIClient interface {
	GetConnectionInfo(ctx context.Context) (ConnectionInfo, error)
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
