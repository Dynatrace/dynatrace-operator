package dtclient

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

const connectionInfoEndpoint = "/v1/deployment/installer/agent/connectioninfo"

var tenantResponse = struct {
	TenantUUID             string
	TenantToken            string
	CommunicationEndpoints []string
}{
	TenantUUID:             "abcd",
	TenantToken:            "1234",
	CommunicationEndpoints: []string{"/some/url"},
}

func TestTenant(t *testing.T) {
	t.Run("GetAgentTenantInfo", func(t *testing.T) {
		dynatraceServer, dynatraceClient := createTestDynatraceClient(t, tenantServerHandler())
		defer dynatraceServer.Close()

		tenantInfo, err := dynatraceClient.GetAgentTenantInfo()
		assert.NoError(t, err)
		assert.NotNil(t, tenantInfo)

		assert.Equal(t, tenantResponse.TenantUUID, tenantInfo.ID)
		assert.Equal(t, tenantResponse.TenantToken, tenantInfo.Token)
		assert.Equal(t, tenantResponse.CommunicationEndpoints, tenantInfo.Endpoints)
		assert.Equal(t,
			strings.Join([]string{tenantResponse.CommunicationEndpoints[0], DtCommunicationSuffix}, Slash),
			tenantInfo.CommunicationEndpoint)
	})
	t.Run("GetAgentTenantInfo handle internal server error", func(t *testing.T) {
		faultyDynatraceServer, faultyDynatraceClient := createTestDynatraceClient(t, tenantInternalServerError())
		defer faultyDynatraceServer.Close()

		tenantInfo, err := faultyDynatraceClient.GetAgentTenantInfo()
		assert.Error(t, err)
		assert.Nil(t, tenantInfo)

		assert.Equal(t, "dynatrace server error 500: error retrieving tenant info", err.Error())
	})
	t.Run("GetAgentTenantInfo handle malformed json", func(t *testing.T) {
		faultyDynatraceServer, faultyDynatraceClient := createTestDynatraceClient(t, tenantMalformedJson())
		defer faultyDynatraceServer.Close()

		tenantInfo, err := faultyDynatraceClient.GetAgentTenantInfo()
		assert.Error(t, err)
		assert.Nil(t, tenantInfo)

		assert.Equal(t, "invalid character 'h' in literal true (expecting 'r')", err.Error())
	})
}

func tenantMalformedJson() http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path == connectionInfoEndpoint {
			_, _ = writer.Write([]byte("this is not json"))
		} else {
			writer.WriteHeader(http.StatusBadRequest)
		}
	}
}

func tenantInternalServerError() http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path == connectionInfoEndpoint {
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

func tenantServerHandler() http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path == connectionInfoEndpoint {
			rawData, err := json.Marshal(tenantResponse)
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
