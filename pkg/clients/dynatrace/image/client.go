package image

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core"
	"github.com/google/go-containerregistry/pkg/name"
)

type ComponentType string

const (
	OneAgent    ComponentType = "oneagent"
	CodeModules ComponentType = "codemodules"
	ActiveGate  ComponentType = "activegate"
)

type Info struct {
	URI      string
	Tag      string
	Registry string
}

const (
	containerImagesPath = "/v2/fleetManagement/components/containerImages"
)

type Client interface {
	ComponentLatestImageInfo(ctx context.Context, component ComponentType, registry string) (*Info, error)
}

type componentResponse struct {
	Type     ComponentType `json:"type"`
	ImageURI string        `json:"imageUri"`
}

type containerImagesResponse struct {
	Components []componentResponse `json:"components"`
}

type ClientImpl struct {
	apiClient core.Client
}

func NewClient(apiClient core.Client) *ClientImpl {
	return &ClientImpl{
		apiClient: apiClient,
	}
}

func (c *ClientImpl) ComponentLatestImageInfo(ctx context.Context, component ComponentType, registry string) (*Info, error) {
	var resp containerImagesResponse

	params := map[string]string{}
	if registry != "" {
		params["registry"] = registry
	}

	err := c.apiClient.GET(ctx, containerImagesPath).WithQueryParams(params).Execute(&resp)
	if err != nil {
		return nil, fmt.Errorf("get latest %s image: %w", component, err)
	}

	if len(resp.Components) == 0 {
		return nil, fmt.Errorf("no %s image found", component)
	}

	idx := slices.IndexFunc(resp.Components, func(c componentResponse) bool { return c.Type == component })
	if idx == -1 {
		return nil, fmt.Errorf("no %s image found", component)
	}

	imageInfo, err := parseImageInfo(resp.Components[idx].ImageURI)
	if err != nil {
		return nil, err
	}

	if registry != "" && imageInfo != nil && imageInfo.Registry != registry {
		return nil, fmt.Errorf("image registry %q does not match requested registry %q", imageInfo.Registry, registry)
	}

	return imageInfo, nil
}

func parseImageInfo(imageURI string) (*Info, error) {
	ref, err := name.ParseReference(imageURI, name.WithDefaultTag(""))
	if err != nil {
		return nil, fmt.Errorf("parse image URI %q: %w", imageURI, err)
	}

	info := &Info{
		URI:      imageURI,
		Registry: ref.Context().RegistryStr(),
	}

	// cut the digest part <imagePart with/or without tag>@digest
	imagePart, _, _ := strings.Cut(imageURI, "@")

	// Parse the image part to extract tag
	if tag, err := name.NewTag(imagePart, name.WithDefaultTag("")); err == nil {
		info.Tag = tag.TagStr()
	}

	return info, nil
}
