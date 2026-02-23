package image

import (
	"context"
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core"
)

// var log = logd.Get().WithName("dtclient-image")

const (
	LatestOneAgentImagePath    = "/v1/deployment/image/agent/oneAgent/latest"
	LatestCodeModulesImagePath = "/v1/deployment/image/agent/codeModules/latest"
	LatestActiveGateImagePath  = "/v1/deployment/image/gateway/latest"
)

type APIClient interface {
	LatestOneAgentImage(ctx context.Context) (*LatestImageInfo, error)
	LatestCodeModulesImage(ctx context.Context) (*LatestImageInfo, error)
	LatestActiveGateImage(ctx context.Context) (*LatestImageInfo, error)
}

type LatestImageInfo struct {
	Source string `json:"source"`
	Tag    string `json:"tag"`
}

type Client struct {
	apiClient core.APIClient
}

func (image LatestImageInfo) String() string {
	return image.Source + ":" + image.Tag
}

func NewClient(apiClient core.APIClient) *Client {
	return &Client{
		apiClient: apiClient,
	}
}

func (c *Client) LatestOneAgentImage(ctx context.Context) (*LatestImageInfo, error) {
	latestImageInfo := &LatestImageInfo{}

	err := c.apiClient.GET(ctx, LatestOneAgentImagePath).Execute(latestImageInfo)
	if err != nil {
		return &LatestImageInfo{}, fmt.Errorf("get latest OneAgent image: %w", err)
	}

	return latestImageInfo, nil
}

func (c *Client) LatestCodeModulesImage(ctx context.Context) (*LatestImageInfo, error) {
	latestImageInfo := &LatestImageInfo{}

	err := c.apiClient.GET(ctx, LatestCodeModulesImagePath).Execute(latestImageInfo)
	if err != nil {
		return &LatestImageInfo{}, fmt.Errorf("get latest CodeModules image: %w", err)
	}

	return latestImageInfo, nil
}

func (c *Client) LatestActiveGateImage(ctx context.Context) (*LatestImageInfo, error) {
	latestImageInfo := &LatestImageInfo{}

	err := c.apiClient.GET(ctx, LatestActiveGateImagePath).Execute(latestImageInfo)
	if err != nil {
		return &LatestImageInfo{}, fmt.Errorf("get latest ActiveGate image: %w", err)
	}

	return latestImageInfo, nil
}
