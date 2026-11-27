// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package settings

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core/middleware"
	coremock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestGetNetworkZoneObject(t *testing.T) {
	ctx := t.Context()

	params := map[string]string{
		schemaIDsQueryParam: networkZonesSchemaID,
		fieldsQueryParam:    "value",
		filterQueryParam:    "value.id='zone-1'",
	}

	t.Run("success", func(t *testing.T) {
		apiClient := coremock.NewClient(t)
		request := coremock.NewRequest(t)
		request.EXPECT().WithQueryParams(params).Return(request).Once()
		request.EXPECT().Execute(new(getNetworkZonesResponse)).Run(injectResponse(getNetworkZonesResponse{
			TotalCount: 1,
			Items: []networkZoneItem{
				{Value: NetworkZoneObject{ID: "zone-1", FallbackMode: "NONE", Description: "my zone", AlternativeZones: []string{"zone-2"}}},
			},
		})).Return(nil).Once()
		apiClient.EXPECT().GET(ctx, ObjectsPath).Return(request).Once()

		client := NewClient(apiClient)
		zone, err := client.GetNetworkZoneObject(ctx, "zone-1")
		require.NoError(t, err)
		assert.Equal(t, NetworkZoneObject{ID: "zone-1", FallbackMode: "NONE", Description: "my zone", AlternativeZones: []string{"zone-2"}}, zone)
	})

	t.Run("error from API", func(t *testing.T) {
		apiClient := coremock.NewClient(t)
		request := coremock.NewRequest(t)
		request.EXPECT().WithQueryParams(params).Return(request).Once()
		request.EXPECT().Execute(new(getNetworkZonesResponse)).Return(errors.New("api error")).Once()
		apiClient.EXPECT().GET(ctx, ObjectsPath).Return(request).Once()

		client := NewClient(apiClient)
		zone, err := client.GetNetworkZoneObject(ctx, "zone-1")
		require.Error(t, err)
		assert.Equal(t, NetworkZoneObject{}, zone)
	})

	t.Run("network zone does not exist", func(t *testing.T) {
		apiClient := coremock.NewClient(t)
		request := coremock.NewRequest(t)
		request.EXPECT().WithQueryParams(params).Return(request).Once()
		request.EXPECT().Execute(new(getNetworkZonesResponse)).Return(nil).Once()
		apiClient.EXPECT().GET(ctx, ObjectsPath).Return(request).Once()

		client := NewClient(apiClient)
		zone, err := client.GetNetworkZoneObject(ctx, "zone-1")
		require.NoError(t, err)
		assert.Equal(t, NetworkZoneObject{}, zone)
	})

	t.Run("empty networkZoneID", func(t *testing.T) {
		apiClient := coremock.NewClient(t)

		client := NewClient(apiClient)
		zone, err := client.GetNetworkZoneObject(ctx, "")
		require.ErrorIs(t, err, errMissingNetworkZoneID)
		assert.Equal(t, NetworkZoneObject{}, zone)
	})

	t.Run("response is cacheable, also when empty", func(t *testing.T) {
		assert.False(t, (&getNetworkZonesResponse{}).IsEmpty())
		assert.False(t, (&getNetworkZonesResponse{Items: []networkZoneItem{{}}}).IsEmpty())
	})
}

func TestGetNetworkZoneObjectCaching(t *testing.T) {
	var calls atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)

		if strings.Contains(r.URL.RawQuery, "zone-1") {
			_, _ = w.Write([]byte(`{"totalCount":1,"items":[{"value":{"id":"zone-1","fallbackMode":"NONE"}}]}`))
		} else {
			_, _ = w.Write([]byte(`{"totalCount":0,"items":[]}`))
		}
	}))
	defer server.Close()

	baseURL, err := url.Parse(server.URL)
	require.NoError(t, err)

	apiClient := core.NewClient(core.Config{
		BaseURL:    baseURL,
		HTTPClient: &http.Client{Transport: middleware.NewCacheRoundTripper(http.DefaultTransport, time.Minute)},
	})
	client := NewClient(apiClient)
	ctx := t.Context()

	// found is cached
	zone, err := client.GetNetworkZoneObject(ctx, "zone-1")
	require.NoError(t, err)
	assert.Equal(t, "zone-1", zone.ID)
	assert.Equal(t, int32(1), calls.Load())

	zone, err = client.GetNetworkZoneObject(ctx, "zone-1")
	require.NoError(t, err)
	assert.Equal(t, "zone-1", zone.ID)
	assert.Equal(t, int32(1), calls.Load(), "non-empty response must be served from cache")

	// not found is cached too
	zone, err = client.GetNetworkZoneObject(ctx, "zone-2")
	require.NoError(t, err)
	assert.Empty(t, zone.ID)
	assert.Equal(t, int32(2), calls.Load())

	zone, err = client.GetNetworkZoneObject(ctx, "zone-2")
	require.NoError(t, err)
	assert.Empty(t, zone.ID)
	assert.Equal(t, int32(2), calls.Load(), "empty response must be served from cache")
}

