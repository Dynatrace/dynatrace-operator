package dynatrace

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testActiveGateVersionGetLatestActiveGateVersion(t *testing.T, dynatraceClient Client) {
	{
		latestAgentVersion, err := dynatraceClient.GetLatestActiveGateVersion(OsUnix)

		require.NoError(t, err)
		assert.Equal(t, "1.242.0.20220429-180918", latestAgentVersion, "latest agent version equals expected version")
	}
}

func handleLatestActiveGateVersion(request *http.Request, writer http.ResponseWriter) {
	switch request.Method {
	case http.MethodGet:
		writer.WriteHeader(http.StatusOK)

		out, _ := json.Marshal(map[string]string{"latestGatewayVersion": "1.242.0.20220429-180918"})
		_, _ = writer.Write(out)
	default:
		writeError(writer, http.StatusMethodNotAllowed)
	}
}
