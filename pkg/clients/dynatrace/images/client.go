package images

import (
	"context"
	"fmt"
	"slices"

	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/opencontainers/go-digest"
)

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

type Client interface {
	ComponentLatestImageURI(ctx context.Context, component ComponentType, registry string) (string, error)
}

type ComponentResponse struct {
	Type     ComponentType `json:"type"`
	ImageURI string        `json:"imageUri"`
}

type ClientImpl struct {
	apiClient core.Client
}

func NewClient(apiClient core.Client) *ClientImpl {
	return &ClientImpl{
		apiClient: apiClient,
	}
}

func (c *ClientImpl) ComponentLatestImageURI(ctx context.Context, component ComponentType, registry string) (string, error) {
	var components []ComponentResponse

	params := map[string]string{}
	if registry != "" {
		params["registry"] = registry
	}

	err := c.apiClient.GET(ctx, containerImagesPath).WithQueryParams(params).Execute(&components)
	if err != nil {
		return "", fmt.Errorf("get latest %s image: %w", component, err)
	}

	if len(components) == 0 {
		return "", fmt.Errorf("no %s image found", component)
	}

	idx := slices.IndexFunc(components, func(c ComponentResponse) bool { return c.Type == component })
	if idx != -1 {
		imageRef, err := name.ParseReference(components[idx].ImageURI)
		if err != nil {
			return "", err
		}
		return imageRef.String(), nil
	}

	return "", fmt.Errorf("no %s image found", component)
}
