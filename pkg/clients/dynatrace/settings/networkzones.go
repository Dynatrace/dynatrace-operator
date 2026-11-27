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
	networkZonesSchemaVersion = "1.0.1"
	networkZonesScope         = "environment"
)

var (
	errMissingNetworkZoneID = errors.New("no network zone ID given")
)

// NetworkZoneObject is the relevant info of a `builtin:networkzones.zones` settings object.
type NetworkZoneObject struct {
	ID               string   `json:"id"`
	FallbackMode     string   `json:"fallbackMode"`
	Description      string   `json:"description,omitempty"`
	AlternativeZones []string `json:"alternativeZones"`
}

type getNetworkZonesResponse struct {
	Items      []networkZoneItem `json:"items"`
	TotalCount int               `json:"totalCount"`
}

type networkZoneItem struct {
	Value NetworkZoneObject `json:"value"`
}

// IsEmpty always returns false: all responses are cached, including an empty list (zone does not exist).
func (r *getNetworkZonesResponse) IsEmpty() bool {
	return false
}

// GetNetworkZoneObject returns the network zone with the given ID, or an empty object if it does not exist.
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

// CreateNetworkZoneObject returns the object ID of the created network zone.
// An existing zone with the same ID is replaced, so pass the object from GetNetworkZoneObject to keep its other fields.
func (c *ClientImpl) CreateNetworkZoneObject(ctx context.Context, zone NetworkZoneObject) (string, error) {
	if zone.ID == "" {
		return "", errMissingNetworkZoneID
	}

	if zone.AlternativeZones == nil {
		zone.AlternativeZones = []string{}
	}

	body := newPostObjectsBody(
		networkZonesSchemaID,
		networkZonesSchemaVersion,
		networkZonesScope,
		zone,
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
