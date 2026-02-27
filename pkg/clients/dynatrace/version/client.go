package version

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core"
)

// Relevant installer types.
const (
	InstallerTypeDefault = "default"
	InstallerTypePaaS    = "paas"
)

const (
	OsUnix = "unix"
	// Commented for linter, left for further reference
	// OsWindows = "windows"
	// OsAix     = "aix"
	// OsSolaris = "solaris"
)

type APIClient interface {
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

type Client struct {
	apiClient core.APIClient
}

func NewClient(apiClient core.APIClient) *Client {
	return &Client{
		apiClient: apiClient,
	}
}
