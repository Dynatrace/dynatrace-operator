package dynatrace

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Dynatrace/dynatrace-operator/pkg/clients/utils"
	"github.com/pkg/errors"
)

const (
	MarkedForTerminationEvent = "MARKED_FOR_TERMINATION"
)

// EventData struct which defines what event payload should contain
type EventData struct {
	EventType     string               `json:"eventType"`
	Description   string               `json:"description"`
	Source        string               `json:"source"`
	AttachRules   EventDataAttachRules `json:"attachRules"`
	StartInMillis uint64               `json:"start"`
	EndInMillis   uint64               `json:"end"`
}

type EventDataAttachRules struct {
	EntityIDs []string `json:"entityIds"`
}

func (dtc *dynatraceClient) SendEvent(ctx context.Context, eventData *EventData) error {
	if eventData == nil {
		return errors.New("no data found in eventData payload")
	}

	if eventData.EventType == "" {
		return errors.New("no key set for eventType in eventData payload")
	}

	jsonStr, err := json.Marshal(eventData)
	if err != nil {
		return errors.WithStack(err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, dtc.getEventsUrl(), bytes.NewBuffer(jsonStr))
	if err != nil {
		return fmt.Errorf("error initializing http request: %w", err)
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", ApiTokenHeader+dtc.apiToken)

	response, err := dtc.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("error making post request to dynatrace api: %w", err)
	}

	defer utils.CloseBodyAfterRequest(response)

	_, err = dtc.getServerResponseData(response)

	return errors.WithStack(err)
}
