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
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
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

		request := networkZoneRequestBody{
			AlternativeZones: alternativeZones,
			FallbackMode:     string(fallbackMode),
		}

		body, err := json.Marshal(request)
		require.NoError(t, err)

		statusCode, statusMsg, err := executeNetworkZoneRequest(secret, networkZone, http.MethodPut, bytes.NewReader(body))
		require.NoError(t, err)
		assert.Truef(t, statusCode == http.StatusCreated || statusCode == http.StatusNoContent, statusMsg)

		return ctx
	}
}

func WaitForNetworkZoneDeletion(secret Secret, networkZone string) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		err := wait.For(deleteNetworkZone(secret, networkZone), wait.WithTimeout(3*time.Minute))
		require.NoError(t, err)
		return ctx
	}
}

func createNetworkZoneRequest(secret Secret, networkZone string, method string, body io.Reader) (*http.Request, error) {
	nzApiUrl := fmt.Sprintf("%s/v2/networkZones/%s", secret.ApiUrl, networkZone)

	req, err := http.NewRequest(method, nzApiUrl, body)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", "Api-Token "+secret.ApiToken)
	req.Header.Add("Content-Type", "application/json")

	return req, nil
}

func executeNetworkZoneRequest(secret Secret, networkZone string, method string, body io.Reader) (statusCode int, nsg string, err error) {
	client := &http.Client{}

	req, err := createNetworkZoneRequest(secret, networkZone, method, body)
	if err != nil {
		return 0, "", err
	}

	resp, err := client.Do(req)
	defer func() {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()

	if err != nil {
		return 0, "", err
	}
	if resp == nil {
		return 0, "", errors.Errorf("response was nil")
	}

	var respBody []byte
	if resp.Body != nil {
		respBody, err = io.ReadAll(resp.Body)
		if err != nil {
			return resp.StatusCode, resp.Status, err
		}
	}

	return resp.StatusCode, resp.Status + " " + string(respBody), nil
}

func deleteNetworkZone(secret Secret, networkZone string) func(ctx context.Context) (done bool, err error) {
	return func(ctx context.Context) (done bool, err error) {
		// API documentation
		// https://www.dynatrace.com/support/help/dynatrace-api/environment-api/network-zones/del-network-zone

		statusCode, statusMsg, err := executeNetworkZoneRequest(secret, networkZone, http.MethodDelete, nil)
		if err != nil {
			return false, err
		}

		if statusCode == http.StatusOK || statusCode == http.StatusNoContent {
			return true, nil
		} else if statusCode == http.StatusBadRequest {
			// this error can indicate, that the networkzone is still used by an ActiveGate, just try again later
			return false, nil
		}

		return false, errors.Errorf("delete network zone request returned status = %d (%s)", statusCode, statusMsg)
	}
}
