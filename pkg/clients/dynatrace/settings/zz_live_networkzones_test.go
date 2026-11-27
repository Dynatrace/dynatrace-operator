//go:build live

// TEMPORARY: live check against a real tenant. Do not commit.
package settings_test

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core/middleware"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/settings"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// DT_LIVE_API_URL: tenant API base, e.g. https://<tenant>/api ; DT_LIVE_API_TOKEN: token with settings.read + settings.write
func TestLiveNetworkZones(t *testing.T) {
	apiURL, token := os.Getenv("DT_LIVE_API_URL"), os.Getenv("DT_LIVE_API_TOKEN")
	if apiURL == "" || token == "" {
		t.Skip("DT_LIVE_API_URL / DT_LIVE_API_TOKEN not set")
	}

	baseURL, err := url.Parse(apiURL)
	require.NoError(t, err)

	apiClient := core.NewClient(core.Config{
		BaseURL:    baseURL,
		HTTPClient: &http.Client{Transport: middleware.NewCacheRoundTripper(http.DefaultTransport, 15*time.Minute)},
		APIToken:   token,
	})
	client := settings.NewClient(apiClient)
	ctx := t.Context()

	zoneID := fmt.Sprintf("operator-live-test-%d", time.Now().Unix())

	zone, err := client.GetNetworkZoneObject(ctx, zoneID)
	require.NoError(t, err)
	assert.Empty(t, zone.ID, "zone must not exist yet")

	objectID, err := client.CreateNetworkZoneObject(ctx, settings.NetworkZoneObject{ID: zoneID, FallbackMode: "NONE"})
	require.NoError(t, err)
	require.NotEmpty(t, objectID)
	t.Logf("created object, id prefix %.12s...", objectID)

	t.Cleanup(func() {
		// t.Context() is already canceled when Cleanup runs
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		err := apiClient.DELETE(cleanupCtx, settings.ObjectsPath+"/"+url.PathEscape(objectID)).Execute(nil)
		t.Logf("cleanup delete error: %v", err)
	})

	// per ticket/KOD all responses are cached, so the cached client still sees "not found"
	zone, err = client.GetNetworkZoneObject(ctx, zoneID)
	require.NoError(t, err)
	assert.Empty(t, zone.ID, "empty response is served from cache")

	// uncached client (ttl 0) must see the created zone
	uncached := settings.NewClient(core.NewClient(core.Config{
		BaseURL:    baseURL,
		HTTPClient: &http.Client{Transport: middleware.NewCacheRoundTripper(http.DefaultTransport, 0)},
		APIToken:   token,
	}))

	zone, err = uncached.GetNetworkZoneObject(ctx, zoneID)
	require.NoError(t, err)
	assert.Equal(t, zoneID, zone.ID)
	assert.Equal(t, "NONE", zone.FallbackMode)
}

type bearerTransport struct {
	next  http.RoundTripper
	token string
}

func (b bearerTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	r.Header.Set("Authorization", "Bearer "+b.token)

	return b.next.RoundTrip(r)
}

