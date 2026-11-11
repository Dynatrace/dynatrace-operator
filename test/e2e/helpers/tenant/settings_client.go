// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

//go:build e2e

package tenant

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/metadataenrichment"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core"
	dtsettings "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/settings"
)

const (
	LegacyMetadataEnrichmentSchemaID = "builtin:kubernetes.generic.metadata.enrichment"
	MetadataEnrichmentSchemaID       = "builtin:ingest.enrichment.config"

	objectsPath            = dtsettings.ObjectsPath
	fieldsQueryParam       = "fields"
	validateOnlyQueryParam = "validateOnly"
	schemaIDsQueryParam    = "schemaIds"
	scopesQueryParam       = "scopes"
)

var (
	errDeleteSettings       = errors.New("delete monitored entity settings failed")
	errNoSettingsIDProvided = errors.New("no settings ID provided")
)

// SettingsClient extends the operator's settings client with operations that are only needed by the e2e tests.
type SettingsClient struct {
	dtsettings.Client
	apiClient core.Client
}

// RawObject is a settings object with its value kept as raw JSON, which allows restoring it later on.
type RawObject struct {
	ObjectID string          `json:"objectId"`
	Value    json.RawMessage `json:"value,omitempty"`
}

type rawObjectsResponse struct {
	Items []RawObject `json:"items"`
}

type enrichmentRuleValue struct {
	Type        metadataenrichment.RuleType `json:"type"`
	ValueSource string                      `json:"valueSource"`
	Target      string                      `json:"target"`
}

type legacyEnrichmentValue struct {
	Rules []metadataenrichment.Rule `json:"rules"`
}

type postObjectsResponse struct {
	ObjectID string `json:"objectId"`
}

type postObjectsBody[T any] struct {
	SchemaID      string `json:"schemaId"`
	SchemaVersion string `json:"schemaVersion,omitempty"`
	Scope         string `json:"scope,omitempty"`
	Value         T      `json:"value"`
}

func newPostObjectsBody[T any](schemaID, schemaVersion, scope string, values ...T) []postObjectsBody[T] {
	body := make([]postObjectsBody[T], len(values))

	for i, value := range values {
		body[i] = postObjectsBody[T]{SchemaID: schemaID, SchemaVersion: schemaVersion, Scope: scope, Value: value}
	}

	return body
}

func BuildSettingsClient(secretConfig Secret) (*SettingsClient, error) {
	baseURL, err := url.Parse(secretConfig.APIURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	apiToken := secretConfig.TokensWithSettingsScope().APIToken
	if apiToken == "" {
		return nil, errors.New("api token is empty")
	}

	return newSettingsClient(core.NewClient(core.Config{
		BaseURL:    baseURL,
		HTTPClient: &http.Client{},
		APIToken:   apiToken,
	})), nil
}

func newSettingsClient(apiClient core.Client) *SettingsClient {
	return &SettingsClient{
		Client:    dtsettings.NewClient(apiClient),
		apiClient: apiClient,
	}
}

// CreateEnrichmentRuleObject creates a settings object for the builtin:ingest.enrichment.config schema.
func (c *SettingsClient) CreateEnrichmentRuleObject(ctx context.Context, scope string, rules ...metadataenrichment.Rule) ([]string, error) {
	return c.createEnrichmentRule(ctx, MetadataEnrichmentSchemaID, scope, rules)
}

// CreateLegacyEnrichmentRuleObject creates a settings object for the builtin:kubernetes.generic.metadata.enrichment schema.
// The schema allows a single object per scope holding all rules, so this fails if the scope already has one.
func (c *SettingsClient) CreateLegacyEnrichmentRuleObject(ctx context.Context, scope string, rules ...metadataenrichment.Rule) ([]string, error) {
	return c.createEnrichmentRule(ctx, LegacyMetadataEnrichmentSchemaID, scope, rules)
}

func (c *SettingsClient) createEnrichmentRule(ctx context.Context, schemaID, scope string, rules []metadataenrichment.Rule) ([]string, error) {
	if scope == "" {
		return nil, errors.New("no scope (MEID) was provided for creating the enrichment rule")
	}

	buildBody := func() any {
		if schemaID == MetadataEnrichmentSchemaID {
			values := make([]enrichmentRuleValue, len(rules))
			for i, rule := range rules {
				values[i] = convertRule(rule)
			}

			return newPostObjectsBody(schemaID, "", scope, values...)
		}

		return newPostObjectsBody(schemaID, "", scope, legacyEnrichmentValue{Rules: rules})
	}

	var response []postObjectsResponse

	err := c.apiClient.POST(ctx, objectsPath).
		WithQueryParams(map[string]string{validateOnlyQueryParam: "false"}).
		WithJSONBody(buildBody()).
		Execute(&response)
	if err != nil {
		return nil, fmt.Errorf("create enrichment rule (%s): %w", schemaID, err)
	}

	objectIDs := make([]string, len(response))
	for i, resp := range response {
		objectIDs[i] = resp.ObjectID
	}

	return objectIDs, nil
}

func convertRule(rule metadataenrichment.Rule) enrichmentRuleValue {
	return enrichmentRuleValue{
		Type:        rule.Type,
		ValueSource: rule.Source,
		Target:      rule.Target,
	}
}

// GetRawObjects returns the settings objects (with objectId and raw value) of the given schema for the given scope.
// Does not handle pagination.
func (c *SettingsClient) GetRawObjects(ctx context.Context, schemaID, scope string) ([]RawObject, error) {
	if scope == "" {
		return nil, errors.New("no scope provided for getting settings objects")
	}

	var resp rawObjectsResponse

	err := c.apiClient.GET(ctx, objectsPath).
		WithQueryParams(map[string]string{
			schemaIDsQueryParam: schemaID,
			scopesQueryParam:    scope,
			fieldsQueryParam:    "objectId,value",
		}).
		Execute(&resp)
	if err != nil {
		return nil, fmt.Errorf("get settings objects (%s): %w", schemaID, err)
	}

	return resp.Items, nil
}

// CreateRawObject creates a settings object with the given raw value, e.g. to restore a previously backed up object.
func (c *SettingsClient) CreateRawObject(ctx context.Context, schemaID, scope string, value json.RawMessage) (string, error) {
	if scope == "" {
		return "", errors.New("no scope (MEID) was provided for creating the settings object")
	}

	var response []postObjectsResponse

	err := c.apiClient.POST(ctx, objectsPath).
		WithQueryParams(map[string]string{validateOnlyQueryParam: "false"}).
		WithJSONBody(newPostObjectsBody(schemaID, "", scope, value)).
		Execute(&response)
	if err != nil {
		return "", fmt.Errorf("create settings object (%s): %w", schemaID, err)
	}

	if len(response) != 1 {
		return "", fmt.Errorf("expected exactly one created object, got %d", len(response))
	}

	return response[0].ObjectID, nil
}

// DeleteSettings deletes the settings using the settings object ID.
func (c *SettingsClient) DeleteSettings(ctx context.Context, objectID string) error {
	if objectID == "" {
		return errNoSettingsIDProvided
	}

	err := c.apiClient.DELETE(ctx, path.Join(objectsPath, objectID)).
		Execute(nil)
	if err != nil {
		return fmt.Errorf("%w: %w", errDeleteSettings, err)
	}

	return nil
}
