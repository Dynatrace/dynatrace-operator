package dtclient

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type dashboardHandler struct {
}

func (handler *dashboardHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	if request.Method == "POST" && request.URL.Path == "/config/v1/kubernetes/credentials" {
		resObject := kubernetesCredentialResponse{
			Id: "an id",
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

func TestDynatraceClient_AddToDashboard(t *testing.T) {
	dynatraceServer, _ := createTestDynatraceClient(t, &dashboardHandler{})
	defer dynatraceServer.Close()

	dtc := dynatraceClient{
		logger:     log.Log.WithName("dtc"),
		apiToken:   apiToken,
		paasToken:  paasToken,
		httpClient: dynatraceServer.Client(),
		url:        dynatraceServer.URL,
	}

	response, err := dtc.AddToDashboard("a label", "an endpoint", "a token")
	assert.NoError(t, err)
	assert.Equal(t, "an id", response)
}
