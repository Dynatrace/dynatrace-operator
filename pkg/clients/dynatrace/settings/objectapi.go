// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package settings

import (
	"context"
	"fmt"
)

// objectAPI is the minimal set of settings operations whose wire format differs between the gen2 and
// gen3 settings APIs: paths, query-parameter casing, and whether a request/response carries a single
// object or a list of them. Every higher-level settings.Client method below is implemented once
// against this interface, so adding a generation means implementing objectAPI, not the whole Client.
type objectAPI interface {
	// createObject creates or updates one or more settings objects under the same schema/scope and
	// returns their objectIds, in the order the values were given. How many underlying HTTP requests
	// this takes is up to the implementation: gen2 batches every value into a single request, gen3
	// issues one request per value since it only ever creates a single object per call.
	createObject(ctx context.Context, schemaID, schemaVersion, scope string, values ...any) ([]string, error)

	// listObjects executes a GET against the given collection and decodes the response into out.
	listObjects(ctx context.Context, collection listCollection, params listParams, out any) error

	// deleteObject deletes the settings object with the given ID.
	deleteObject(ctx context.Context, objectID string) error

	// scope returns the scope to address settings calls for this cluster with. Both generations use
	// EntityScope directly once it is known, which may still be empty (e.g. before the Kubernetes
	// connection setting has been created) - gen2 callers fall back to a Filter in that case. Gen3 has
	// no unscoped writes, so until EntityScope is known it instead derives a scope from EntityID via the
	// Smartscape cluster-UID lookup, which Dynatrace resolves to the real Monitored Entity scope.
	scope(registration K8sClusterRegistration) string
}

// listCollection selects which settings collection a listObjects call addresses.
type listCollection int

const (
	objectsCollection listCollection = iota
	effectiveValuesCollection
)

// listParams are the logical parameters a listObjects call may need. Not every generation uses every
// field, and not every call sets every field - e.g. PageSize is only ever set by GetK8sClusterME, and
// Filter only matters when Scope is empty: the Smartscape cluster-UID lookup has no gen2 equivalent, so
// gen2 falls back to filtering across every object of the schema instead of scoping to one.
type listParams struct {
	SchemaID     string
	Scope        string
	Filter       string
	AddFields    string
	PageSize     string
	ValidateOnly bool
}

// objectBody is the wire shape of a single settings object in a create/update request.
type objectBody struct {
	SchemaID      string `json:"schemaId"`
	SchemaVersion string `json:"schemaVersion,omitempty"`
	Scope         string `json:"scope,omitempty"`
	Value         any    `json:"value"`
}

type postObjectsResponse struct {
	ObjectID string `json:"objectId"`
}

// singleID extracts the one expected objectId from a createObject call.
// We only create (post) settings if they do not exist yet, so receiving back not exactly one object is
// a cause for alarm.
func singleID(ids []string) (string, error) {
	if len(ids) != 1 {
		return "", notSingleEntryError(len(ids))
	}

	return ids[0], nil
}

type notSingleEntryError int

func (num notSingleEntryError) Error() string {
	return fmt.Sprintf("response is not containing exactly one entry, got %d entries", int(num))
}
