//go:build e2e

package tenant

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

type FallbackMode string

const (
	FallbackAnyActiveGate FallbackMode = "ANY_ACTIVE_GATE"
	FallbackNone          FallbackMode = "NONE"
	FallbackDefaultZone   FallbackMode = "ONLY_DEFAULT_ZONE"
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
		defer func() { _ = resp.Body.Close() }()
		require.NoError(t, err)
		require.True(t, resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusNoContent)

		return ctx
	}
}

func WaitForNetworkZoneDeletion(secret Secret, networkZone string) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		err := wait.For(deleteNetworkZone(secret, networkZone))
		require.NoError(t, err)
		return ctx
	}
}

func deleteNetworkZone(secret Secret, networkZone string) func(ctx context.Context) (done bool, err error) {
	return func(ctx context.Context) (done bool, err error) {
		// API documentation
		// https://www.dynatrace.com/support/help/dynatrace-api/environment-api/network-zones/del-network-zone

		nzApiUrl := fmt.Sprintf("%s/v2/networkZones/%s", secret.ApiUrl, networkZone)

		client := &http.Client{}
		req, err := http.NewRequest(http.MethodDelete, nzApiUrl, nil)
		if err != nil {
			return false, err
		}

		req.Header.Add("Authorization", "Api-Token "+secret.ApiToken)
		req.Header.Add("Content-Type", "application/json")

		resp, err := client.Do(req)
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent {
			return true, nil
		}

		if resp.StatusCode == http.StatusBadRequest {
			// this error can indicate, that the networkzone is still used by an ActiveGate, just try again later
			return false, nil
		}

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return false, err
		}
		return false, errors.Errorf("delete network zone request failed, status = %d (%s): %s", resp.StatusCode, resp.Status, string(respBody))
	}
}
