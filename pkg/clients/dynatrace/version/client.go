package version

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core"
)

type Client interface {
	// GetLatestAgentVersion gets the latest agent version for the given OS and installer type.
	// Returns the version as received from the server on success.
	//
	// Returns an error for the following conditions:
	//  - os or installerType is empty
	//  - IO error or unexpected response
	//  - error response from the server (e.g. authentication failure)
	//  - the agent version is not set or empty
	GetLatestAgentVersion(ctx context.Context, os, installerType string) (string, error)

	// GetLatestActiveGateVersion gets the latest gateway version for the given OS and arch.
	// Returns the version as received from the server on success.
	GetLatestActiveGateVersion(ctx context.Context, os string) (string, error)
}

type client struct {
	apiClient core.APIClient
}

func NewClient(apiClient core.APIClient) *client {
	return &client{
		apiClient: apiClient,
	}
}
