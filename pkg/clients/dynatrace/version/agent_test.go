package version

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/arch"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/installer"
	coremock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetLatestAgentVersion(t *testing.T) {
	setupMockedClient := func(t *testing.T, os string, installerType string, queryParams map[string]string) *Client {
		req := coremock.NewAPIRequest(t)
		req.EXPECT().
			WithQueryParams(queryParams).
			Return(req).Once()
		req.EXPECT().
			WithPaasToken().
			Return(req).Once()
		req.EXPECT().
			Execute(new(struct {
				LatestAgentVersion string `json:"latestAgentVersion"`
			})).
			Run(func(model any) {
				resp := model.(*struct {
					LatestAgentVersion string `json:"latestAgentVersion"`
				})
				resp.LatestAgentVersion = "1.2.3"
			}).
			Return(nil).Once()
		client := coremock.NewAPIClient(t)
		client.EXPECT().GET(t.Context(), getLatestAgentVersionPath(os, installerType)).Return(req).Once()

		return NewClient(client)
	}

	t.Run("ok, uses paas token, installer.TypeDefault", func(t *testing.T) {
		queryParams := map[string]string{
			"bitness": "64",
			"flavor":  arch.FlavorDefault,
		}

		client := setupMockedClient(t, installer.OsUnix, installer.TypeDefault, queryParams)
		version, err := client.GetLatestAgentVersion(t.Context(), installer.OsUnix, installer.TypeDefault)
		require.NoError(t, err)
		assert.Equal(t, "1.2.3", version)
	})

	t.Run("ok, uses paas token, installer.TypePaaS", func(t *testing.T) {
		queryParams := map[string]string{
			"bitness": "64",
			"flavor":  arch.Flavor,
			"arch":    arch.Arch,
		}

		client := setupMockedClient(t, installer.OsUnix, installer.TypePaaS, queryParams)
		version, err := client.GetLatestAgentVersion(t.Context(), installer.OsUnix, installer.TypePaaS)
		require.NoError(t, err)
		assert.Equal(t, "1.2.3", version)
	})

	t.Run("ok", func(t *testing.T) {
		versionClient := setupVersionClient(agentServerHandlerOk())

		response, err := versionClient.GetLatestAgentVersion(t.Context(), installer.OsUnix, installer.TypeDefault)

		require.NoError(t, err)

		assert.Equal(t, "1.2.3", response)
	})

	t.Run("bad request", func(t *testing.T) {
		versionClient := setupVersionClient(agentServerHandlerBadRequest())

		_, err := versionClient.GetLatestAgentVersion(t.Context(), installer.OsUnix, installer.TypeDefault)

		var httpErr *core.HTTPError
		ok := errors.As(err, &httpErr)
		require.True(t, ok)

		require.Len(t, httpErr.ServerErrors, 1)
		assert.Equal(t, http.StatusBadRequest, httpErr.ServerErrors[0].Code)
		assert.Equal(t, "Constraints violated.", httpErr.ServerErrors[0].Message)
	})

	t.Run("unauthorized", func(t *testing.T) {
		versionClient := setupVersionClient(agentServerHandlerUnauthorized())

		_, err := versionClient.GetLatestAgentVersion(t.Context(), installer.OsUnix, installer.TypeDefault)

		var httpErr *core.HTTPError
		ok := errors.As(err, &httpErr)
		require.True(t, ok)

		require.Len(t, httpErr.ServerErrors, 1)
		assert.Equal(t, http.StatusUnauthorized, httpErr.ServerErrors[0].Code)
		assert.Equal(t, "Token Authentication failed", httpErr.ServerErrors[0].Message)
	})
}

func setupVersionClient(handler http.HandlerFunc) *Client {
	dtServer := httptest.NewServer(handler)

	dtServerURL, _ := url.Parse(dtServer.URL)

	apiClient := core.NewClient(core.Config{
		BaseURL:    dtServerURL,
		HTTPClient: dtServer.Client(),
	})

	return NewClient(apiClient)
}

func agentServerHandlerOk() http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")

		switch request.URL.Path {
		case getLatestAgentVersionPath(installer.OsUnix, installer.TypeDefault):
			writer.WriteHeader(http.StatusOK)

			response := "{\"latestAgentVersion\":\"1.2.3\"}"

			_, _ = writer.Write([]byte(response))
		default:
			writer.WriteHeader(http.StatusInternalServerError)
		}
	}
}

func agentServerHandlerBadRequest() http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")

		switch request.URL.Path {
		case getLatestAgentVersionPath(installer.OsUnix, installer.TypeDefault):
			writer.WriteHeader(http.StatusBadRequest)

			_, _ = writer.Write([]byte("{\"error\":{\"code\":400,\"message\":\"Constraints violated.\",\"constraintViolations\":[{\"path\":\"bitness\",\"message\":\"'any' must be any of [...]\"}]}}"))
		default:
			writer.WriteHeader(http.StatusInternalServerError)
		}
	}
}

func agentServerHandlerUnauthorized() http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")

		switch request.URL.Path {
		case getLatestAgentVersionPath(installer.OsUnix, installer.TypeDefault):
			writer.WriteHeader(http.StatusUnauthorized)

			_, _ = writer.Write([]byte("{\"error\":{\"code\":401,\"message\":\"Token Authentication failed\"}}"))
		default:
			writer.WriteHeader(http.StatusInternalServerError)
		}
	}
}
