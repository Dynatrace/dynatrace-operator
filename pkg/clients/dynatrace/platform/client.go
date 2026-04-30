package platform

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core"
)

type Client interface {
	GetTenantPhase(ctx context.Context) (int, error)
}

type ClientImpl struct {
	apiClient core.Client
}

func NewClient(apiClient core.Client) *ClientImpl {
	return &ClientImpl{
		apiClient: apiClient,
	}
}
