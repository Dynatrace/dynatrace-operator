// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package hostevent

import (
	"context"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core"
	coremock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var anyCtx = mock.MatchedBy(func(context.Context) bool { return true })

func newEntity(id string, lastSeen int64, zone string, ips ...string) hostEntity {
	e := hostEntity{EntityID: id, LastSeenTms: lastSeen}
	e.Properties.NetworkZone = zone
	e.Properties.IPAddress = ips

	return e
}

func expectSelectorRequest(t *testing.T, coreClient *coremock.Client, ip string, entities []hostEntity, err error) {
	req := coremock.NewRequest(t)
	req.EXPECT().
		WithQueryParams(map[string]string{
			"entitySelector": `type(HOST),ipAddress("` + ip + `")`,
			"from":           "now-30m",
			"fields":         "+lastSeenTms,+properties.networkZone,+properties.ipAddress",
		}).
		Return(req).Once()
	req.EXPECT().
		Execute(new(entitiesResponse)).
		Run(func(model any) {
			if err == nil {
				model.(*entitiesResponse).Entities = entities
			}
		}).
		Return(err).Once()
	coreClient.EXPECT().GET(anyCtx, entitiesPath).Return(req).Once()
}

func TestGetEntityIDForIPBySelector(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		coreClient := coremock.NewClient(t)
		expectSelectorRequest(t, coreClient, "1.1.1.1", []hostEntity{newEntity("HOST-42", 1, "", "1.1.1.1")}, nil)
		entityID, err := NewClient(coreClient, "").GetEntityIDForIP(t.Context(), "1.1.1.1")
		require.NoError(t, err)
		assert.Equal(t, "HOST-42", entityID)
	})

	t.Run("not found", func(t *testing.T) {
		coreClient := coremock.NewClient(t)
		expectSelectorRequest(t, coreClient, "1.1.1.2", nil, nil)
		_, err := NewClient(coreClient, "").GetEntityIDForIP(t.Context(), "1.1.1.2")
		require.ErrorAs(t, err, new(EntityNotFoundError))
	})

	t.Run("reused IP picks most recently seen host", func(t *testing.T) {
		coreClient := coremock.NewClient(t)
		expectSelectorRequest(t, coreClient, "1.1.1.1", []hostEntity{
			newEntity("HOST-NEW", 200, "zone-a", "1.1.1.1"),
			newEntity("HOST-OLD", 100, "zone-a", "1.1.1.1"),
		}, nil)
		entityID, err := NewClient(coreClient, "zone-a").GetEntityIDForIP(t.Context(), "1.1.1.1")
		require.NoError(t, err)
		assert.Equal(t, "HOST-NEW", entityID)
	})

	t.Run("network zone filter still applies", func(t *testing.T) {
		coreClient := coremock.NewClient(t)
		expectSelectorRequest(t, coreClient, "1.1.1.1", []hostEntity{
			newEntity("HOST-OTHER-ZONE", 200, "zone-b", "1.1.1.1"),
			newEntity("HOST-MY-ZONE", 100, "zone-a", "1.1.1.1"),
		}, nil)
		entityID, err := NewClient(coreClient, "zone-a").GetEntityIDForIP(t.Context(), "1.1.1.1")
		require.NoError(t, err)
		assert.Equal(t, "HOST-MY-ZONE", entityID)
	})

	t.Run("api error 404", func(t *testing.T) {
		coreClient := coremock.NewClient(t)
		expectSelectorRequest(t, coreClient, "1.1.1.1", nil, &core.HTTPError{StatusCode: 404, Message: "nope"})
		_, err := NewClient(coreClient, "").GetEntityIDForIP(t.Context(), "1.1.1.1")
		require.True(t, core.IsNotFound(err))
		assert.EqualError(t, err, entitiesPath+" is not available on the tenant")
	})

	t.Run("api error generic is not swallowed", func(t *testing.T) {
		expectErr := &core.HTTPError{StatusCode: 418, Message: "teapot"}
		coreClient := coremock.NewClient(t)
		expectSelectorRequest(t, coreClient, "1.1.1.1", nil, expectErr)
		_, err := NewClient(coreClient, "").GetEntityIDForIP(t.Context(), "1.1.1.1")
		require.ErrorIs(t, err, expectErr)
	})
}

