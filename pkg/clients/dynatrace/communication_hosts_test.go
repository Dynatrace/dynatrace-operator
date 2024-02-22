package dynatrace

import (
	"context"
	"net/http"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	dc := &dynatraceClient{}

	readFromString := func(json string) (OneAgentConnectionInfo, error) {
		response := []byte(json)

		return dc.readResponseForOneAgentConnectionInfo(response)
	}

	{
		m, err := readFromString(goodCommunicationEndpointsResponse)
		require.NoError(t, err)

		expected := []CommunicationHost{
			{Protocol: "https", Host: "10.0.0.1", Port: 8000},
			{Protocol: "https", Host: "example.live.dynatrace.com", Port: 443},
			{Protocol: "http", Host: "insecurehost", Port: 80},
			{Protocol: "https", Host: "managedhost.com", Port: 9999},
		}

		sort.Slice(m.CommunicationHosts, func(i, j int) bool {
			return m.CommunicationHosts[i].Host < m.CommunicationHosts[j].Host
		})
		assert.Equal(t, expected, m.CommunicationHosts)
	}
	{
		m, err := readFromString(mixedCommunicationEndpointsResponse)
		require.NoError(t, err)

		expected := []CommunicationHost{
			{Protocol: "https", Host: "example.live.dynatrace.com", Port: 443},
		}
		assert.Equal(t, expected, m.CommunicationHosts)
	}
	{
		_, err := readFromString("")
		require.Error(t, err, "empty response")
	}
	{
		_, err := readFromString(`{"communicationEndpoints": ["shouldnotbeparsed"]}`)
		require.NoError(t, err)
	}
}

func TestParseEndpoints(t *testing.T) {
	var err error

	var ch CommunicationHost

	ch, err = ParseEndpoint("https://example.live.dynatrace.com/communication")
	require.NoError(t, err)
	assert.Equal(t, CommunicationHost{
		Protocol: "https",
		Host:     "example.live.dynatrace.com",
		Port:     443,
	}, ch)

	ch, err = ParseEndpoint("https://managedhost.com:9999/here/communication")
	require.NoError(t, err)
	assert.Equal(t, CommunicationHost{
		Protocol: "https",
		Host:     "managedhost.com",
		Port:     9999,
	}, ch)

	ch, err = ParseEndpoint("https://example.live.dynatrace.com/communication")
	require.NoError(t, err)
	assert.Equal(t, CommunicationHost{
		Protocol: "https",
		Host:     "example.live.dynatrace.com",
		Port:     443,
	}, ch)

	ch, err = ParseEndpoint("https://10.0.0.1:8000/communication")
	require.NoError(t, err)
	assert.Equal(t, CommunicationHost{
		Protocol: "https",
		Host:     "10.0.0.1",
		Port:     8000,
	}, ch)

	ch, err = ParseEndpoint("http://insecurehost/communication")
	require.NoError(t, err)
	assert.Equal(t, CommunicationHost{
		Protocol: "http",
		Host:     "insecurehost",
		Port:     80,
	}, ch)

	// Failures

	_, err = ParseEndpoint("https://managedhost.com:notaport/here/communication")
	require.Error(t, err)

	_, err = ParseEndpoint("example.live.dynatrace.com:80/communication")
	require.Error(t, err)

	_, err = ParseEndpoint("ftp://randomhost.com:80/communication")
	require.Error(t, err)

	_, err = ParseEndpoint("unix:///some/local/file")
	require.Error(t, err)

	_, err = ParseEndpoint("shouldnotbeparsed")
	require.Error(t, err)
}

func testCommunicationHostsGetCommunicationHosts(t *testing.T, dynatraceClient Client) {
	ctx := context.Background()
	res, err := dynatraceClient.GetOneAgentConnectionInfo(ctx)

	require.NoError(t, err)
	assert.ObjectsAreEqualValues(res.CommunicationHosts, []CommunicationHost{
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
	case http.MethodGet:
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write(commHostOutput)
	default:
		writeError(writer, http.StatusMethodNotAllowed)
	}
}
