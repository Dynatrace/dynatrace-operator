package hostevent

import (
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core"
	coremock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetEntityIDForIP(t *testing.T) {
	setupClient := func(t *testing.T, err error) *Client {
		req := coremock.NewAPIRequest(t)
		req.EXPECT().
			WithQueryParams(map[string]string{
				"relativeTime":   "30mins",
				"includeDetails": "false",
			}).
			Return(req).Once()
		req.EXPECT().
			Execute(new([]hostsResponse)).
			Run(func(model any) {
				if err == nil {
					resp := model.(*[]hostsResponse)
					*resp = []hostsResponse{
						{EntityID: "HOST-42", IPAddresses: []string{"1.1.1.1"}},
					}
				}
			}).
			Return(err).Once()
		client := coremock.NewAPIClient(t)
		client.EXPECT().GET(t.Context(), hostsPath).Return(req).Once()

		return NewClient(client, "")
	}

	t.Run("found", func(t *testing.T) {
		client := setupClient(t, nil)
		entityID, err := client.GetEntityIDForIP(t.Context(), "1.1.1.1")
		require.NoError(t, err)
		assert.Equal(t, "HOST-42", entityID)
	})

	t.Run("not found", func(t *testing.T) {
		client := setupClient(t, nil)
		_, err := client.GetEntityIDForIP(t.Context(), "1.1.1.2")
		require.ErrorAs(t, err, new(EntityNotFoundError))
	})

	t.Run("api error 404", func(t *testing.T) {
		client := setupClient(t, &core.HTTPError{StatusCode: 404, Message: "nope"})
		_, err := client.GetEntityIDForIP(t.Context(), "1.1.1.1")
		require.True(t, core.IsNotFound(err))
		assert.Equal(t, hostsPath+" is not available on the tenant", err.(*core.HTTPError).Message)
	})

	t.Run("api error generic", func(t *testing.T) {
		expectErr := &core.HTTPError{StatusCode: 418, Message: "teapot"}
		client := setupClient(t, expectErr)
		_, err := client.GetEntityIDForIP(t.Context(), "1.1.1.1")
		require.ErrorIs(t, err, expectErr)
	})
}

func Test_buildHostEntityMap(t *testing.T) {
	hosts := []hostsResponse{
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
		hosts       []hostsResponse
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
			got := buildHostEntityMap(tt.hosts, tt.networkZone)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSendEvent(t *testing.T) {
	setupClient := func(t *testing.T, err error) *Client {
		req := coremock.NewAPIRequest(t)
		req.EXPECT().WithJSONBody(Event{EventType: "TEST"}).Return(req).Once()
		req.EXPECT().Execute(nil).Return(err).Once()
		client := coremock.NewAPIClient(t)
		client.EXPECT().POST(t.Context(), eventsPath).Return(req).Once()

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
		assert.Equal(t, eventsPath+" is not available on the tenant", err.(*core.HTTPError).Message)
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
	}, NewMarkForTerminationEvent("foo", "bar", "baz", timestamp))
}