// Exploratory: logs what the API does, asserts nothing about the open questions.
// Env: DT_LIVE_API_URL, DT_LIVE_API_TOKEN (classic, settings.read + settings.write).
// Optional: DT_LIVE_PLATFORM_TOKEN (sent as Bearer) for the platform token check.
func TestLiveNetworkZonesExploratory(t *testing.T) {
	apiURL, token := os.Getenv("DT_LIVE_API_URL"), os.Getenv("DT_LIVE_API_TOKEN")
	if apiURL == "" || token == "" {
		t.Skip("DT_LIVE_API_URL / DT_LIVE_API_TOKEN not set")
	}

	baseURL, err := url.Parse(apiURL)
	require.NoError(t, err)

	newAPIClient := func(rt http.RoundTripper, apiToken string) core.Client {
		return core.NewClient(core.Config{
			BaseURL:    baseURL,
			HTTPClient: &http.Client{Transport: middleware.NewCacheRoundTripper(rt, 0)}, // ttl 0: no caching
			APIToken:   apiToken,
		})
	}

	apiClient := newAPIClient(http.DefaultTransport, token)
	client := settings.NewClient(apiClient)
	ctx := t.Context()
	ts := time.Now().Unix()

	created := map[string]struct{}{}

	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		for id := range created {
			err := apiClient.DELETE(cleanupCtx, settings.ObjectsPath+"/"+url.PathEscape(id)).Execute(nil)
			t.Logf("cleanup delete %.12s...: err=%v", id, err)
		}
	})

	create := func(zoneID, mode string) (string, error) {
		id, err := client.CreateNetworkZoneObject(ctx, settings.NetworkZoneObject{ID: zoneID, FallbackMode: mode})
		if err == nil && id != "" {
			created[id] = struct{}{}
		}

		return id, err
	}

	t.Run("second create with existing id", func(t *testing.T) {
		zoneID := fmt.Sprintf("operator-live-dup-%d", ts)

		id1, err := create(zoneID, "NONE")
		t.Logf("1st create: id=%.12s... err=%v", id1, err)
		require.NoError(t, err)

		id2, err := create(zoneID, "NONE")
		t.Logf("2nd create (same id, same mode): id=%.12s... same objectId=%t err=%v", id2, id1 == id2, err)

		zone, err := client.GetNetworkZoneObject(ctx, zoneID)
		t.Logf("get after 2nd create: %+v err=%v", zone, err)

		// different mode on same id: does it replace?
		id3, err := create(zoneID, "ANY_ACTIVE_GATE")
		t.Logf("3rd create (same id, other mode ANY_ACTIVE_GATE): id=%.12s... err=%v", id3, err)

		zone, err = client.GetNetworkZoneObject(ctx, zoneID)
		t.Logf("get after 3rd create: %+v err=%v", zone, err)
	})

	t.Run("lowercase fallbackMode", func(t *testing.T) {
		for _, mode := range []string{"none", "default", "any"} {
			zoneID := fmt.Sprintf("operator-live-mode-%s-%d", mode, ts)
			id, err := create(zoneID, mode)
			t.Logf("create mode=%q: id=%.12s... err=%v", mode, id, err)
		}
	})

	t.Run("mixed-case id is rejected", func(t *testing.T) {
		zoneID := fmt.Sprintf("Operator-Live-CASE-%d", ts)

		_, err := create(zoneID, "NONE")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "lowercase")

		zone, err := client.GetNetworkZoneObject(ctx, zoneID)
		t.Logf("get mixed-case id: found=%t err=%v", zone.ID != "", err)
	})

	t.Run("replace keeps description and alternative zones", func(t *testing.T) {
		altID := fmt.Sprintf("operator-live-alt-%d", ts)
		zoneID := fmt.Sprintf("operator-live-keep-%d", ts)

		_, err := create(altID, "NONE")
		require.NoError(t, err)

		id, err := client.CreateNetworkZoneObject(ctx, settings.NetworkZoneObject{
			ID: zoneID, FallbackMode: "NONE", Description: "live test", AlternativeZones: []string{altID},
		})
		t.Logf("create with description + alternative zone: id=%.12s... err=%v", id, err)
		require.NoError(t, err)

		created[id] = struct{}{}

		existing, err := client.GetNetworkZoneObject(ctx, zoneID)
		t.Logf("get: %+v err=%v", existing, err)
		require.NoError(t, err)
		assert.Equal(t, "live test", existing.Description)
		assert.Equal(t, []string{altID}, existing.AlternativeZones)

		// what the reconciler would do on fallbackMode drift: copy the zone, change the mode
		existing.FallbackMode = "ANY_ACTIVE_GATE"
		id2, err := client.CreateNetworkZoneObject(ctx, existing)
		t.Logf("replace with Get result: id=%.12s... same objectId=%t err=%v", id2, id == id2, err)
		require.NoError(t, err)

		after, err := client.GetNetworkZoneObject(ctx, zoneID)
		t.Logf("get after replace: %+v err=%v", after, err)
		require.NoError(t, err)
		assert.Equal(t, existing, after, "replace with the Get result must keep description and alternative zones")

		// control: replace with only id + mode, shows whether the other fields get wiped
		id3, err := client.CreateNetworkZoneObject(ctx, settings.NetworkZoneObject{ID: zoneID, FallbackMode: "NONE"})
		t.Logf("control replace with id + mode only: id=%.12s... err=%v", id3, err)

		control, err := client.GetNetworkZoneObject(ctx, zoneID)
		t.Logf("get after control replace: %+v err=%v", control, err)
	})

	t.Run("quote in id", func(t *testing.T) {
		zone, err := client.GetNetworkZoneObject(ctx, "a'b")
		t.Logf("get id with quote: %+v err=%v", zone, err)
	})

	t.Run("platform token", func(t *testing.T) {
		platformToken := os.Getenv("DT_LIVE_PLATFORM_TOKEN")
		if platformToken == "" {
			t.Skip("DT_LIVE_PLATFORM_TOKEN not set")
		}

		pc := settings.NewClient(newAPIClient(bearerTransport{next: http.DefaultTransport, token: platformToken}, ""))
		zone, err := pc.GetNetworkZoneObject(ctx, fmt.Sprintf("operator-live-dup-%d", ts))
		t.Logf("get with platform token: %+v err=%v", zone, err)

		zoneID := fmt.Sprintf("operator-live-platform-%d", ts)
		id, err := pc.CreateNetworkZoneObject(ctx, settings.NetworkZoneObject{ID: zoneID, FallbackMode: "NONE"})
		t.Logf("create with platform token: id=%.12s... err=%v", id, err)

		if err == nil && id != "" {
			created[id] = struct{}{}
		}
	})
}
