package testing

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

type Event struct {
	EventType string
	Reason    string
	Message   string
}
type Events []Event

// FakeEventRecorders(aka.: FakeRecorders) push the "sent" events into a string channel, using this format: "eventType eventReason eventMessage"
// So this function just parses this format and compares the produced fields with the fields of the provided Event structs IN ORDER.
// In case of the Event.Message field it only checks if it "CONTAINED" in sent event.
func AssertEvents(t *testing.T, eventsCh chan string, expectedEvents Events) {
	assert.Equal(t, len(eventsCh), len(expectedEvents))
	for _, event := range expectedEvents {
		eventString := <-eventsCh
		assert.NotNil(t, eventString)
		tmp := strings.Split(eventString, " ")
		eventType := tmp[0]
		reason := tmp[1]
		message := strings.Join(tmp[2:], " ")
		assert.Equal(t, eventType, event.EventType)
		assert.Equal(t, reason, event.Reason)
		if len(event.Message) > 0 {
			assert.Contains(t, message, event.Message)
		}
	}
}
