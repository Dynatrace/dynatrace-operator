package dynatrace

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCommunicationHost(t *testing.T) {
	var err error

	var ch CommunicationHost

	ch, err = NewCommunicationHost("https://example.live.dynatrace.com/communication")
	require.NoError(t, err)
	assert.Equal(t, CommunicationHost{
		Protocol: "https",
		Host:     "example.live.dynatrace.com",
		Port:     443,
	}, ch)

	ch, err = NewCommunicationHost("https://managedhost.com:9999/here/communication")
	require.NoError(t, err)
	assert.Equal(t, CommunicationHost{
		Protocol: "https",
		Host:     "managedhost.com",
		Port:     9999,
	}, ch)

	ch, err = NewCommunicationHost("https://example.live.dynatrace.com/communication")
	require.NoError(t, err)
	assert.Equal(t, CommunicationHost{
		Protocol: "https",
		Host:     "example.live.dynatrace.com",
		Port:     443,
	}, ch)

	ch, err = NewCommunicationHost("https://10.0.0.1:8000/communication")
	require.NoError(t, err)
	assert.Equal(t, CommunicationHost{
		Protocol: "https",
		Host:     "10.0.0.1",
		Port:     8000,
	}, ch)

	ch, err = NewCommunicationHost("http://insecurehost/communication")
	require.NoError(t, err)
	assert.Equal(t, CommunicationHost{
		Protocol: "http",
		Host:     "insecurehost",
		Port:     80,
	}, ch)

	// Failures

	_, err = NewCommunicationHost("https://managedhost.com:notaport/here/communication")
	require.Error(t, err)

	_, err = NewCommunicationHost("example.live.dynatrace.com:80/communication")
	require.Error(t, err)

	_, err = NewCommunicationHost("ftp://randomhost.com:80/communication")
	require.Error(t, err)

	_, err = NewCommunicationHost("unix:///some/local/file")
	require.Error(t, err)

	_, err = NewCommunicationHost("shouldnotbeparsed")
	require.Error(t, err)
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
