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
		client := MockEdgeConnectClient(http.StatusOK)
		es, err := client.GetConnectionSetting("test-name", "test-namespace", "test-uid")
		require.NoError(t, err)
		require.NotNil(t, es)
	})
	t.Run("Server response NOK", func(t *testing.T) {
		client := MockEdgeConnectClient(http.StatusBadRequest)
		es, err := client.GetConnectionSetting("test-name", "test-namespace", "test-uid")
		require.Error(t, err)
		require.Equal(t, EnvironmentSetting{}, es)
	})
}

func TestCreateConnectionSetting(t *testing.T) {
	t.Run("Server response OK", func(t *testing.T) {
		client := MockEdgeConnectClient(http.StatusOK)
		err := client.CreateConnectionSetting(testEnvironmentSetting)
		require.NoError(t, err)
	})
	t.Run("Server response NOK", func(t *testing.T) {
		client := MockEdgeConnectClient(http.StatusBadRequest)
		err := client.CreateConnectionSetting(testEnvironmentSetting)
		require.Error(t, err)
	})
}

func TestUpdateConnectionSetting(t *testing.T) {
	t.Run("Server response OK", func(t *testing.T) {
		client := MockEdgeConnectClient(http.StatusOK)
		err := client.UpdateConnectionSetting(testEnvironmentSetting)
		require.NoError(t, err)
	})
	t.Run("Server response NOK", func(t *testing.T) {
		client := MockEdgeConnectClient(http.StatusBadRequest)
		err := client.UpdateConnectionSetting(testEnvironmentSetting)
		require.Error(t, err)
	})
}

func TestDeleteConnectionSetting(t *testing.T) {
	t.Run("Server response OK", func(t *testing.T) {
		client := MockEdgeConnectClient(http.StatusOK)
		err := client.DeleteConnectionSetting("test-objectId")
		require.NoError(t, err)
	})
	t.Run("Server response NOK", func(t *testing.T) {
		client := MockEdgeConnectClient(http.StatusBadRequest)
		err := client.DeleteConnectionSetting("test-objectId")
		require.Error(t, err)
	})
}

func MockEdgeConnectClient(status int) Client {
	edgeConnectServer := httptest.NewServer(edgeConnectServerHandler(status))

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

func edgeConnectServerHandler(status int) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")

		switch request.URL.Path {
		case "/sso/oauth2/token":
			writeOauthTokenResponse(writer)
		case "/platform/classic/environment-api/v2/settings/objects":
			writeEnvironmentSettingsResponse(writer, status)
		default:
			writeSettingsApiStatusResponse(writer, status)
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
		writeSettingsApiStatusResponse(w, status)
	}
}

func writeSettingsApiStatusResponse(w http.ResponseWriter, status int) {
	errorResponse := []SettingsApiResponse{}
	errorResponse = append(errorResponse, SettingsApiResponse{
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
	},
	)
	result, _ := json.Marshal(&errorResponse)

	w.WriteHeader(status)
	_, _ = w.Write(result)
}
