package version

import (
	"errors"
	"net/http"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core"
	coremock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetLatestActiveGateVersion(t *testing.T) {
	setupMockedClient := func(t *testing.T, os string) *Client {
		req := coremock.NewAPIRequest(t)
		req.EXPECT().
			WithPaasToken().
			Return(req).Once()
		req.EXPECT().
			Execute(new(struct {
				LatestGatewayVersion string `json:"latestGatewayVersion"`
			})).
			Run(func(model any) {
				resp := model.(*struct {
					LatestGatewayVersion string `json:"latestGatewayVersion"`
				})
				resp.LatestGatewayVersion = "1.2.3"
			}).
			Return(nil).Once()
		client := coremock.NewAPIClient(t)
		client.EXPECT().GET(t.Context(), getLatestActiveGateVersionPath(os)).Return(req).Once()

		return NewClient(client)
	}

	t.Run("ok, paas token", func(t *testing.T) {
		client := setupMockedClient(t, OsUnix)
		version, err := client.GetLatestActiveGateVersion(t.Context(), OsUnix)
		require.NoError(t, err)
		assert.Equal(t, "1.2.3", version)
	})

	t.Run("ok", func(t *testing.T) {
		versionClient := setupVersionClient(activeGateServerHandlerOk())

		response, err := versionClient.GetLatestActiveGateVersion(t.Context(), OsUnix)

		require.NoError(t, err)

		assert.Equal(t, "1.2.3", response)
	})

	t.Run("bad request", func(t *testing.T) {
		versionClient := setupVersionClient(activeGateServerHandlerBadRequest())

		_, err := versionClient.GetLatestActiveGateVersion(t.Context(), OsUnix)

		var httpErr *core.HTTPError
		ok := errors.As(err, &httpErr)
		require.True(t, ok)

		require.Len(t, httpErr.ServerErrors, 1)
		assert.Equal(t, http.StatusBadRequest, httpErr.ServerErrors[0].Code)
		assert.Equal(t, "Constraints violated.", httpErr.ServerErrors[0].Message)
	})

	t.Run("unauthorized", func(t *testing.T) {
		versionClient := setupVersionClient(activeGateServerHandlerUnauthorized())

		_, err := versionClient.GetLatestActiveGateVersion(t.Context(), OsUnix)

		var httpErr *core.HTTPError
		ok := errors.As(err, &httpErr)
		require.True(t, ok)

		require.Len(t, httpErr.ServerErrors, 1)
		assert.Equal(t, http.StatusUnauthorized, httpErr.ServerErrors[0].Code)
		assert.Equal(t, "Token Authentication failed", httpErr.ServerErrors[0].Message)
	})
}

func activeGateServerHandlerOk() http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")

		switch request.URL.Path {
		case getLatestActiveGateVersionPath(OsUnix):
			writer.WriteHeader(http.StatusOK)

			response := "{\"latestGatewayVersion\":\"1.2.3\"}"

			_, _ = writer.Write([]byte(response))
		default:
			writer.WriteHeader(http.StatusInternalServerError)
		}
	}
}

func activeGateServerHandlerBadRequest() http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")

		switch request.URL.Path {
		case getLatestActiveGateVersionPath(OsUnix):
			writer.WriteHeader(http.StatusBadRequest)
			_, _ = writer.Write([]byte("{\"error\":{\"code\":400,\"message\":\"Constraints violated.\",\"constraintViolations\":[{\"path\":\"bitness\",\"message\":\"'any' must be any of [...]\"}]}}"))
		default:
			writer.WriteHeader(http.StatusInternalServerError)
		}
	}
}

func activeGateServerHandlerUnauthorized() http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")

		switch request.URL.Path {
		case getLatestActiveGateVersionPath(OsUnix):
			writer.WriteHeader(http.StatusUnauthorized)
			_, _ = writer.Write([]byte("{\"error\":{\"code\":401,\"message\":\"Token Authentication failed\"}}"))
		default:
			writer.WriteHeader(http.StatusInternalServerError)
		}
	}
}
