package dtclient

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const goodProcessModuleConfigResponse = `
{
	"revision": 1,
	"properties": [{
			"section": "general",
			"key": "field",
			"value": "test"
		},
		{
			"section": "test",
			"key": "a",
			"value": "b"
		}

	]
}
`

func TestCreateProcessModuleConfigRequest(t *testing.T) {
	dc := &dynatraceClient{
		paasToken: "token123",
	}
	require.NotNil(t, dc)

	req, err := dc.createProcessModuleConfigRequest(0)
	require.Nil(t, err)
	assert.Equal(t, "0", req.URL.Query().Get("revision"))
	assert.Contains(t, req.Header.Get("Authorization"), dc.paasToken)
}

func TestSpecialProcessModuleConfigRequestStatus(t *testing.T) {
	dc := &dynatraceClient{}
	require.NotNil(t, dc)

	assert.True(t, dc.specialProcessModuleConfigRequestStatus(&http.Response{StatusCode: http.StatusNotModified}))
	assert.True(t, dc.specialProcessModuleConfigRequestStatus(&http.Response{StatusCode: http.StatusNotFound}))
	assert.False(t, dc.specialProcessModuleConfigRequestStatus(&http.Response{StatusCode: http.StatusOK}))
	assert.False(t, dc.specialProcessModuleConfigRequestStatus(&http.Response{StatusCode: http.StatusInternalServerError}))
}

func TestReadResponseForProcessModuleConfig(t *testing.T) {
	dc := &dynatraceClient{}
	require.NotNil(t, dc)

	processConfig, err := dc.readResponseForProcessModuleConfig([]byte(goodProcessModuleConfigResponse))
	require.Nil(t, err)
	assert.Equal(t, uint(1), processConfig.Revision)
	require.Len(t, processConfig.Properties, 2)
	assert.Equal(t, "general", processConfig.Properties[0].Section)
	assert.Equal(t, "field", processConfig.Properties[0].Key)
	assert.Equal(t, "test", processConfig.Properties[0].Value)
	assert.Equal(t, "test", processConfig.Properties[1].Section)
	assert.Equal(t, "a", processConfig.Properties[1].Key)
	assert.Equal(t, "b", processConfig.Properties[1].Value)
}
