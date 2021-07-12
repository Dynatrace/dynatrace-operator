package testing_utils

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

func AssertEvents(t *testing.T, eventsCh chan string, expectedEvents Events) {
	close(eventsCh)
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
