package edgeconnect

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core"
)

// APIClient is the interface for the Dynatrace EdgeConnect REST API client.
type APIClient interface {
	// GetEdgeConnect return details of single EdgeConnect
	GetEdgeConnect(ctx context.Context, edgeConnectID string) (GetResponse, error)

	// CreateEdgeConnect creates EdgeConnect
	CreateEdgeConnect(ctx context.Context, request *Request) (CreateResponse, error)

	// UpdateEdgeConnect updates EdgeConnect
	UpdateEdgeConnect(ctx context.Context, edgeConnectID string, request *Request) error

	// DeleteEdgeConnect deletes EdgeConnect
	DeleteEdgeConnect(ctx context.Context, edgeConnectID string) error

	// GetEdgeConnects returns list of EdgeConnects
	GetEdgeConnects(ctx context.Context, name string) (ListResponse, error)

	// GetEnvironmentSettings returns all connection setting objects
	GetEnvironmentSettings(ctx context.Context) ([]EnvironmentSetting, error)

	// CreateEnvironmentSetting creates a connection setting object
	CreateEnvironmentSetting(ctx context.Context, es EnvironmentSetting) error

	// UpdateEnvironmentSetting updates a connection setting object
	UpdateEnvironmentSetting(ctx context.Context, es EnvironmentSetting) error

	// DeleteEnvironmentSetting deletes a connection setting object
	DeleteEnvironmentSetting(ctx context.Context, objectID string) error
}

type client struct {
	apiClient core.APIClient
}

func NewClient(apiClient core.APIClient) APIClient {
	return &client{
		apiClient: apiClient,
	}
}
