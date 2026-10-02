// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package settings

import (
	"context"
	"path"

	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core"
)

const (
	gen3ObjectsPath         = "/objects"
	gen3EffectiveValuesPath = "/effective-values"

	gen3ValidateOnlyParam             = "validate-only"
	gen3SchemaIDParam                 = "schema-id"
	gen3ScopeParam                    = "scope"
	gen3AddFieldsParam                = "add-fields"
	gen3OptimisticLockingVersionParam = "optimistic-locking-version"
)

// gen3ObjectAPI implements objectAPI against the gen3 (`/platform/settings/v1/...`) settings API.
type gen3ObjectAPI struct {
	apiClient core.Client
}

func newGen3ObjectAPI(apiClient core.Client) objectAPI {
	return &gen3ObjectAPI{apiClient: apiClient}
}

// scope reuses the real Monitored Entity scope once the Kubernetes connection setting has resolved
// one. Until then, gen3 has no unscoped writes, so it falls back to the Smartscape cluster-UID lookup
// scope, which Dynatrace resolves to the real scope once the setting is persisted.
func (g *gen3ObjectAPI) scope(registration K8sClusterRegistration) string {
	if registration.EntityScope != "" {
		return registration.EntityScope
	}

	return k8sClusterUIDScope(registration.EntityID)
}

// createObject creates one object per value: the gen3 API only ever accepts a single object per
// request, unlike gen2 which accepts a batch.
func (g *gen3ObjectAPI) createObject(ctx context.Context, schemaID, schemaVersion, scope string, values ...any) ([]string, error) {
	ids := make([]string, 0, len(values))

	for _, value := range values {
		body := objectBody{SchemaID: schemaID, SchemaVersion: schemaVersion, Scope: scope, Value: value}

		var response postObjectsResponse

		err := g.apiClient.POST(ctx, gen3ObjectsPath).
			WithQueryParams(map[string]string{gen3ValidateOnlyParam: "false"}).
			WithJSONBody(body).
			Execute(&response)
		if err != nil {
			return nil, err
		}

		ids = append(ids, response.ObjectID)
	}

	return ids, nil
}

func (g *gen3ObjectAPI) listObjects(ctx context.Context, collection listCollection, params listParams, out any) error {
	objectsPath := gen3ObjectsPath
	if collection == effectiveValuesCollection {
		objectsPath = gen3EffectiveValuesPath
	}

	query := map[string]string{gen3SchemaIDParam: params.SchemaID}

	if params.Scope != "" {
		query[gen3ScopeParam] = params.Scope
	}

	if params.AddFields != "" {
		query[gen3AddFieldsParam] = params.AddFields
	}

	return g.apiClient.GET(ctx, objectsPath).WithQueryParams(query).Execute(out)
}

type settingsObjectVersion struct {
	Version string `json:"version"`
}

// deleteObject deletes the settings object with the given ID. Unlike gen2, deletion requires the
// current optimistic-locking-version of the object, so it is fetched first.
func (g *gen3ObjectAPI) deleteObject(ctx context.Context, objectID string) error {
	var obj settingsObjectVersion

	if err := g.apiClient.GET(ctx, path.Join(gen3ObjectsPath, objectID)).Execute(&obj); err != nil {
		return err
	}

	return g.apiClient.DELETE(ctx, path.Join(gen3ObjectsPath, objectID)).
		WithQueryParams(map[string]string{gen3OptimisticLockingVersionParam: obj.Version}).
		Execute(nil)
}
