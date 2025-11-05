package dynatrace

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetEntityIDForIP(t *testing.T) {
	ctx := context.Background()

	dynatraceServer, _ := createTestDynatraceServer(t, &ipHandler{}, "")
	defer dynatraceServer.Close()

	dtc := dynatraceClient{
		apiToken:   apiToken,
		paasToken:  paasToken,
		httpClient: dynatraceServer.Client(),
		url:        dynatraceServer.URL,
	}
	require.NoError(t, dtc.setHostCacheFromResponse([]byte(
		fmt.Sprintf(`[
	{
		"entityId": "HOST-42",
		"displayName": "A",
		"firstSeenTimestamp": 1589940921731,
		"lastSeenTimestamp": %v,
		"ipAddresses": [
			"1.1.1.1"
		],
		"monitoringMode": "FULL_STACK",
		"networkZoneId": "default",
		"agentVersion": {
			"major": 1,
			"minor": 195,
			"revision": 0,
			"timestamp": "20200515-045253",
			"sourceRevision": ""
		}
	}
]`, time.Now().UTC().Unix()*1000))))

	id, err := dtc.GetHostEntityIDForIP(ctx, "1.1.1.1")
	require.NoError(t, err)
	assert.NotEmpty(t, id)
	assert.Equal(t, "HOST-42", id)

	id, err = dtc.GetHostEntityIDForIP(ctx, "2.2.2.2")

	require.Error(t, err)
	assert.Empty(t, id)

	require.NoError(t, dtc.setHostCacheFromResponse([]byte(
		fmt.Sprintf(`[
	{
		"entityId": "",
		"displayName": "A",
		"firstSeenTimestamp": 1589940921731,
		"lastSeenTimestamp": %v,
		"ipAddresses": [
			"1.1.1.1"
		],
		"monitoringMode": "FULL_STACK",
		"networkZoneId": "default",
		"agentVersion": {
			"major": 1,
			"minor": 195,
			"revision": 0,
			"timestamp": "20200515-045253",
			"sourceRevision": ""
		}
	}
]`, time.Now().UTC().Unix()*1000))))

	id, err = dtc.GetHostEntityIDForIP(ctx, "1.1.1.1")

	require.Error(t, err)
	assert.Empty(t, id)
}
