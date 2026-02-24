package dynatrace

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	activeGateConnectionInfoEndpoint = "/v1/deployment/installer/gateway/connectioninfo"

	testTenantUUID  = "1234"
	testTenantToken = "abcd"
	testEndpoint    = "/some/url"
)

func Test_GetActiveGateConnectionInfo(t *testing.T) {
	ctx := context.Background()
	activegateJSONResponse := &activeGateConnectionInfoJSONResponse{
		TenantUUID:             testTenantUUID,
		TenantToken:            testTenantToken,
		CommunicationEndpoints: testEndpoint,
	}
	expectedActivegateConnectionInfo := ActiveGateConnectionInfo{
		ConnectionInfo: ConnectionInfo{
			TenantUUID:  testTenantUUID,
			TenantToken: testTenantToken,
			Endpoints:   testEndpoint,
		},
	}

	t.Run("no network zone", func(t *testing.T) {
		dynatraceServer, dynatraceClient := createTestDynatraceServer(t, connectionInfoServerHandler(activeGateConnectionInfoEndpoint, activegateJSONResponse), "")
		defer dynatraceServer.Close()

		connectionInfo, err := dynatraceClient.GetActiveGateConnectionInfo(ctx)
		require.NoError(t, err)
		assert.NotNil(t, connectionInfo)

		assert.Equal(t, expectedActivegateConnectionInfo, connectionInfo)
	})
	t.Run("with network zone", func(t *testing.T) {
		dynatraceServer, dynatraceClient := createTestDynatraceServer(t, connectionInfoServerHandler(activeGateConnectionInfoEndpoint, activegateJSONResponse), "nz")
		defer dynatraceServer.Close()

		connectionInfo, err := dynatraceClient.GetActiveGateConnectionInfo(ctx)
		require.NoError(t, err)
		assert.NotNil(t, connectionInfo)

		assert.Equal(t, expectedActivegateConnectionInfo, connectionInfo)
	})
	t.Run("with non-existent network zone", func(t *testing.T) {
		dynatraceServer, dynatraceClient := createTestDynatraceServer(t, connectionInfoServerHandler(activeGateConnectionInfoEndpoint, activegateJSONResponse), "")
		defer dynatraceServer.Close()

		connectionInfo, err := dynatraceClient.GetActiveGateConnectionInfo(ctx)
		require.NoError(t, err)
		assert.NotNil(t, connectionInfo)

		assert.Equal(t, expectedActivegateConnectionInfo, connectionInfo)
	})
	t.Run("handle malformed json", func(t *testing.T) {
		faultyDynatraceServer, faultyDynatraceClient := createTestDynatraceServer(t, tenantMalformedJSON(activeGateConnectionInfoEndpoint), "")
		defer faultyDynatraceServer.Close()

		connectionInfo, err := faultyDynatraceClient.GetActiveGateConnectionInfo(ctx)
		require.Error(t, err)
		assert.Equal(t, "invalid character 'h' in literal true (expecting 'r')", err.Error())

		assert.NotNil(t, connectionInfo)
		assert.Equal(t, ActiveGateConnectionInfo{}, connectionInfo)
	})
	t.Run("handle internal server error", func(t *testing.T) {
		faultyDynatraceServer, faultyDynatraceClient := createTestDynatraceServer(t, tenantInternalServerError(activeGateConnectionInfoEndpoint), "")
		defer faultyDynatraceServer.Close()

		connectionInfo, err := faultyDynatraceClient.GetActiveGateConnectionInfo(ctx)
		require.Error(t, err)
		assert.NotNil(t, connectionInfo)
		assert.Equal(t, ActiveGateConnectionInfo{}, connectionInfo)

		assert.Equal(t, "dynatrace server error 500: error retrieving tenant info", err.Error())
	})
}

func tenantMalformedJSON(url string) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path == url {
			writer.Write([]byte("this is not json"))
		} else {
			writer.WriteHeader(http.StatusBadRequest)
		}
	}
}

func tenantInternalServerError(url string) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path == url {
			rawData, err := json.Marshal(serverErrorResponse{
				ErrorMessage: ServerError{
					Code:    http.StatusInternalServerError,
					Message: "error retrieving tenant info",
				}})

			writer.WriteHeader(http.StatusInternalServerError)

			if err == nil {
				_, _ = writer.Write(rawData)
			}
		} else {
			writer.WriteHeader(http.StatusBadRequest)
		}
	}
}

func connectionInfoServerHandler(url string, response any) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path == url {
			rawData, err := json.Marshal(response)
			if err != nil {
				writer.WriteHeader(http.StatusInternalServerError)
			} else {
				writer.Header().Add("Content-Type", "application/json")
				_, _ = writer.Write(rawData)
			}
		} else {
			writer.WriteHeader(http.StatusBadRequest)
		}
	}
}
