package dtclient

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const goodRuxitProcConfResponse = `
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

func TestCreateRuxitProcConfRequest(t *testing.T) {
	dc := &dynatraceClient{
		paasToken: "token123",
		logger:    consoleLogger,
	}
	require.NotNil(t, dc)

	req, err := dc.createRuxitProcConfRequest(0)
	require.Nil(t, err)
	assert.Equal(t, "0", req.URL.Query().Get("revision"))
	assert.Contains(t, req.Header.Get("Authorization"), dc.paasToken)
}

func TestSpecialRuxitProcConfRequestStatus(t *testing.T) {
	dc := &dynatraceClient{
		logger: consoleLogger,
	}
	require.NotNil(t, dc)

	assert.True(t, dc.specialRuxitProcConfRequestStatus(&http.Response{StatusCode: http.StatusNotModified}))
	assert.True(t, dc.specialRuxitProcConfRequestStatus(&http.Response{StatusCode: http.StatusNotFound}))
	assert.False(t, dc.specialRuxitProcConfRequestStatus(&http.Response{StatusCode: http.StatusOK}))
	assert.False(t, dc.specialRuxitProcConfRequestStatus(&http.Response{StatusCode: http.StatusInternalServerError}))
}

func TestReadResponseForRuxitProcConf(t *testing.T) {
	dc := &dynatraceClient{
		logger: consoleLogger,
	}
	require.NotNil(t, dc)

	ruxitProc, err := dc.readResponseForRuxitProcConf([]byte(goodRuxitProcConfResponse))
	require.Nil(t, err)
	assert.Equal(t, uint(1), ruxitProc.Revision)
	require.Len(t, ruxitProc.Properties, 2)
	assert.Equal(t, "general", ruxitProc.Properties[0].Section)
	assert.Equal(t, "field", ruxitProc.Properties[0].Key)
	assert.Equal(t, "test", ruxitProc.Properties[0].Value)
	assert.Equal(t, "test", ruxitProc.Properties[1].Section)
	assert.Equal(t, "a", ruxitProc.Properties[1].Key)
	assert.Equal(t, "b", ruxitProc.Properties[1].Value)
}
