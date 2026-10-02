// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package settings

import (
	"context"
	"path"

	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core"
)

const (
	gen2ObjectsPath         = "/v2/settings/objects"
	gen2EffectiveValuesPath = "/v2/settings/effectiveValues"

	gen2ValidateOnlyParam = "validateOnly"
	gen2PageSizeParam     = "pageSize"
	gen2SchemaIDsParam    = "schemaIds"
	gen2ScopesParam       = "scopes" // used on /v2/settings/objects
	gen2ScopeParam        = "scope"  // used on /v2/settings/effectiveValues
	gen2FilterParam       = "filter"
	gen2FieldsParam       = "fields"
)

// gen2ObjectAPI implements objectAPI against the gen2 (`/v2/settings/...`) settings API.
type gen2ObjectAPI struct {
	apiClient core.Client
}

func newGen2ObjectAPI(apiClient core.Client) objectAPI {
	return &gen2ObjectAPI{apiClient: apiClient}
}

func (g *gen2ObjectAPI) scope(registration K8sClusterRegistration) string {
	return registration.EntityScope
}

func (g *gen2ObjectAPI) createObject(ctx context.Context, schemaID, schemaVersion, scope string, values ...any) ([]string, error) {
	body := make([]objectBody, len(values))
	for i, value := range values {
		body[i] = objectBody{SchemaID: schemaID, SchemaVersion: schemaVersion, Scope: scope, Value: value}
	}

	var response []postObjectsResponse

	err := g.apiClient.POST(ctx, gen2ObjectsPath).
		WithQueryParams(map[string]string{gen2ValidateOnlyParam: "false"}).
		WithJSONBody(body).
		Execute(&response)
	if err != nil {
		return nil, err
	}

	if len(response) != len(values) {
		return nil, notSingleEntryError(len(response))
	}

	ids := make([]string, len(response))
	for i, item := range response {
		ids[i] = item.ObjectID
	}

	return ids, nil
}

func (g *gen2ObjectAPI) listObjects(ctx context.Context, collection listCollection, params listParams, out any) error {
	objectsPath, scopeParam := gen2ObjectsPath, gen2ScopesParam
	if collection == effectiveValuesCollection {
		objectsPath, scopeParam = gen2EffectiveValuesPath, gen2ScopeParam
	}

	query := map[string]string{gen2SchemaIDsParam: params.SchemaID}

	if params.ValidateOnly {
		query[gen2ValidateOnlyParam] = "true"
	}

	// The Smartscape cluster-UID lookup scope has no gen2 equivalent, so gen2 only ever uses Scope
	// when it is a real, already-known scope; an empty Scope falls back to Filter instead.
	if params.Scope != "" {
		query[scopeParam] = params.Scope
	} else if params.Filter != "" {
		query[gen2FilterParam] = params.Filter
	}

	if params.AddFields != "" {
		query[gen2FieldsParam] = params.AddFields
	}

	if params.PageSize != "" {
		query[gen2PageSizeParam] = params.PageSize
	}

	return g.apiClient.GET(ctx, objectsPath).WithQueryParams(query).Execute(out)
}

func (g *gen2ObjectAPI) deleteObject(ctx context.Context, objectID string) error {
	return g.apiClient.DELETE(ctx, path.Join(gen2ObjectsPath, objectID)).Execute(nil)
}
