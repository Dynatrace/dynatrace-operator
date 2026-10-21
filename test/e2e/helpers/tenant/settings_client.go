// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

//go:build e2e

package tenant

import (
	"context"
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
	legacyMetadataEnrichmentSchemaID = "builtin:kubernetes.generic.metadata.enrichment"
	metadataEnrichmentSchemaID       = "builtin:ingest.enrichment.config"

	objectsPath            = dtsettings.ObjectsPath
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

// EnrichmentRuleObject holds the objectId of a single enrichment rule settings object,
// used to identify rules for deletion.
type EnrichmentRuleObject struct {
	ObjectID string `json:"objectId"`
}

type enrichmentRulesObjectsResponse struct {
	Items []EnrichmentRuleObject `json:"items"`
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

func (c *SettingsClient) getEnrichmentRuleObjectsForSchema(ctx context.Context, schemaID, scope string) ([]EnrichmentRuleObject, error) {
	var resp enrichmentRulesObjectsResponse

	err := c.apiClient.GET(ctx, objectsPath).
		WithQueryParams(map[string]string{
			schemaIDsQueryParam: schemaID,
			scopesQueryParam:    scope,
		}).
		Execute(&resp)
	if err != nil {
		return nil, fmt.Errorf("get enrichment rule objects (%s): %w", schemaID, err)
	}

	return resp.Items, nil
}

// GetEnrichmentRuleObjects returns the list of enrichment rule settings objects (with objectIds) for the given scope.
// Does not handle pagination.
func (c *SettingsClient) GetEnrichmentRuleObjects(ctx context.Context, scope string) ([]EnrichmentRuleObject, error) {
	if scope == "" {
		return nil, errors.New("no scope provided for getting enrichment rule objects")
	}

	return c.getEnrichmentRuleObjectsForSchema(ctx, metadataEnrichmentSchemaID, scope)
}

// GetLegacyEnrichmentRuleObjects returns enrichment rule settings objects for the legacy schema (builtin:kubernetes.generic.metadata.enrichment).
// Does not handle pagination.
func (c *SettingsClient) GetLegacyEnrichmentRuleObjects(ctx context.Context, scope string) ([]EnrichmentRuleObject, error) {
	if scope == "" {
		return nil, errors.New("no scope provided for getting legacy enrichment rule objects")
	}

	return c.getEnrichmentRuleObjectsForSchema(ctx, legacyMetadataEnrichmentSchemaID, scope)
}

// CreateEnrichmentRuleObject creates a settings object for the builtin:ingest.enrichment.config schema.
func (c *SettingsClient) CreateEnrichmentRuleObject(ctx context.Context, scope string, rules ...metadataenrichment.Rule) ([]string, error) {
	return c.createEnrichmentRule(ctx, metadataEnrichmentSchemaID, scope, rules)
}

// CreateLegacyEnrichmentRuleObject creates a settings object for the builtin:kubernetes.generic.metadata.enrichment schema.
func (c *SettingsClient) CreateLegacyEnrichmentRuleObject(ctx context.Context, scope string, rules ...metadataenrichment.Rule) ([]string, error) {
	return c.createEnrichmentRule(ctx, legacyMetadataEnrichmentSchemaID, scope, rules)
}

func (c *SettingsClient) createEnrichmentRule(ctx context.Context, schemaID, scope string, rules []metadataenrichment.Rule) ([]string, error) {
	if scope == "" {
		return nil, errors.New("no scope (MEID) was provided for creating the enrichment rule")
	}

	var response []postObjectsResponse

	err := c.apiClient.POST(ctx, objectsPath).
		WithQueryParams(map[string]string{validateOnlyQueryParam: "false"}).
		WithJSONBody(buildEnrichmentBody(schemaID, scope, rules)).
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

func buildEnrichmentBody(schemaID, scope string, rules []metadataenrichment.Rule) any {
	if schemaID == metadataEnrichmentSchemaID {
		values := make([]enrichmentRuleValue, len(rules))
		for i, rule := range rules {
			values[i] = convertRule(rule)
		}

		return newPostObjectsBody(schemaID, "", scope, values...)
	}

	return newPostObjectsBody(schemaID, "", scope, legacyEnrichmentValue{Rules: rules})
}

func convertRule(rule metadataenrichment.Rule) enrichmentRuleValue {
	return enrichmentRuleValue{
		Type:        rule.Type,
		ValueSource: rule.Source,
		Target:      rule.Target,
	}
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
