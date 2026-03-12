package images

import (
	"context"
	"fmt"
	"slices"

	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/opencontainers/go-digest"
)

var log = logd.Get().WithName("dtclient-images")

type ComponentType string

const (
	OneAgent    ComponentType = "oneagent"
	CodeModules ComponentType = "codemodules"
	ActiveGate  ComponentType = "activegate"
)

type ImageInfo struct {
	Tag      string
	Digest   digest.Digest
	Registry string
}

const (
	containerImagesPath = "/v2/fleetManagement/components/containerImages'"
)

type APIClient interface {
	ComponentLatestImageURI(ctx context.Context, component ComponentType, registry string) (string, error)
}

type ComponentResponse struct {
	Type     ComponentType `json:"type"`
	ImageURI string        `json:"imageUri"`
}

type Client struct {
	apiClient core.APIClient
	registry  string
}

func NewClient(apiClient core.APIClient, registry string) *Client {
	return &Client{
		apiClient: apiClient,
		registry:  registry,
	}
}

func (c *Client) ComponentLatestImageURI(ctx context.Context, component ComponentType) (name.Reference, error) {
	var components []ComponentResponse

	err := c.apiClient.GET(ctx, containerImagesPath).Execute(&components)
	if err != nil {
		return nil, fmt.Errorf("get latest %s image: %w", component, err)
	}

	if len(components) == 0 {
		return nil, fmt.Errorf("no %s image found", component)
	}

	idx := slices.IndexFunc(components, func(c ComponentResponse) bool { return c.Type == component })
	if idx != -1 {
		imageRef, err := name.ParseReference(components[idx].ImageURI)
		if err != nil {
			return nil, err
		}
		return imageRef, nil
	}

	return nil, fmt.Errorf("no %s image found", component)
}
