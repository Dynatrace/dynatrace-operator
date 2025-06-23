package edgeconnect

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha2/edgeconnect"
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
		WithOauthScopes([]string{"test"}),
		WithBaseURL("http://test.com"),
		WithTokenURL("http://test.com/token"),
	)
	require.NoError(t, err)
	assert.NotNil(t, client)
}

func TestCreateEdgeConnect(t *testing.T) {
	t.Run("create basic edge connect", func(t *testing.T) {
		edgeConnectServer, edgeConnectClient := createTestEdgeConnectServer(t, edgeConnectCreateServerHandler(false))
		defer edgeConnectServer.Close()

		resp, err := edgeConnectClient.CreateEdgeConnect(NewRequest("InternalServices", []string{"*.internal.org"}, []edgeconnect.HostMapping{}, "dt0s02.AIOUP56P"))
		require.NoError(t, err)
		assert.Equal(t, "InternalServices", resp.Name)
	})
	t.Run("create basic edge connect without name returns error", func(t *testing.T) {
		edgeConnectServer, edgeConnectClient := createTestEdgeConnectServer(t, edgeConnectCreateServerHandler(true))
		defer edgeConnectServer.Close()

		_, err := edgeConnectClient.CreateEdgeConnect(NewRequest("", []string{"*.internal.org"}, []edgeconnect.HostMapping{}, "dt0s02.AIOUP56P"))
		require.Error(t, err, "edgeconnect server error 400: Constraints violated.")
	})
	t.Run("create edge connect with hostMappings", func(t *testing.T) {
		edgeConnectServer, edgeConnectClient := createTestEdgeConnectServer(t, edgeConnectCreateServerHandler(false))
		defer edgeConnectServer.Close()

		hostMappings := []edgeconnect.HostMapping{
			{
				From: "test-edgeconnect.test-namespace.test-kube-system-uid.kubernetes-automation",
				To:   edgeconnect.KubernetesDefaultDNS,
			},
		}

		_, err := edgeConnectClient.CreateEdgeConnect(NewRequest("InternalServices", []string{"*.internal.org"}, hostMappings, "dt0s02.AIOUP56P"))
		require.NoError(t, err)
	})
}

func TestGetEdgeConnect(t *testing.T) {
	t.Run("get edge connect", func(t *testing.T) {
		edgeConnectServer, edgeConnectClient := createTestEdgeConnectServer(t, edgeConnectGetServerHandler())
		defer edgeConnectServer.Close()

		resp, err := edgeConnectClient.GetEdgeConnect("348b4cd9-ba31-4670-9c45-9125a7d87439")
		require.NoError(t, err)
		assert.Equal(t, "InternalServices", resp.Name)
	})

	t.Run("get edge connect with wrong edge connect id", func(t *testing.T) {
		edgeConnectServer, edgeConnectClient := createTestEdgeConnectServer(t, edgeConnectGetServerHandler())
		defer edgeConnectServer.Close()

		_, err := edgeConnectClient.GetEdgeConnect("not-found")
		require.Error(t, err, http.StatusBadRequest)
	})
}

func TestDeleteEdgeConnect(t *testing.T) {
	t.Run("delete edge connect", func(t *testing.T) {
		edgeConnectServer, edgeConnectClient := createTestEdgeConnectServer(t, edgeConnectDeleteServerHandler())
		defer edgeConnectServer.Close()

		err := edgeConnectClient.DeleteEdgeConnect("348b4cd9-ba31-4670-9c45-9125a7d87439")
		require.NoError(t, err)
	})

	t.Run("delete edge connect with wrong edge connect id", func(t *testing.T) {
		edgeConnectServer, edgeConnectClient := createTestEdgeConnectServer(t, edgeConnectDeleteServerHandler())
		defer edgeConnectServer.Close()

		err := edgeConnectClient.DeleteEdgeConnect("not-found")
		require.Error(t, err, http.StatusBadRequest)
	})
}

func TestUpdateEdgeConnect(t *testing.T) {
	t.Run("update edge connect", func(t *testing.T) {
		edgeConnectServer, edgeConnectClient := createTestEdgeConnectServer(t, edgeConnectUpdateServerHandler())
		defer edgeConnectServer.Close()

		err := edgeConnectClient.UpdateEdgeConnect(EdgeConnectID, NewRequest("test_name", []string{""}, []edgeconnect.HostMapping{}, ""))
		require.NoError(t, err)
	})

	t.Run("update edge connect returns error", func(t *testing.T) {
		edgeConnectServer, edgeConnectClient := createTestEdgeConnectServer(t, edgeConnectUpdateServerHandler())
		defer edgeConnectServer.Close()

		err := edgeConnectClient.UpdateEdgeConnect("", NewRequest("test_name", []string{""}, []edgeconnect.HostMapping{}, ""))
		require.Error(t, err, http.StatusBadRequest)
	})
}

