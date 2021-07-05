package dtclient

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

const goodCommunicationEndpointsResponse = `{
	"tenantUUID": "aabb",
	"tenantToken": "testtoken",
	"communicationEndpoints": [
		"https://example.live.dynatrace.com/communication",
		"https://managedhost.com:9999/here/communication",
		"https://10.0.0.1:8000/communication",
		"http://insecurehost/communication"
	]
}`

const mixedCommunicationEndpointsResponse = `{
	"tenantUUID": "aabb",
	"tenantToken": "testtoken",
	"communicationEndpoints": [
		"https://example.live.dynatrace.com/communication",
		"https://managedhost.com:notaport/here/communication",
		"example.live.dynatrace.com:80/communication",
		"ftp://randomhost.com:80/communication",
		"unix:///some/local/file",
		"shouldnotbeparsed"
	]
}`

func TestReadCommunicationHosts(t *testing.T) {
	dc := &dynatraceClient{
		logger: consoleLogger,
	}

	readFromString := func(json string) (ConnectionInfo, error) {
		r := []byte(json)
		return dc.readResponseForConnectionInfo(r)
	}

	{
		m, err := readFromString(goodCommunicationEndpointsResponse)
		if assert.NoError(t, err) {
			expected := []*CommunicationHost{
				{Protocol: "https", Host: "example.live.dynatrace.com", Port: 443},
				{Protocol: "https", Host: "managedhost.com", Port: 9999},
				{Protocol: "https", Host: "10.0.0.1", Port: 8000},
				{Protocol: "http", Host: "insecurehost", Port: 80},
			}
			assert.Equal(t, expected, m.CommunicationHosts)
		}
	}
	{
		m, err := readFromString(mixedCommunicationEndpointsResponse)
		if assert.NoError(t, err) {
			expected := []*CommunicationHost{
				{Protocol: "https", Host: "example.live.dynatrace.com", Port: 443},
			}
			assert.Equal(t, expected, m.CommunicationHosts)
		}
	}
	{
		_, err := readFromString("")
		assert.Error(t, err, "empty response")
	}
	{
		_, err := readFromString(`{"communicationEndpoints": ["shouldnotbeparsed"]}`)
		assert.Error(t, err, "no hosts available")
	}
}

func TestParseEndpoints(t *testing.T) {
	var err error
	var ch *CommunicationHost

	ch, err = parseEndpoint("https://example.live.dynatrace.com/communication")
	assert.NoError(t, err)
	assert.Equal(t, &CommunicationHost{
		Protocol: "https",
		Host:     "example.live.dynatrace.com",
		Port:     443,
	}, ch)

	ch, err = parseEndpoint("https://managedhost.com:9999/here/communication")
	assert.NoError(t, err)
	assert.Equal(t, &CommunicationHost{
		Protocol: "https",
		Host:     "managedhost.com",
		Port:     9999,
	}, ch)

	ch, err = parseEndpoint("https://example.live.dynatrace.com/communication")
	assert.NoError(t, err)
	assert.Equal(t, &CommunicationHost{
		Protocol: "https",
		Host:     "example.live.dynatrace.com",
		Port:     443,
	}, ch)

	ch, err = parseEndpoint("https://10.0.0.1:8000/communication")
	assert.NoError(t, err)
	assert.Equal(t, &CommunicationHost{
		Protocol: "https",
		Host:     "10.0.0.1",
		Port:     8000,
	}, ch)

	ch, err = parseEndpoint("http://insecurehost/communication")
	assert.NoError(t, err)
	assert.Equal(t, &CommunicationHost{
		Protocol: "http",
		Host:     "insecurehost",
		Port:     80,
	}, ch)

	// Failures

	_, err = parseEndpoint("https://managedhost.com:notaport/here/communication")
	assert.Error(t, err)

	_, err = parseEndpoint("example.live.dynatrace.com:80/communication")
	assert.Error(t, err)

	_, err = parseEndpoint("ftp://randomhost.com:80/communication")
	assert.Error(t, err)

	_, err = parseEndpoint("unix:///some/local/file")
	assert.Error(t, err)

	_, err = parseEndpoint("shouldnotbeparsed")
	assert.Error(t, err)
}

func testCommunicationHostsGetCommunicationHosts(t *testing.T, dynatraceClient Client) {
	tenantInfo, err := dynatraceClient.GetAgentTenantInfo()
	res := tenantInfo.ConnectionInfo

	assert.NoError(t, err)
	assert.ObjectsAreEqualValues(res.CommunicationHosts, []*CommunicationHost{
		{Host: "host1.dynatracelabs.com", Port: 80, Protocol: "http"},
		{Host: "host2.dynatracelabs.com", Port: 443, Protocol: "https"},
		{Host: "12.0.9.1", Port: 80, Protocol: "http"},
	})
}

func handleCommunicationHosts(request *http.Request, writer http.ResponseWriter) {
	commHostOutput := []byte(`{
		"tenantUUID": "string",
		"tenantToken": "string",
		"communicationEndpoints": [
		  "http://host1.domain.com",
		  "https://host2.domain.com",
		  "http://host3.domain.com",
		  "http://12.0.9.1",
		  "http://12.0.10.1"
		]
	}`)

	switch request.Method {
	case "GET":
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write(commHostOutput)
	default:
		writeError(writer, http.StatusMethodNotAllowed)
	}
}
