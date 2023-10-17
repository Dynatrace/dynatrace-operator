package edgeconnect

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

const (
	EdgeConnectOAuthClientID     = "test_client_id"
	EdgeConnectOAuthClientSecret = "test_client_secret"
	EdgeConnectID                = "348b4cd9-ba31-4670-9c45-9125a7d87439"
)

func TestNewClient(t *testing.T) {
	client, err := NewClient(
		"dummy_client_id",
		"dummy_client_secret",
		WithBaseURL("http://test.com"),
		WithTokenURL("http://test.com/token"),
	)
	assert.NoError(t, err)
	assert.NotNil(t, client)
}

func TestCreateEdgeConnect(t *testing.T) {
	t.Run("create basic edge connect", func(t *testing.T) {
		edgeConnectServer, edgeConnectClient := createTestEdgeConnectServer(t, edgeConnectServerHandler())
		defer edgeConnectServer.Close()

		resp, err := edgeConnectClient.CreateEdgeConnect("InternalServices", []string{"*.internal.org"}, "dt0s02.AIOUP56P")
		assert.NoError(t, err)
		assert.Equal(t, resp.Name, "InternalServices")
	})
}

func TestGetEdgeConnect(t *testing.T) {
	t.Run("get edge connect", func(t *testing.T) {
		edgeConnectServer, edgeConnectClient := createTestEdgeConnectServer(t, edgeConnectServerHandler())
		defer edgeConnectServer.Close()

		resp, err := edgeConnectClient.GetEdgeConnect("348b4cd9-ba31-4670-9c45-9125a7d87439")
		assert.NoError(t, err)
		assert.Equal(t, resp.Name, "InternalServices")
	})
}

func createTestEdgeConnectServer(t *testing.T, handler http.Handler) (*httptest.Server, Client) {
	edgeConnectServer := httptest.NewServer(handler)

	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, edgeConnectServer.Client())

	edgeConnectClient, err := NewClient(
		EdgeConnectOAuthClientID,
		EdgeConnectOAuthClientSecret,
		WithBaseURL(edgeConnectServer.URL),
		WithTokenURL(edgeConnectServer.URL+"/sso/oauth2/token"),
		WithContext(ctx),
	)

	require.NoError(t, err)
	require.NotNil(t, edgeConnectClient)

	return edgeConnectServer, edgeConnectClient
}

func edgeConnectServerHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		handleRequest(r, w)
	}
}

func handleRequest(request *http.Request, writer http.ResponseWriter) {
	switch {
	case request.URL.Path == "/sso/oauth2/token":
		writer.WriteHeader(http.StatusOK)
		out, _ := json.Marshal(map[string]string{
			"scope":        "app-engine:edge-connects:write app-engine:edge-connects:read oauth2:clients:manage app-engine:edge-connects:delete",
			"token_type":   "Bearer",
			"expires_in":   "300",
			"access_token": "access_token",
		})
		_, _ = writer.Write(out)
	case request.URL.Path == "/edge-connects" && request.Method == http.MethodPost:
		writer.WriteHeader(http.StatusOK)
		resp := CreateResponse{
			ID:            "348b4cd9-ba31-4670-9c45-9125a7d87439",
			Name:          "InternalServices",
			HostPatterns:  []string{"*.internal.org"},
			OauthClientId: "dt0s02.example",
			ModificationInfo: ModificationInfo{
				LastModifiedBy:   "72ece475-e4d5-4774-afed-65d04e8c9f24",
				LastModifiedTime: nil,
			},
		}
		out, _ := json.Marshal(resp)
		_, _ = writer.Write(out)
	case request.URL.Path == fmt.Sprintf("/edge-connects/%s", EdgeConnectID) && request.Method == http.MethodGet:
		writer.WriteHeader(http.StatusOK)
		resp := GetResponse{
			ID:            "348b4cd9-ba31-4670-9c45-9125a7d87439",
			Name:          "InternalServices",
			HostPatterns:  []string{"*.internal.org"},
			OauthClientId: "dt0s02.example",
			ModificationInfo: ModificationInfo{
				LastModifiedBy:   "72ece475-e4d5-4774-afed-65d04e8c9f24",
				LastModifiedTime: nil,
			},
		}
		out, _ := json.Marshal(resp)
		_, _ = writer.Write(out)
	default:
		writeError(writer, http.StatusBadRequest)
	}
}

func writeError(w http.ResponseWriter, status int) {
	message := serverErrorResponse{
		ErrorMessage: ServerError{
			Code:    status,
			Message: "error received from server",
		},
	}
	result, _ := json.Marshal(&message)

	w.WriteHeader(status)
	_, _ = w.Write(result)
}
