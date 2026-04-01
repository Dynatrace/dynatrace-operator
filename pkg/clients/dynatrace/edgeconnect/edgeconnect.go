package edgeconnect

import (
	"context"
	"fmt"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha2/edgeconnect"
	"github.com/pkg/errors"
)

// EdgeConnect API
const (
	edgeConnectAPIPath  = "/platform/app-engine/edge-connect/v1"
	edgeConnectsAPIPath = edgeConnectAPIPath + "/edge-connects"
	edgeConnectPath     = edgeConnectsAPIPath + "/%s"
)

type Instance struct {
	Version    string `json:"version,omitempty"`
	InstanceID string `json:"instanceId,omitempty"`
}

type Metadata struct {
	OauthClientStatus string     `json:"oauthClientStatus"`
	Instances         []Instance `json:"instances"`
}

type ModificationInfo struct {
	LastModifiedTime *time.Time `json:"lastModifiedTime,omitempty"`
	LastModifiedBy   string     `json:"lastModifiedBy,omitempty"`
}

type GetResponse struct {
	ModificationInfo           ModificationInfo `json:"modificationInfo"`
	Metadata                   Metadata         `json:"metadata"`
	ID                         string           `json:"id,omitempty"`
	Name                       string           `json:"name"`
	OauthClientID              string           `json:"oauthClientId"`
	HostPatterns               []string         `json:"hostPatterns"`
	ManagedByDynatraceOperator bool             `json:"managedByDynatraceOperator,omitempty"`
}

type CreateResponse struct {
	ModificationInfo           ModificationInfo          `json:"modificationInfo"`
	Metadata                   Metadata                  `json:"metadata"`
	ID                         string                    `json:"id,omitempty"`
	Name                       string                    `json:"name"`
	OauthClientID              string                    `json:"oauthClientId"`
	OauthClientSecret          string                    `json:"oauthClientSecret"`
	OauthClientResource        string                    `json:"oauthClientResource"`
	HostPatterns               []string                  `json:"hostPatterns"`
	HostMappings               []edgeconnect.HostMapping `json:"hostMappings"`
	ManagedByDynatraceOperator bool                      `json:"managedByDynatraceOperator,omitempty"`
}

type ListResponse struct {
	EdgeConnects []GetResponse `json:"edgeConnects"`
	TotalCount   int           `json:"totalCount"`
}

type Request struct {
	Name                       string                    `json:"name"`
	OauthClientID              string                    `json:"oauthClientId,omitempty"`
	HostPatterns               []string                  `json:"hostPatterns"`
	HostMappings               []edgeconnect.HostMapping `json:"hostMappings"`
	ManagedByDynatraceOperator bool                      `json:"managedByDynatraceOperator,omitempty"`
}

func NewRequest(name string, hostPatterns []string, hostMappings []edgeconnect.HostMapping, oauthClientID string) *Request {
	return &Request{
		Name:                       name,
		HostPatterns:               hostPatterns,
		HostMappings:               hostMappings,
		OauthClientID:              oauthClientID,
		ManagedByDynatraceOperator: true,
	}
}

// GetEdgeConnect returns EdgeConnect if it exists
func (c *client) GetEdgeConnect(ctx context.Context, edgeConnectID string) (GetResponse, error) {
	if edgeConnectID == "" {
		return GetResponse{}, errors.New("no EdgeConnect ID given")
	}

	var response GetResponse

	err := c.apiClient.GET(ctx, fmt.Sprintf(edgeConnectPath, edgeConnectID)).WithOAuthToken().Execute(&response)
	if err != nil {
		return GetResponse{}, errors.Wrap(err, "failed to get EdgeConnect")
	}

	return response, nil
}

// CreateEdgeConnect creates new EdgeConnect
func (c *client) CreateEdgeConnect(ctx context.Context, request *Request) (CreateResponse, error) {
	var response CreateResponse

	err := c.apiClient.POST(ctx, edgeConnectsAPIPath).WithOAuthToken().WithJSONBody(request).Execute(&response)
	if err != nil {
		return CreateResponse{}, errors.Wrap(err, "failed to create EdgeConnect")
	}

	return response, nil
}

// UpdateEdgeConnect updates existing EdgeConnect
func (c *client) UpdateEdgeConnect(ctx context.Context, edgeConnectID string, request *Request) error {
	if edgeConnectID == "" {
		return errors.New("no EdgeConnect ID given")
	}

	err := c.apiClient.PUT(ctx, fmt.Sprintf(edgeConnectPath, edgeConnectID)).WithOAuthToken().WithJSONBody(request).Execute(nil)
	if err != nil {
		return errors.Wrap(err, "failed to update EdgeConnect")
	}

	return nil
}

// GetEdgeConnects get list of EdgeConnects
func (c *client) GetEdgeConnects(ctx context.Context, name string) (ListResponse, error) {
	var response ListResponse

	qp := map[string]string{
		"add-fields": "name,managedByDynatraceOperator",
		"filter":     fmt.Sprintf("name='%s'", name),
	}

	err := c.apiClient.GET(ctx, edgeConnectsAPIPath).WithOAuthToken().WithQueryParams(qp).Execute(&response)
	if err != nil {
		return ListResponse{}, errors.Wrap(err, "failed to get EdgeConnects")
	}

	return response, nil
}

// DeleteEdgeConnect deletes EdgeConnect using DELETE method for given edgeConnectId
func (c *client) DeleteEdgeConnect(ctx context.Context, edgeConnectID string) error {
	if edgeConnectID == "" {
		return errors.New("no EdgeConnect ID given")
	}

	err := c.apiClient.DELETE(ctx, fmt.Sprintf(edgeConnectPath, edgeConnectID)).WithOAuthToken().Execute(nil)
	if err != nil {
		return errors.Wrap(err, "failed to delete EdgeConnect")
	}

	return nil
}
