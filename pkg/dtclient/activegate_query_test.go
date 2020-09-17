package dtclient

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
)

func TestBuildQueryParams(t *testing.T) {
	query := ActiveGateQuery{}
	defaultParams := ""
	params := buildQueryParams(&query)
	assert.Equal(t, "osType="+OsLinux+"&type=ENVIRONMENT", params)

	params = buildQueryParams(nil)
	assert.Equal(t, "osType="+OsLinux+"&type=ENVIRONMENT", params)
	defaultParams = params

	query.UpdateStatus = "updating"
	params = buildQueryParams(&query)
	assert.Equal(t, "updateStatus=updating&"+defaultParams, params)

	query.UpdateStatus = ""
	query.NetworkAddress = "1.2.3.4"
	params = buildQueryParams(&query)
	assert.Equal(t, "networkAddress=1.2.3.4&"+defaultParams, params)

	query.NetworkAddress = ""
	query.NetworkZone = "zone"
	params = buildQueryParams(&query)
	assert.Equal(t, "networkZone=zone&"+defaultParams, params)

	query.NetworkZone = ""
	query.Hostname = "host-42"
	params = buildQueryParams(&query)
	assert.Equal(t, "hostname=host-42&"+defaultParams, params)

	query.UpdateStatus = "updating"
	query.NetworkAddress = "1.2.3.4"
	query.NetworkZone = "zone"
	query.Hostname = "host-42"
	params = buildQueryParams(&query)
	assert.Equal(t, "hostname=host-42&"+
		"networkZone=zone&"+
		"networkAddress=1.2.3.4&"+
		"updateStatus=updating&"+defaultParams, params)
}

func TestQueryActiveGates(t *testing.T) {
	query := ActiveGateQuery{}
	dynatraceServer, dynatraceClient := createTestDynatraceClient(t, activeGateHandler())
	defer dynatraceServer.Close()

	t.Run("QueryActiveGates", func(t *testing.T) {
		activeGates, err := dynatraceClient.QueryActiveGates(&query)
		assert.NoError(t, err)
		assert.NotNil(t, activeGates)
		assert.NotEmpty(t, activeGates)
		assert.Equal(t, 1, len(activeGates))

		activeGates, err = dynatraceClient.QueryOutdatedActiveGates(&query)
		assert.NoError(t, err)
		assert.NotNil(t, activeGates)
		assert.NotEmpty(t, activeGates)
		assert.Equal(t, 1, len(activeGates))
		assert.Equal(t, query.UpdateStatus, StatusOutdated)
	})
	t.Run("QueryActiveGates nil query", func(t *testing.T) {
		activeGates, err := dynatraceClient.QueryActiveGates(nil)
		assert.NoError(t, err)
		assert.NotNil(t, activeGates)
		assert.NotEmpty(t, activeGates)
		assert.Equal(t, 1, len(activeGates))
	})
	t.Run("QueryActiveGates handle server error", func(t *testing.T) {
		dynatraceServer, dynatraceClient = createTestDynatraceClient(t, activeGateHandlerError())
		defer dynatraceServer.Close()

		activeGates, err := dynatraceClient.QueryActiveGates(nil)
		assert.Error(t, err)
		assert.Nil(t, activeGates)
	})
	t.Run("QueryActiveGates handle malformed json", func(t *testing.T) {
		dynatraceServer, dynatraceClient = createTestDynatraceClient(t, activeGateHandlerMalformedJson())
		defer dynatraceServer.Close()

		activeGates, err := dynatraceClient.QueryActiveGates(nil)
		assert.Error(t, err)
		assert.Nil(t, activeGates)
	})
	t.Run("QueryActiveGates handle request error", func(t *testing.T) {
		dynatraceServer, dynatraceClient = createTestDynatraceClient(t, activeGateHandlerMalformedJson())
		dynatraceServer.Close()

		activeGates, err := dynatraceClient.QueryActiveGates(nil)
		assert.Error(t, err)
		assert.Nil(t, activeGates)
	})
}

func activeGateHandlerMalformedJson() http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		_, _ = writer.Write([]byte("not json"))
	}
}

func activeGateHandlerError() http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		writeError(writer, http.StatusInternalServerError)
	}
}

func activeGateHandler() http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		data := struct {
			ActiveGates []ActiveGate
		}{
			ActiveGates: []ActiveGate{
				{Hostname: "host-42"},
				{
					Hostname:     "host-43",
					OfflineSince: 100,
				},
			},
		}
		rawData, _ := json.Marshal(data)
		_, _ = writer.Write(rawData)
	}
}