func TestCreateNetworkZoneObject(t *testing.T) {
	ctx := t.Context()

	params := map[string]string{validateOnlyQueryParam: "false"}

	matchBody := func(expected NetworkZoneObject) any {
		return mock.MatchedBy(func(arg any) bool {
			body, ok := arg.([]postObjectsBody[NetworkZoneObject])

			return ok && len(body) == 1 &&
				body[0].SchemaID == networkZonesSchemaID &&
				body[0].Scope == networkZonesScope &&
				assert.ObjectsAreEqual(expected, body[0].Value)
		})
	}

	newZone := NetworkZoneObject{ID: "zone-1", FallbackMode: "NONE", AlternativeZones: []string{}}

	t.Run("empty network zone ID", func(t *testing.T) {
		apiClient := coremock.NewClient(t)

		client := NewClient(apiClient)
		objectID, err := client.CreateNetworkZoneObject(ctx, NetworkZoneObject{FallbackMode: "NONE"})
		require.ErrorIs(t, err, errMissingNetworkZoneID)
		assert.Empty(t, objectID)
	})

	t.Run("success", func(t *testing.T) {
		apiClient := coremock.NewClient(t)
		request := coremock.NewRequest(t)
		request.EXPECT().WithQueryParams(params).Return(request).Once()
		request.EXPECT().WithJSONBody(matchBody(newZone)).Return(request).Once()
		request.EXPECT().Execute(new([]postObjectsResponse)).Run(injectResponse([]postObjectsResponse{{ObjectID: "obj-123"}})).Return(nil).Once()
		apiClient.EXPECT().POST(ctx, ObjectsPath).Return(request).Once()

		client := NewClient(apiClient)
		objectID, err := client.CreateNetworkZoneObject(ctx, NetworkZoneObject{ID: "zone-1", FallbackMode: "NONE"})
		require.NoError(t, err)
		assert.Equal(t, "obj-123", objectID)
	})

	t.Run("keeps the other fields of an existing zone", func(t *testing.T) {
		existing := NetworkZoneObject{ID: "zone-1", FallbackMode: "NONE", Description: "my zone", AlternativeZones: []string{"zone-2", "zone-3"}}

		apiClient := coremock.NewClient(t)
		request := coremock.NewRequest(t)
		request.EXPECT().WithQueryParams(params).Return(request).Once()
		request.EXPECT().WithJSONBody(matchBody(existing)).Return(request).Once()
		request.EXPECT().Execute(new([]postObjectsResponse)).Run(injectResponse([]postObjectsResponse{{ObjectID: "obj-123"}})).Return(nil).Once()
		apiClient.EXPECT().POST(ctx, ObjectsPath).Return(request).Once()

		client := NewClient(apiClient)
		objectID, err := client.CreateNetworkZoneObject(ctx, existing)
		require.NoError(t, err)
		assert.Equal(t, "obj-123", objectID)
	})

	t.Run("error from API", func(t *testing.T) {
		apiClient := coremock.NewClient(t)
		request := coremock.NewRequest(t)
		request.EXPECT().WithQueryParams(params).Return(request).Once()
		request.EXPECT().WithJSONBody(matchBody(newZone)).Return(request).Once()
		request.EXPECT().Execute(new([]postObjectsResponse)).Return(errors.New("api error")).Once()
		apiClient.EXPECT().POST(ctx, ObjectsPath).Return(request).Once()

		client := NewClient(apiClient)
		objectID, err := client.CreateNetworkZoneObject(ctx, NetworkZoneObject{ID: "zone-1", FallbackMode: "NONE"})
		require.Error(t, err)
		assert.Empty(t, objectID)
	})

	t.Run("response not exactly one entry", func(t *testing.T) {
		apiClient := coremock.NewClient(t)
		request := coremock.NewRequest(t)
		request.EXPECT().WithQueryParams(params).Return(request).Once()
		request.EXPECT().WithJSONBody(matchBody(newZone)).Return(request).Once()
		request.EXPECT().Execute(new([]postObjectsResponse)).Return(nil).Once()
		apiClient.EXPECT().POST(ctx, ObjectsPath).Return(request).Once()

		client := NewClient(apiClient)
		objectID, err := client.CreateNetworkZoneObject(ctx, NetworkZoneObject{ID: "zone-1", FallbackMode: "NONE"})
		require.ErrorAs(t, err, new(notSingleEntryError))
		assert.Empty(t, objectID)
	})
}

func TestNetworkZoneObjectJSON(t *testing.T) {
	t.Run("empty description is omitted, alternative zones are an empty list", func(t *testing.T) {
		body, err := json.Marshal(NetworkZoneObject{ID: "zone-1", FallbackMode: "NONE", AlternativeZones: []string{}})
		require.NoError(t, err)
		assert.JSONEq(t, `{"id":"zone-1","fallbackMode":"NONE","alternativeZones":[]}`, string(body))
	})

	t.Run("all fields", func(t *testing.T) {
		body, err := json.Marshal(NetworkZoneObject{ID: "zone-1", FallbackMode: "ANY_ACTIVE_GATE", Description: "my zone", AlternativeZones: []string{"zone-2"}})
		require.NoError(t, err)
		assert.JSONEq(t, `{"id":"zone-1","fallbackMode":"ANY_ACTIVE_GATE","description":"my zone","alternativeZones":["zone-2"]}`, string(body))
	})
}
