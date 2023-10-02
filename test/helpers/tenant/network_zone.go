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
	AlternativeZones []string `json:"alternativeZones"`
	FallbackMode     string   `json:"fallbackMode"`
	Id               string   `json:"id"`
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

		_, err = client.Do(req)
		require.NoError(t, err)

		return ctx
	}
}
