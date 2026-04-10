package edgeconnect

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core"
)

var _ APIClient = (*Client)(nil)

// APIClient is the interface for the Dynatrace EdgeConnect REST API Client.
type APIClient interface {
	// GetEdgeConnect return details of single EdgeConnect
	GetEdgeConnect(ctx context.Context, id string) (APIResponse, error)

	// CreateEdgeConnect creates EdgeConnect
	CreateEdgeConnect(ctx context.Context, request *Request) (APIResponse, error)

	// UpdateEdgeConnect updates EdgeConnect
	UpdateEdgeConnect(ctx context.Context, id string, request *Request) error

	// DeleteEdgeConnect deletes EdgeConnect
	DeleteEdgeConnect(ctx context.Context, id string) error

	// ListEdgeConnects  list of EdgeConnects
	ListEdgeConnects(ctx context.Context, name string) ([]APIResponse, error)

	// ListEnvironmentSettings  list of environment setting objects
	ListEnvironmentSettings(ctx context.Context) ([]EnvironmentSetting, error)

	// CreateEnvironmentSetting creates environment setting object
	CreateEnvironmentSetting(ctx context.Context, es EnvironmentSetting) error

	// UpdateEnvironmentSetting updates environment setting object
	UpdateEnvironmentSetting(ctx context.Context, es EnvironmentSetting) error

	// DeleteEnvironmentSetting deletes environment setting object
	DeleteEnvironmentSetting(ctx context.Context, objectID string) error
}

type Client struct {
	apiClient core.APIClient
}

func NewClient(apiClient core.APIClient) *Client {
	return &Client{
		apiClient: apiClient,
	}
}
