// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package settings

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
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
				{Value: NetworkZoneObject{ID: "zone-1", FallbackMode: "NONE"}},
			},
		})).Return(nil).Once()
		apiClient.EXPECT().GET(ctx, ObjectsPath).Return(request).Once()

		client := NewClient(apiClient)
		zone, err := client.GetNetworkZoneObject(ctx, "zone-1")
		require.NoError(t, err)
		assert.Equal(t, NetworkZoneObject{ID: "zone-1", FallbackMode: "NONE"}, zone)
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

	t.Run("IsEmpty", func(t *testing.T) {
		assert.True(t, (&getNetworkZonesResponse{}).IsEmpty())
		assert.False(t, (&getNetworkZonesResponse{Items: []networkZoneItem{{}}}).IsEmpty())
	})
}

func TestGetNetworkZoneObjectCaching(t *testing.T) {
	calls := 0
	exists := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++

		if exists {
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

	// not found is not cached
	zone, err := client.GetNetworkZoneObject(ctx, "zone-1")
	require.NoError(t, err)
	assert.Empty(t, zone.ID)
	assert.Equal(t, 1, calls)

	zone, err = client.GetNetworkZoneObject(ctx, "zone-1")
	require.NoError(t, err)
	assert.Empty(t, zone.ID)
	assert.Equal(t, 2, calls, "empty response must not be cached")

	// found is cached
	exists = true

	zone, err = client.GetNetworkZoneObject(ctx, "zone-1")
	require.NoError(t, err)
	assert.Equal(t, "zone-1", zone.ID)
	assert.Equal(t, 3, calls)

	zone, err = client.GetNetworkZoneObject(ctx, "zone-1")
	require.NoError(t, err)
	assert.Equal(t, "zone-1", zone.ID)
	assert.Equal(t, 3, calls, "non-empty response must be served from cache")
}

func TestCreateNetworkZoneObject(t *testing.T) {
	ctx := t.Context()

	params := map[string]string{validateOnlyQueryParam: "false"}

	matchBody := mock.MatchedBy(func(arg any) bool {
		body, ok := arg.([]postObjectsBody[NetworkZoneObject])

		return ok && len(body) == 1 &&
			body[0].SchemaID == networkZonesSchemaID &&
			body[0].Scope == networkZonesScope &&
			body[0].Value == NetworkZoneObject{ID: "zone-1", FallbackMode: "NONE"}
	})

	t.Run("empty networkZoneID", func(t *testing.T) {
		apiClient := coremock.NewClient(t)

		client := NewClient(apiClient)
		objectID, err := client.CreateNetworkZoneObject(ctx, "", "NONE")
		require.ErrorIs(t, err, errMissingNetworkZoneID)
		assert.Empty(t, objectID)
	})

	t.Run("success", func(t *testing.T) {
		apiClient := coremock.NewClient(t)
		request := coremock.NewRequest(t)
		request.EXPECT().WithQueryParams(params).Return(request).Once()
		request.EXPECT().WithJSONBody(matchBody).Return(request).Once()
		request.EXPECT().Execute(new([]postObjectsResponse)).Run(injectResponse([]postObjectsResponse{{ObjectID: "obj-123"}})).Return(nil).Once()
		apiClient.EXPECT().POST(ctx, ObjectsPath).Return(request).Once()

		client := NewClient(apiClient)
		objectID, err := client.CreateNetworkZoneObject(ctx, "zone-1", "NONE")
		require.NoError(t, err)
		assert.Equal(t, "obj-123", objectID)
	})

	t.Run("error from API", func(t *testing.T) {
		apiClient := coremock.NewClient(t)
		request := coremock.NewRequest(t)
		request.EXPECT().WithQueryParams(params).Return(request).Once()
		request.EXPECT().WithJSONBody(matchBody).Return(request).Once()
		request.EXPECT().Execute(new([]postObjectsResponse)).Return(errors.New("api error")).Once()
		apiClient.EXPECT().POST(ctx, ObjectsPath).Return(request).Once()

		client := NewClient(apiClient)
		objectID, err := client.CreateNetworkZoneObject(ctx, "zone-1", "NONE")
		require.Error(t, err)
		assert.Empty(t, objectID)
	})

	t.Run("response not exactly one entry", func(t *testing.T) {
		apiClient := coremock.NewClient(t)
		request := coremock.NewRequest(t)
		request.EXPECT().WithQueryParams(params).Return(request).Once()
		request.EXPECT().WithJSONBody(matchBody).Return(request).Once()
		request.EXPECT().Execute(new([]postObjectsResponse)).Return(nil).Once()
		apiClient.EXPECT().POST(ctx, ObjectsPath).Return(request).Once()

		client := NewClient(apiClient)
		objectID, err := client.CreateNetworkZoneObject(ctx, "zone-1", "NONE")
		require.ErrorAs(t, err, new(notSingleEntryError))
		assert.Empty(t, objectID)
	})
}
