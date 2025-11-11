package dynatrace

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEventDataMarshal(t *testing.T) {
	testJSONInput := []byte(`{
		"eventType": "MARKED_FOR_TERMINATION",
		"start": 20,
		"end": 20,
		"description": "K8s node was marked unschedulable. Node is likely being drained",
		"attachRules": {
			"entityIds": [ "HOST-CA78D78BBC6687D3" ]
		},
		"source": "OneAgent Operator"
	}`)

	var testEventData EventData
	err := json.Unmarshal(testJSONInput, &testEventData)
	require.NoError(t, err)
	assert.Equal(t, "MARKED_FOR_TERMINATION", testEventData.EventType)
	assert.ElementsMatch(t, testEventData.AttachRules.EntityIDs, []string{"HOST-CA78D78BBC6687D3"})
	assert.Equal(t, "OneAgent Operator", testEventData.Source)

	jsonBuffer, err := json.Marshal(testEventData)
	require.NoError(t, err)
	assert.JSONEq(t, string(jsonBuffer), string(testJSONInput))
}

func TestSendEvent(t *testing.T) {
	ctx := context.Background()
	empty := EventData{}
	eventTypeOnly := EventData{
		EventType: "abcd",
	}

	t.Run("SendEvent no event data", func(t *testing.T) {
		dynatraceServer, dynatraceClient := createTestDynatraceServer(t, sendEventHandlerStub(), "")
		defer dynatraceServer.Close()

		err := dynatraceClient.SendEvent(ctx, nil)
		require.Error(t, err)
		assert.Equal(t, "no data found in eventData payload", err.Error())
	})
	t.Run("SendEvent incomplete event data", func(t *testing.T) {
		dynatraceServer, dynatraceClient := createTestDynatraceServer(t, sendEventHandlerStub(), "")
		defer dynatraceServer.Close()

		err := dynatraceClient.SendEvent(ctx, &empty)
		require.Error(t, err)
		assert.Equal(t, "no key set for eventType in eventData payload", err.Error())

		err = dynatraceClient.SendEvent(ctx, &eventTypeOnly)
		require.NoError(t, err)
	})
	t.Run("SendEvent request error", func(t *testing.T) {
		dynatraceServer, dynatraceClient := createTestDynatraceServer(t, sendEventHandlerError(), "")

		err := dynatraceClient.SendEvent(ctx, &empty)
		require.Error(t, err)
		assert.Equal(t, "no key set for eventType in eventData payload", err.Error())

		err = dynatraceClient.SendEvent(ctx, &eventTypeOnly)
		require.Error(t, err)
		assert.Equal(t, "dynatrace server error 500: error received from server", err.Error())

		dynatraceServer.Close()

		err = dynatraceClient.SendEvent(ctx, &eventTypeOnly)
		require.Error(t, err)
		assert.True(t,
			// Reason differs between local tests and travis test, so only check main error message
			strings.HasPrefix(err.Error(),
				"error making post request to dynatrace api: Post \""+
					dynatraceServer.URL+
					"/v1/events\""))
	})

	t.Run("SendEvent 404 request error", func(t *testing.T) {
		testValidEventData := []byte(`{
			"eventType": "MARKED_FOR_TERMINATION",
			"start": 20,
			"end": 20,
			"description": "K8s node was marked unschedulable. Node is likely being drained",
			"attachRules": {
				"entityIds": [ "HOST-CA78D78BBC6687D3" ]
			},
			"source": "OneAgent Operator"
		}`)

		var testEventData EventData
		err := json.Unmarshal(testValidEventData, &testEventData)
		require.NoError(t, err)

		dynatraceServer, dynatraceClient := createTestDynatraceServer(t, sendEventHandler404Error(), "")

		err = dynatraceClient.SendEvent(ctx, &testEventData)
		require.ErrorAs(t, err, &V1EventsAPINotAvailableErr{})

		dynatraceServer.Close()
	})
}

func sendEventHandlerStub() http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {}
}

func sendEventHandlerError() http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		writeError(writer, http.StatusInternalServerError)
	}
}

func sendEventHandler404Error() http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		writeError(writer, http.StatusNotFound)
	}
}

func testSendEvent(t *testing.T, dynatraceClient Client) {
	ctx := context.Background()

	t.Run("happy path", func(t *testing.T) {
		testValidEventData := []byte(`{
			"eventType": "MARKED_FOR_TERMINATION",
			"start": 20,
			"end": 20,
			"description": "K8s node was marked unschedulable. Node is likely being drained",
			"attachRules": {
				"entityIds": [ "HOST-CA78D78BBC6687D3" ]
			},
			"source": "OneAgent Operator"
		}`)

		var testEventData EventData
		err := json.Unmarshal(testValidEventData, &testEventData)
		require.NoError(t, err)

		err = dynatraceClient.SendEvent(ctx, &testEventData)
		require.NoError(t, err)
	})
	t.Run("invalid event type sent -> error from API", func(t *testing.T) {
		testInvalidEventData := []byte(`{
			"start": 20,
			"end": 20,
			"description": "K8s node was marked unschedulable. Node is likely being drained",
			"attachRules": {
				"entityIds": [ "HOST-CA78D78BBC6687D3" ]
			},
			"source": "OneAgent Operator"
		}`)

		var testEventData EventData
		err := json.Unmarshal(testInvalidEventData, &testEventData)
		require.NoError(t, err)

		err = dynatraceClient.SendEvent(ctx, &testEventData)
		require.Error(t, err, "no eventType set")
	})
	t.Run("extra keys are ignored", func(t *testing.T) {
		testExtraKeysEventData := []byte(`{
			"eventType": "MARKED_FOR_TERMINATION",
			"start": 20,
			"end": 20,
			"description": "K8s node was marked unschedulable. Node is likely being drained",
			"attachRules": {
				"entityIds": [ "HOST-CA78D78BBC6687D3" ]
			},
			"source": "OneAgent Operator",
		 	"cat": "potato"
		}`)

		var testEventData EventData
		err := json.Unmarshal(testExtraKeysEventData, &testEventData)
		require.NoError(t, err)

		err = dynatraceClient.SendEvent(ctx, &testEventData)
		require.NoError(t, err)
	})
}

func handleSendEvent(request *http.Request, writer http.ResponseWriter) {
	eventPostResponse := []byte(`{
		"storedEventIds": [1],
		"storedIds": ["string"],
		"storedCorrelationIds": ["string"]}`)

	switch request.Method {
	case http.MethodPost:
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write(eventPostResponse)
	default:
		writeError(writer, http.StatusMethodNotAllowed)
	}
}
