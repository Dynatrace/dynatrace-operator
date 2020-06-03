package dtclient

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

const (
	MarkedForTerminationEvent = "MARKED_FOR_TERMINATION"
)

// EventData struct which defines what event payload should contain
type EventData struct {
	EventType     string               `json:"eventType"`
	StartInMillis uint64               `json:"start"`
	EndInMillis   uint64               `json:"end"`
	Description   string               `json:"description"`
	AttachRules   EventDataAttachRules `json:"attachRules"`
	Source        string               `json:"source"`
}

type EventDataAttachRules struct {
	EntityIDs []string `json:"entityIds"`
}

func (dc *dynatraceClient) SendEvent(eventData *EventData) error {
	if eventData == nil {
		return errors.New("no data found in eventData payload")
	}

	if eventData.EventType == "" {
		return errors.New("no key set for eventType in eventData payload")
	}

	jsonStr, err := json.Marshal(eventData)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/v1/events", dc.url)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
	if err != nil {
		return fmt.Errorf("error initialising http request: %s", err.Error())
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", fmt.Sprintf("Api-Token %s", dc.apiToken))

	response, err := dc.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("error making post request to dynatrace api: %s", err.Error())
	}

	_, err = dc.getServerResponseData(response)
	return err
}
