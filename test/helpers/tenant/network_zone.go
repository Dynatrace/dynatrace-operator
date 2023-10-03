package tenant

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

type FallbackMode string

const (
	FallbackAnyActiveGate FallbackMode = "ANY_ACTIVE_GATE"
	FallbackNone                       = "NONE"
	FallbackDefaultZone                = "ONLY_DEFAULT_ZONE"
)

type networkZoneRequestBody struct {
	AlternativeZones []string `json:"alternativeZones,omitempty"`
	FallbackMode     string   `json:"fallbackMode,omitempty"`
	Id               string   `json:"id,omitempty"`
}

func CreateNetworkZone(secret Secret, networkZone string, alternativeZones []string, fallbackMode FallbackMode) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		// API documentation
		// https://www.dynatrace.com/support/help/dynatrace-api/environment-api/network-zones/put-network-zone

		nzApiUrl := fmt.Sprintf("%s/v2/networkZones/%s", secret.ApiUrl, networkZone)

		request := networkZoneRequestBody{
			AlternativeZones: alternativeZones,
			FallbackMode:     string(fallbackMode),
		}

		body, err := json.Marshal(request)
		require.NoError(t, err)

		client := &http.Client{}
		req, err := http.NewRequest(http.MethodPut, nzApiUrl, bytes.NewReader(body))
		require.NoError(t, err)

		req.Header.Add("Authorization", "Api-Token "+secret.ApiToken)
		req.Header.Add("Content-Type", "application/json")

		resp, err := client.Do(req)
		require.NoError(t, err)
		require.True(t, resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusNoContent)

		return ctx
	}
}

func DeleteNetworkZone(secret Secret, networkZone string) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		// API documentation
		// https://www.dynatrace.com/support/help/dynatrace-api/environment-api/network-zones/del-network-zone

		nzApiUrl := fmt.Sprintf("%s/v2/networkZones/%s", secret.ApiUrl, networkZone)

		client := &http.Client{}
		req, err := http.NewRequest(http.MethodDelete, nzApiUrl, nil)
		require.NoError(t, err)

		req.Header.Add("Authorization", "Api-Token "+secret.ApiToken)
		req.Header.Add("Content-Type", "application/json")

		resp, err := client.Do(req)
		defer func() { _ = resp.Body.Close() }()

		require.NoError(t, err)
		require.Truef(t, resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent, "delete network request failed, status = %d (%s)", resp.StatusCode, resp.Status)

		return ctx
	}
}
