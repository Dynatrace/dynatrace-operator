// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package settings

import (
	"context"
	"errors"
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core"
)

var _ core.Cacheable = &getNetworkZonesResponse{}

const (
	networkZonesSchemaID      = "builtin:networkzones.zones"
	networkZonesSchemaVersion = ""
	networkZonesScope         = "environment"
)

var (
	errMissingNetworkZoneID = errors.New("no network zone ID given")
)

// NetworkZoneObject is the relevant info of a `builtin:networkzones.zones` settings object.
type NetworkZoneObject struct {
	ID           string `json:"id"`
	FallbackMode string `json:"fallbackMode"`
}

type getNetworkZonesResponse struct {
	Items      []networkZoneItem `json:"items"`
	TotalCount int               `json:"totalCount"`
}

type networkZoneItem struct {
	Value NetworkZoneObject `json:"value"`
}

// IsEmpty returns true if no network zone was found.
// Such a response must not stay cached, otherwise a zone created right afterwards would not be visible until the cache entry expires.
func (r *getNetworkZonesResponse) IsEmpty() bool {
	return len(r.Items) == 0
}

// GetNetworkZoneObject returns the network zone settings object with the given ID.
// In case the network zone does not exist, we return an empty object, without an error.
func (c *ClientImpl) GetNetworkZoneObject(ctx context.Context, networkZoneID string) (NetworkZoneObject, error) {
	if networkZoneID == "" {
		return NetworkZoneObject{}, errMissingNetworkZoneID
	}

	var response getNetworkZonesResponse

	err := c.apiClient.GET(ctx, ObjectsPath).
		WithQueryParams(map[string]string{
			schemaIDsQueryParam: networkZonesSchemaID,
			fieldsQueryParam:    "value",
			filterQueryParam:    fmt.Sprintf("value.id='%s'", networkZoneID),
		}).
		Execute(&response)
	if err != nil {
		return NetworkZoneObject{}, fmt.Errorf("get network zone object: %w", err)
	}

	if len(response.Items) == 0 {
		return NetworkZoneObject{}, nil
	}

	return response.Items[0].Value, nil
}

// CreateNetworkZoneObject returns the object ID of the created network zone settings object.
func (c *ClientImpl) CreateNetworkZoneObject(ctx context.Context, networkZoneID string, fallbackMode string) (string, error) {
	if networkZoneID == "" {
		return "", errMissingNetworkZoneID
	}

	body := newPostObjectsBody(
		networkZonesSchemaID,
		networkZonesSchemaVersion,
		networkZonesScope,
		NetworkZoneObject{
			ID:           networkZoneID,
			FallbackMode: fallbackMode,
		},
	)

	var response []postObjectsResponse

	err := c.apiClient.POST(ctx, ObjectsPath).
		WithQueryParams(map[string]string{
			validateOnlyQueryParam: "false",
		}).
		WithJSONBody(body).
		Execute(&response)
	if err != nil {
		return "", fmt.Errorf("create network zone object: %w", err)
	}

	return getObjectID(response)
}
