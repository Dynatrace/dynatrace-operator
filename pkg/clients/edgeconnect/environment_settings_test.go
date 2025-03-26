package edgeconnect

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

var testObjectId = "test-objectId"

var testEnvironmentSetting = EnvironmentSetting{
	ObjectId: &testObjectId,
	SchemaId: KubernetesConnectionSchemaID,
	Scope:    KubernetesConnectionScope,
	Value: EnvironmentSettingValue{
		Name:      "test-name",
		UID:       "test-uid",
		Namespace: "test-namespace",
		Token:     "test-token",
	},
}

func TestGetConnectionSetting(t *testing.T) {
	t.Run("Server response OK", func(t *testing.T) {
		client := mockEdgeConnectClient(mockServerHandler(http.StatusOK))
		got, err := client.GetConnectionSettings()
		require.NoError(t, err)
		require.NotNil(t, got)
	})
	t.Run("Server response NOK", func(t *testing.T) {
		client := mockEdgeConnectClient(mockServerHandler(http.StatusBadRequest))
		got, err := client.GetConnectionSettings()
		require.Error(t, err)
		require.Nil(t, got)
	})
	t.Run("Server response unexpected", func(t *testing.T) {
		client := mockEdgeConnectClient(mockUnexpectedServerHandler())
		got, err := client.GetConnectionSettings()
		require.Error(t, err)
		require.Nil(t, got)
	})
}

func TestCreateConnectionSetting(t *testing.T) {
	t.Run("Server response OK", func(t *testing.T) {
		client := mockEdgeConnectClient(mockServerHandler(http.StatusOK))
		err := client.CreateConnectionSetting(testEnvironmentSetting)
		require.NoError(t, err)
	})
	t.Run("Server response NOK", func(t *testing.T) {
		client := mockEdgeConnectClient(mockServerHandler(http.StatusBadRequest))
		err := client.CreateConnectionSetting(testEnvironmentSetting)
		require.Error(t, err)
	})
	t.Run("Server response unexpected", func(t *testing.T) {
		client := mockEdgeConnectClient(mockUnexpectedServerHandler())
		err := client.CreateConnectionSetting(testEnvironmentSetting)
		require.Error(t, err)
	})
}

func TestUpdateConnectionSetting(t *testing.T) {
	t.Run("Server response OK", func(t *testing.T) {
		client := mockEdgeConnectClient(mockServerHandler(http.StatusOK))
		err := client.UpdateConnectionSetting(testEnvironmentSetting)
		require.NoError(t, err)
	})
	t.Run("Server response NOK", func(t *testing.T) {
		client := mockEdgeConnectClient(mockServerHandler(http.StatusBadRequest))
		err := client.UpdateConnectionSetting(testEnvironmentSetting)
		require.Error(t, err)
	})
	t.Run("Server response unexpected", func(t *testing.T) {
		client := mockEdgeConnectClient(mockUnexpectedServerHandler())
		err := client.UpdateConnectionSetting(testEnvironmentSetting)
		require.Error(t, err)
	})
}

func TestDeleteConnectionSetting(t *testing.T) {
	t.Run("Server response OK", func(t *testing.T) {
		client := mockEdgeConnectClient(mockServerHandler(http.StatusOK))
		err := client.DeleteConnectionSetting(testObjectId)
		require.NoError(t, err)
	})
	t.Run("Server response NOK", func(t *testing.T) {
		client := mockEdgeConnectClient(mockServerHandler(http.StatusBadRequest))
		err := client.DeleteConnectionSetting(testObjectId)
		require.Error(t, err)
	})
	t.Run("Server response unexpected", func(t *testing.T) {
		client := mockEdgeConnectClient(mockUnexpectedServerHandler())
		err := client.DeleteConnectionSetting(testObjectId)
		require.Error(t, err)
	})
}

func mockEdgeConnectClient(handler http.HandlerFunc) Client {
	edgeConnectServer := httptest.NewServer(handler)
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, edgeConnectServer.Client())

	edgeConnectClient, _ := NewClient(
		EdgeConnectOAuthClientID,
		EdgeConnectOAuthClientSecret,
		WithOauthScopes([]string{"test_scopes"}),
		WithBaseURL(edgeConnectServer.URL),
		WithTokenURL(edgeConnectServer.URL+"/sso/oauth2/token"),
		WithContext(ctx),
	)

	return edgeConnectClient
}

func mockServerHandler(status int) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")

		switch request.URL.Path {
		case "/sso/oauth2/token":
			writeOauthTokenResponse(writer)
		case "/platform/classic/environment-api/v2/settings/objects":
			writeEnvironmentSettingsResponse(writer, status)
		default:
			writeSettingsApiResponse(writer, status)
		}
	}
}

func mockUnexpectedServerHandler() http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")

		switch request.URL.Path {
		case "/sso/oauth2/token":
			writeOauthTokenResponse(writer)
		default:
			writeUnexpectedResponse(writer)
		}
	}
}

func writeEnvironmentSettingsResponse(w http.ResponseWriter, status int) {
	if status == http.StatusOK {
		response := EnvironmentSettingsResponse{
			Items: []EnvironmentSetting{
				testEnvironmentSetting,
			},
			TotalCount: 1,
			PageSize:   1,
		}

		result, _ := json.Marshal(&response)

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(result)
	} else {
		writeSettingsApiResponse(w, status)
	}
}

func writeSettingsApiResponse(w http.ResponseWriter, status int) {
	errorResponse := SettingsApiResponse{
		Error: SettingsApiError{
			Message: "test-message",
			ConstraintViolations: []ConstraintViolations{
				{
					Message:           "test-constraint-message",
					Path:              "test-constraint-path",
					ParameterLocation: "test-constraint-parameterLocation",
				},
			},
			Code: status,
		},
		Code: status,
	}

	result, _ := json.Marshal(&errorResponse)

	w.WriteHeader(status)
	_, _ = w.Write(result)
}

func writeUnexpectedResponse(w http.ResponseWriter) {
	unexpectedResponse := `{"status":500, "message":"everything is broken!!!"}`
	result, _ := json.Marshal(&unexpectedResponse)

	w.WriteHeader(http.StatusServiceUnavailable)
	_, _ = w.Write(result)
}
