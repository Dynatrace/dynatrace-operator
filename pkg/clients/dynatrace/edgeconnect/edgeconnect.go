package edgeconnect

import (
	"context"
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha2/edgeconnect"
	"github.com/pkg/errors"
)

// EdgeConnect API
const (
	edgeConnectsPath = "/platform/app-engine/edge-connect/v1/edge-connects"
)

var errNoEdgeConnectID = errors.New("no EdgeConnect ID given")

type APIResponse struct {
	ID                         string   `json:"id"`
	Name                       string   `json:"name"`
	OauthClientID              string   `json:"oauthClientId"`
	OauthClientSecret          string   `json:"oauthClientSecret"`
	OauthClientResource        string   `json:"oauthClientResource"`
	HostPatterns               []string `json:"hostPatterns"`
	ManagedByDynatraceOperator bool     `json:"managedByDynatraceOperator"`
}

type listResponse struct {
	EdgeConnects []APIResponse `json:"edgeConnects"`
}

type Request struct {
	Name                       string                    `json:"name"`
	OauthClientID              string                    `json:"oauthClientId,omitempty"`
	HostPatterns               []string                  `json:"hostPatterns"`
	HostMappings               []edgeconnect.HostMapping `json:"hostMappings"`
	ManagedByDynatraceOperator bool                      `json:"managedByDynatraceOperator,omitempty"`
}

func newRequest(name string, hostPatterns []string, hostMappings []edgeconnect.HostMapping, oauthClientID string) *Request {
	return &Request{
		Name:                       name,
		HostPatterns:               hostPatterns,
		HostMappings:               hostMappings,
		OauthClientID:              oauthClientID,
		ManagedByDynatraceOperator: true,
	}
}

func NewCreateRequest(name string, hostPatterns []string, hostMappings []edgeconnect.HostMapping) *Request {
	return newRequest(name, hostPatterns, hostMappings, "")
}

func NewUpdateRequest(name string, hostPatterns []string, hostMappings []edgeconnect.HostMapping, oauthClientID string) *Request {
	return newRequest(name, hostPatterns, hostMappings, oauthClientID)
}

// GetEdgeConnect returns EdgeConnect if it exists
func (c *Client) GetEdgeConnect(ctx context.Context, id string) (APIResponse, error) {
	if id == "" {
		return APIResponse{}, errNoEdgeConnectID
	}

	var response APIResponse

	err := c.apiClient.GET(ctx, edgeConnectsPath).WithPath(id).WithoutToken().Execute(&response)
	if err != nil {
		return APIResponse{}, errors.Wrap(err, "failed to get EdgeConnect")
	}

	return response, nil
}

// CreateEdgeConnect creates new EdgeConnect
func (c *Client) CreateEdgeConnect(ctx context.Context, request *Request) (APIResponse, error) {
	var response APIResponse

	err := c.apiClient.POST(ctx, edgeConnectsPath).WithoutToken().WithJSONBody(request).Execute(&response)
	if err != nil {
		return APIResponse{}, errors.Wrap(err, "failed to create EdgeConnect")
	}

	return response, nil
}

// UpdateEdgeConnect updates existing EdgeConnect
func (c *Client) UpdateEdgeConnect(ctx context.Context, id string, request *Request) error {
	if id == "" {
		return errNoEdgeConnectID
	}

	err := c.apiClient.PUT(ctx, edgeConnectsPath).WithPath(id).WithoutToken().WithJSONBody(request).Execute(nil)
	if err != nil {
		return errors.Wrap(err, "failed to update EdgeConnect")
	}

	return nil
}

// ListEdgeConnects get list of EdgeConnects
func (c *Client) ListEdgeConnects(ctx context.Context, name string) ([]APIResponse, error) {
	var response listResponse

	qp := map[string]string{
		"add-fields": "name,managedByDynatraceOperator",
		"filter":     fmt.Sprintf("name='%s'", name),
	}

	err := c.apiClient.GET(ctx, edgeConnectsPath).WithoutToken().WithQueryParams(qp).Execute(&response)
	if err != nil {
		return []APIResponse{}, errors.Wrap(err, "failed to get EdgeConnects")
	}

	return response.EdgeConnects, nil
}

// DeleteEdgeConnect deletes EdgeConnect using DELETE method for given id
func (c *Client) DeleteEdgeConnect(ctx context.Context, id string) error {
	if id == "" {
		return errNoEdgeConnectID
	}

	err := c.apiClient.DELETE(ctx, edgeConnectsPath).WithPath(id).WithoutToken().Execute(nil)
	if err != nil {
		return errors.Wrap(err, "failed to delete EdgeConnect")
	}

	return nil
}