// TestGetEntityIDForIP covers the v1 fallback, taken when the token gets 403 on /v2/entities.
func TestGetEntityIDForIP(t *testing.T) {
	setupClient := func(t *testing.T, err error) *ClientImpl {
		coreClient := coremock.NewClient(t)
		expectSelectorRequest(t, coreClient, "1.1.1.1", nil, &core.HTTPError{StatusCode: 403, Message: "missing entities.read"})

		req := coremock.NewRequest(t)
		req.EXPECT().
			WithQueryParams(map[string]string{
				"relativeTime":   "30mins",
				"includeDetails": "false",
				"pageSize":       "4000",
			}).
			Return(req).Once()
		req.EXPECT().
			Execute(new([]HostResponse)).
			Run(func(model any) {
				if err == nil {
					resp := model.(*[]HostResponse)
					*resp = []HostResponse{
						{EntityID: "HOST-42", IPAddresses: []string{"1.1.1.1"}},
					}
				}
			}).
			Return(err).Once()
		coreClient.EXPECT().GET(anyCtx, hostsPath).Return(req).Once()

		return NewClient(coreClient, "")
	}

	t.Run("found", func(t *testing.T) {
		coreClient := setupClient(t, nil)
		entityID, err := coreClient.GetEntityIDForIP(t.Context(), "1.1.1.1")
		require.NoError(t, err)
		assert.Equal(t, "HOST-42", entityID)
	})

	t.Run("not found", func(t *testing.T) {
		coreClient := coremock.NewClient(t)
		expectSelectorRequest(t, coreClient, "1.1.1.2", nil, &core.HTTPError{StatusCode: 403, Message: "missing entities.read"})

		req := coremock.NewRequest(t)
		req.EXPECT().WithQueryParams(mock.Anything).Return(req).Once()
		req.EXPECT().Execute(new([]HostResponse)).Run(func(model any) {
			*model.(*[]HostResponse) = []HostResponse{{EntityID: "HOST-42", IPAddresses: []string{"1.1.1.1"}}}
		}).Return(nil).Once()
		coreClient.EXPECT().GET(anyCtx, hostsPath).Return(req).Once()

		_, err := NewClient(coreClient, "").GetEntityIDForIP(t.Context(), "1.1.1.2")
		require.ErrorAs(t, err, new(EntityNotFoundError))
	})

	t.Run("no ip provided", func(t *testing.T) {
		// deliberately not setupClient: no request is expected to be sent at all
		coreClient := NewClient(coremock.NewClient(t), "")
		_, err := coreClient.GetEntityIDForIP(t.Context(), "")
		require.EqualError(t, err, "must provide IP")
	})

	t.Run("api error 404", func(t *testing.T) {
		coreClient := setupClient(t, &core.HTTPError{StatusCode: 404, Message: "nope"})
		_, err := coreClient.GetEntityIDForIP(t.Context(), "1.1.1.1")
		require.True(t, core.IsNotFound(err))
		assert.EqualError(t, err, hostsPath+" is not available on the tenant")
	})

	t.Run("api error generic", func(t *testing.T) {
		expectErr := &core.HTTPError{StatusCode: 418, Message: "teapot"}
		coreClient := setupClient(t, expectErr)
		_, err := coreClient.GetEntityIDForIP(t.Context(), "1.1.1.1")
		require.ErrorIs(t, err, expectErr)
	})
}

func Test_buildHostEntityMap(t *testing.T) {
	hosts := []HostResponse{
		{
			EntityID:      "HOST-1",
			NetworkZoneID: "default",
			IPAddresses:   []string{"1.1.1.1"},
		},
		{
			EntityID:    "HOST-2",
			IPAddresses: []string{"1.1.1.2"},
		},
		{
			EntityID:      "HOST-3",
			NetworkZoneID: "other",
			IPAddresses:   []string{"1.1.1.3"},
		},
	}

	tests := []struct {
		name        string
		hosts       []HostResponse
		networkZone string
		want        hostEntityMap
	}{
		{
			name: "no hosts",
			want: make(hostEntityMap),
		},
		{
			name:  "match unset network zone",
			hosts: hosts,
			want: hostEntityMap{
				"1.1.1.1": "HOST-1",
				"1.1.1.2": "HOST-2",
			},
		},
		{
			name:        "match default network zone",
			hosts:       hosts,
			networkZone: "default",
			want: hostEntityMap{
				"1.1.1.1": "HOST-1",
			},
		},
		{
			name:        "match other network zone",
			hosts:       hosts,
			networkZone: "other",
			want: hostEntityMap{
				"1.1.1.3": "HOST-3",
			},
		},
		{
			name:        "match network zone case-insensitively",
			hosts:       hosts,
			networkZone: "OTHER",
			want: hostEntityMap{
				"1.1.1.3": "HOST-3",
			},
		},
		{
			name:  "no matches without network zone",
			hosts: hosts[2:],
			want:  make(hostEntityMap),
		},
		{
			name:        "no matches with network zone",
			hosts:       hosts[:2],
			networkZone: "other",
			want:        make(hostEntityMap),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildHostEntityMap(t.Context(), tt.hosts, tt.networkZone)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSendEvent(t *testing.T) {
	setupClient := func(t *testing.T, err error) *ClientImpl {
		req := coremock.NewRequest(t)
		req.EXPECT().WithJSONBody(Event{EventType: "TEST"}).Return(req).Once()
		req.EXPECT().Execute(nil).Return(err).Once()
		client := coremock.NewClient(t)
		client.EXPECT().POST(anyCtx, eventsPath).Return(req).Once()

		return NewClient(client, "")
	}

	t.Run("no entity type", func(t *testing.T) {
		client := NewClient(nil, "")
		err := client.SendEvent(t.Context(), Event{})
		require.Error(t, err)
	})

	t.Run("success", func(t *testing.T) {
		client := setupClient(t, nil)
		err := client.SendEvent(t.Context(), Event{EventType: "TEST"})
		require.NoError(t, err)
	})

	t.Run("api error 404", func(t *testing.T) {
		client := setupClient(t, &core.HTTPError{StatusCode: 404, Message: "nope"})
		err := client.SendEvent(t.Context(), Event{EventType: "TEST"})
		require.True(t, core.IsNotFound(err))
		assert.EqualError(t, err, eventsPath+" is not available on the tenant")
	})

	t.Run("api error generic", func(t *testing.T) {
		expectErr := &core.HTTPError{StatusCode: 418, Message: "teapot"}
		client := setupClient(t, expectErr)
		err := client.SendEvent(t.Context(), Event{EventType: "TEST"})
		require.ErrorIs(t, err, expectErr)
	})
}

func TestNewMarkForTerminationEvent(t *testing.T) {
	timestamp := time.Date(2026, time.February, 24, 17, 0, 0, 0, time.UTC)
	assert.Equal(t, Event{
		EventType:   MarkedForTerminationEvent,
		Description: "baz",
		Source:      "bar",
		AttachRules: EventAttachRules{
			EntityIDs: []string{"foo"},
		},
		StartInMillis: 1771952400000,
		EndInMillis:   1771952400000,
	}, NewMarkedForTerminationEvent("foo", "bar", "baz", timestamp))
}
