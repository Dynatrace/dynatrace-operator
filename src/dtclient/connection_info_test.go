package dtclient

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	activeGateConnectionInfoEndpoint = "/v1/deployment/installer/gateway/connectioninfo"

	testTenantUUID  = "1234"
	testTenantToken = "abcd"
	testEndpoint    = "/some/url"
)

func Test_GetActiveGateConnectionInfo(t *testing.T) {
	activegateJsonResponse := &activeGateConnectionInfoJsonResponse{
		TenantUUID:             testTenantUUID,
		TenantToken:            testTenantToken,
		CommunicationEndpoints: testEndpoint,
	}
	expectedActivegateConnectionInfo := &ActiveGateConnectionInfo{
		ConnectionInfo: ConnectionInfo{
			TenantUUID:  testTenantUUID,
			TenantToken: testTenantToken,
			Endpoints:   testEndpoint,
		},
	}
	t.Run("no network zone", func(t *testing.T) {
		dynatraceServer, dynatraceClient := createTestDynatraceClient(t, connectionInfoServerHandler(activeGateConnectionInfoEndpoint, activegateJsonResponse), "")
		defer dynatraceServer.Close()

		connectionInfo, err := dynatraceClient.GetActiveGateConnectionInfo()
		assert.NoError(t, err)
		assert.NotNil(t, connectionInfo)

		assert.Equal(t, expectedActivegateConnectionInfo, connectionInfo)
	})
	t.Run("with network zone", func(t *testing.T) {
		dynatraceServer, dynatraceClient := createTestDynatraceClient(t, connectionInfoServerHandler(activeGateConnectionInfoEndpoint, activegateJsonResponse), "nz")
		defer dynatraceServer.Close()

		connectionInfo, err := dynatraceClient.GetActiveGateConnectionInfo()
		assert.NoError(t, err)
		assert.NotNil(t, connectionInfo)

		assert.Equal(t, expectedActivegateConnectionInfo, connectionInfo)
	})
	t.Run("with non-existent network zone", func(t *testing.T) {
		dynatraceServer, dynatraceClient := createTestDynatraceClient(t, connectionInfoServerHandler(activeGateConnectionInfoEndpoint, activegateJsonResponse), "")
		defer dynatraceServer.Close()

		connectionInfo, err := dynatraceClient.GetActiveGateConnectionInfo()
		assert.NoError(t, err)
		assert.NotNil(t, connectionInfo)

		assert.Equal(t, expectedActivegateConnectionInfo, connectionInfo)
	})
	t.Run("handle malformed json", func(t *testing.T) {
		faultyDynatraceServer, faultyDynatraceClient := createTestDynatraceClient(t, tenantMalformedJson(activeGateConnectionInfoEndpoint), "")
		defer faultyDynatraceServer.Close()

		connectionInfo, err := faultyDynatraceClient.GetActiveGateConnectionInfo()
		assert.Error(t, err)
		assert.Nil(t, connectionInfo)

		assert.Equal(t, "invalid character 'h' in literal true (expecting 'r')", err.Error())
	})
	t.Run("handle internal server error", func(t *testing.T) {
		faultyDynatraceServer, faultyDynatraceClient := createTestDynatraceClient(t, tenantInternalServerError(activeGateConnectionInfoEndpoint), "")
		defer faultyDynatraceServer.Close()

		connectionInfo, err := faultyDynatraceClient.GetActiveGateConnectionInfo()
		assert.Error(t, err)
		assert.Nil(t, connectionInfo)

		assert.Equal(t, "dynatrace server error 500: error retrieving tenant info", err.Error())
	})
}

func tenantMalformedJson(url string) http.HandlerFunc {
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

func connectionInfoServerHandler(url string, response interface{}) http.HandlerFunc {
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
