package dtclient

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

type dashboardHandler struct {
}

func (handler *dashboardHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	if request.Method == "POST" && request.URL.Path == "/v2/settings/objects" {
		resObject := []postObjectsResponse{
			{
				ObjectId: "an id",
			},
		}
		resJson, err := json.Marshal(resObject)
		if err != nil {
			writer.WriteHeader(500)
			_, _ = writer.Write([]byte(err.Error()))
		}
		_, _ = writer.Write(resJson)
	} else {
		writer.WriteHeader(400)
		_, _ = writer.Write([]byte{})
	}
}

func TestDynatraceClient_CreateSetting(t *testing.T) {
	dynatraceServer, _ := createTestDynatraceClient(t, &dashboardHandler{})
	defer dynatraceServer.Close()

	dtc := dynatraceClient{
		apiToken:   apiToken,
		paasToken:  paasToken,
		httpClient: dynatraceServer.Client(),
		url:        dynatraceServer.URL,
	}

	response, err := dtc.CreateSetting("a label", "1234")
	assert.NoError(t, err)
	assert.Equal(t, "an id", response)
}