func createTestEdgeConnectServer(t *testing.T, handler http.Handler) (*httptest.Server, Client) {
	edgeConnectServer := httptest.NewServer(handler)

	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, edgeConnectServer.Client())

	edgeConnectClient, err := NewClient(
		EdgeConnectOAuthClientID,
		EdgeConnectOAuthClientSecret,
		WithOauthScopes([]string{"test_scopes"}),
		WithBaseURL(edgeConnectServer.URL),
		WithTokenURL(edgeConnectServer.URL+"/sso/oauth2/token"),
		WithContext(ctx),
	)

	require.NoError(t, err)
	require.NotNil(t, edgeConnectClient)

	return edgeConnectServer, edgeConnectClient
}

func writeOauthTokenResponse(writer http.ResponseWriter) {
	writer.WriteHeader(http.StatusOK)

	out, _ := json.Marshal(map[string]string{
		"scope":        "app-engine:edge-connects:write app-engine:edge-connects:read oauth2:clients:manage app-engine:edge-connects:delete",
		"token_type":   "Bearer",
		"expires_in":   "300",
		"access_token": "access_token",
	})
	_, _ = writer.Write(out)
}

func edgeConnectCreateServerHandler(errorBadRequest bool) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")

		switch request.URL.Path {
		case "/sso/oauth2/token":
			writeOauthTokenResponse(writer)
		case "/platform/app-engine/edge-connect/v1/edge-connects":
			if !errorBadRequest {
				if !isManagedByOperator(request) {
					writeError(writer, http.StatusBadRequest)

					return
				}

				writer.WriteHeader(http.StatusOK)

				resp := CreateResponse{
					ID:            "348b4cd9-ba31-4670-9c45-9125a7d87439",
					Name:          "InternalServices",
					HostPatterns:  []string{"*.internal.org"},
					OauthClientID: "dt0s02.example",
					ModificationInfo: ModificationInfo{
						LastModifiedBy:   "72ece475-e4d5-4774-afed-65d04e8c9f24",
						LastModifiedTime: nil,
					},
				}
				out, _ := json.Marshal(resp)
				_, _ = writer.Write(out)
			} else {
				writer.WriteHeader(http.StatusBadRequest)

				resp := serverErrorResponse{
					ErrorMessage: ServerError{
						Code:    400,
						Message: "Constraints violated.",
						Details: DetailsError{
							ConstraintViolations: []ConstraintViolations{
								{
									Message:           "must not be null",
									Path:              "path",
									ParameterLocation: "PAYLOAD_BODY",
								},
							},
						},
					},
				}
				out, _ := json.Marshal(resp)
				_, _ = writer.Write(out)
			}
		default:
			writeError(writer, http.StatusBadRequest)
		}
	}
}

func edgeConnectGetServerHandler() http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")

		switch request.URL.Path {
		case "/sso/oauth2/token":
			writeOauthTokenResponse(writer)
		case fmt.Sprintf("/platform/app-engine/edge-connect/v1/edge-connects/%s", EdgeConnectID):
			writer.WriteHeader(http.StatusOK)

			resp := GetResponse{
				ID:            "348b4cd9-ba31-4670-9c45-9125a7d87439",
				Name:          "InternalServices",
				HostPatterns:  []string{"*.internal.org"},
				OauthClientID: "dt0s02.example",
				ModificationInfo: ModificationInfo{
					LastModifiedBy:   "72ece475-e4d5-4774-afed-65d04e8c9f24",
					LastModifiedTime: nil,
				},
				ManagedByDynatraceOperator: true,
			}
			out, _ := json.Marshal(resp)
			_, _ = writer.Write(out)
		default:
			writeError(writer, http.StatusBadRequest)
		}
	}
}

func edgeConnectDeleteServerHandler() http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")

		switch request.URL.Path {
		case "/sso/oauth2/token":
			writeOauthTokenResponse(writer)
		case fmt.Sprintf("/platform/app-engine/edge-connect/v1/edge-connects/%s", EdgeConnectID):
			writer.WriteHeader(http.StatusNoContent)
		default:
			writeError(writer, http.StatusBadRequest)
		}
	}
}

func edgeConnectUpdateServerHandler() http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")

		switch request.URL.Path {
		case "/sso/oauth2/token":
			writeOauthTokenResponse(writer)
		case fmt.Sprintf("/platform/app-engine/edge-connect/v1/edge-connects/%s", EdgeConnectID):
			if !isManagedByOperator(request) {
				writeError(writer, http.StatusBadRequest)

				return
			}

			writer.WriteHeader(http.StatusOK)
		default:
			writeError(writer, http.StatusBadRequest)
		}
	}
}

func isManagedByOperator(request *http.Request) bool {
	edgeConnect := Request{}
	err := json.NewDecoder(request.Body).Decode(&edgeConnect)

	return err == nil && edgeConnect.ManagedByDynatraceOperator
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
