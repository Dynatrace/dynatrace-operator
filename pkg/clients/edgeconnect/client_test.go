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
	t.Run("ok", func(t *testing.T) {
		client, err := NewClient(
			"dummy_client_id",
			"dummy_client_secret",
			WithOauthScopes([]string{"test"}),
			WithBaseURL("http://test.com"),
			WithTokenURL("http://test.com/token"),
			WithCustomCA([]byte(customCA)),
		)
		require.NoError(t, err)
		assert.NotNil(t, client)
	})

	t.Run("invalid cert", func(t *testing.T) {
		client, err := NewClient(
			"dummy_client_id",
			"dummy_client_secret",
			WithOauthScopes([]string{"test"}),
			WithBaseURL("http://test.com"),
			WithTokenURL("http://test.com/token"),
			WithCustomCA([]byte("invalid")),
		)
		require.Error(t, err)
		assert.Nil(t, client)
	})
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

// Generated with:
// openssl genrsa -out ca.key 2048
// openssl req -x509 -new -nodes -key ca.key -sha256 -days 3650 -out ca.crt -subj '/CN=Test CA/C=AT/ST=UA/L=Linz/O=Dynatrace/OU=Operator' -extensions v3_ca

const customCA = `-----BEGIN CERTIFICATE-----
MIIDpTCCAo2gAwIBAgIUTCZa2IYrncHLMV2nLNbXOcQqc/4wDQYJKoZIhvcNAQEL
BQAwYjEQMA4GA1UEAwwHVGVzdCBDQTELMAkGA1UEBhMCQVQxCzAJBgNVBAgMAlVB
MQ0wCwYDVQQHDARMaW56MRIwEAYDVQQKDAlEeW5hdHJhY2UxETAPBgNVBAsMCE9w
ZXJhdG9yMB4XDTI1MTEwNjEzMDQyMVoXDTM1MTEwNDEzMDQyMVowYjEQMA4GA1UE
AwwHVGVzdCBDQTELMAkGA1UEBhMCQVQxCzAJBgNVBAgMAlVBMQ0wCwYDVQQHDARM
aW56MRIwEAYDVQQKDAlEeW5hdHJhY2UxETAPBgNVBAsMCE9wZXJhdG9yMIIBIjAN
BgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEArw564P5tXzT2uo0uRJdjhe+zGyU4
1zWdp6sIFB3J3KWKaAQ9ao7oMu75+pFo11c1XFuZcZpRmucWZ1AMWNm6Mga4yn6y
OcC+cIpDMT1kXnix+u7TH+XwOXkIty0T7I5OyiVV5JEryrl3jTjXRf4YbHRVrc4w
vspbS4JIxx+Hv6u4/sRRSvBI89hQ8miGgtOwuokGBIxcOKf/lqe10Q9SMuK+mAmP
jFlNlnOteFwTRBLWFJlDFgE+jxAyP3FGUIwLNN6w+DKzb4cmjnBk8TK3CHxhJREl
cncQnIXAp4Sq6VfR6mLGGyGpt3OWnm0L/cPASed5gp3V1CUW0T3Iz21VVwIDAQAB
o1MwUTAdBgNVHQ4EFgQULiJWJ0CXf4aoFki24ef2gRH0EU8wHwYDVR0jBBgwFoAU
LiJWJ0CXf4aoFki24ef2gRH0EU8wDwYDVR0TAQH/BAUwAwEB/zANBgkqhkiG9w0B
AQsFAAOCAQEAnLgaLr2qpVM6heHaBHt+vDWNda9YkfUGCfGU64AZf5kT9fQWFaXi
Liv0TC1NBOTHJ35DjSc4O/EshfO/qW0eMnLw8u4gfhPKs7mmADkcy4V/rhyA/hTU
1Vx+MSJsKH2vJAODaELKZZ3AiA9Rfyyt6Nv+nUtHRtBLpRmrYnVLlZHgfMvfSmnk
zWDF6rXZJXT6MJUcf740v4MOLlIWcrNj/igI9VQP9cBrhvJzthHJ0gMEjNqKJPgk
APj12zaRa05OBW3H3Ng+1MmdtrU4gAu+xwLAOz1cxT6q8LUGBGDCBYVcFXvomhKL
kHUfKUp2W9zOWWDlwSB65QuJ3wAQSCVs4g==
-----END CERTIFICATE-----`
