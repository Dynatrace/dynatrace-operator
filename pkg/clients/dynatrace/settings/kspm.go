// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package settings

import (
	"context"
	"errors"
	"fmt"
)

const (
	kspmSettingsSchemaID      = "builtin:kubernetes.security-posture-management"
	kspmSettingsSchemaVersion = "1"
)

type KSPMSettingsResponse struct {
	TotalCount int                `json:"totalCount"`
	Items      []KSPMSettingsItem `json:"items"`
}

type KSPMSettingsItem struct {
	ObjectID string            `json:"objectId"`
	Value    KSPMSettingsValue `json:"value"`
}

type KSPMSettingsValue struct {
	DatasetPipelineEnabled bool `json:"configurationDatasetPipelineEnabled"`
}

// GetKSPMSettings returns the settings response with the number of settings objects and their values.
func (c *ClientImpl) GetKSPMSettings(ctx context.Context, registration K8sClusterRegistration) (KSPMSettingsResponse, error) {
	if registration.EntityScope == "" {
		return KSPMSettingsResponse{}, nil
	}

	var resp KSPMSettingsResponse

	err := c.api.listObjects(ctx, objectsCollection, listParams{
		SchemaID:     kspmSettingsSchemaID,
		Scope:        c.api.scope(registration),
		ValidateOnly: true,
	}, &resp)
	if err != nil {
		return KSPMSettingsResponse{}, fmt.Errorf("get kspm settings: %w", err)
	}

	return resp, nil
}

// CreateKSPMSetting returns the object ID of the created kspm settings.
func (c *ClientImpl) CreateKSPMSetting(ctx context.Context, registration K8sClusterRegistration, datasetPipelineEnabled bool) (string, error) {
	if registration.EntityScope == "" {
		return "", errors.New("no scope (MEID) was provided for creating the KSPM setting object")
	}

	ids, err := c.api.createObject(ctx, kspmSettingsSchemaID, kspmSettingsSchemaVersion, c.api.scope(registration),
		KSPMSettingsValue{DatasetPipelineEnabled: datasetPipelineEnabled})
	if err != nil {
		return "", fmt.Errorf("create kspm setting: %w", err)
	}

	return singleID(ids)
}
