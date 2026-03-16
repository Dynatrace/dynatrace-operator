package oneagent

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core"
)

type APIClient interface {
}

type Client struct {
	apiClient core.APIClient
}

func NewClient(apiClient core.APIClient) *Client {
	return &Client{
		apiClient: apiClient,
	}
}
