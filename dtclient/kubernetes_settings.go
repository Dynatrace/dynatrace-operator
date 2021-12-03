package dtclient

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

type kubernetesSettingsPayload struct {
	Label                           string `json:"label"`
	ClusterIdEnabled                bool   `json:"clusterIdEnabled"`
	ClusterId                       string `json:"clusterId"`
	CloudApplicationPipelineEnabled bool   `json:"cloudApplicationPipelineEnabled"`
	OpenMetricsPipelineEnabled      bool   `json:"openMetricsPipelineEnabled"`
	Enabled                         bool   `json:"enabled"`
	EventProcessingActive           bool   `json:"eventProcessingActive"`
	EventProcessingV2Active         bool   `json:"eventProcessingV2Active"`
	FilterEvents                    bool   `json:"filterEvents"`
}

type postObjectsPayload struct {
	SchemaId      string                    `json:"schemaId"`
	SchemaVersion string                    `json:"schemaVersion"`
	Value         kubernetesSettingsPayload `json:"value"`
}

type postObjectsResponse struct {
	ObjectId string `json:"objectId"`
}

func (dtc *dynatraceClient) CreateSetting(label string, kubeSystemUUID string) (string, error) {
	if label == "" {
		return "", errors.New("no label given")
	}
	if kubeSystemUUID == "" {
		return "", errors.New("no kube-system namespace UUID given")
	}

	payload := []postObjectsPayload{
		{
			SchemaId:      "builtin:cloud.kubernetes",
			SchemaVersion: "1.0.27",
			Value: kubernetesSettingsPayload{
				Label:                           label,
				ClusterIdEnabled:                true,
				ClusterId:                       kubeSystemUUID,
				CloudApplicationPipelineEnabled: true,
				OpenMetricsPipelineEnabled:      true,
				EventProcessingActive:           true,
				Enabled:                         true,
				FilterEvents:                    false,
				EventProcessingV2Active:         true,
			},
		},
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	dtEndpoint := fmt.Sprintf("%s/v2/settings/objects?validateOnly=false", dtc.url)

	req, err := http.NewRequest("POST", dtEndpoint, bytes.NewReader(jsonPayload))
	if err != nil {
		return "", fmt.Errorf("error initializing http request: %s", err.Error())
	}
	req.Header.Add("Content-Type", "application/json")

	req.Header.Add("Accept", "application/json")
	req.Header.Add("Authorization", fmt.Sprintf("Api-Token %s", dtc.apiToken))

	res, err := dtc.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("error making post request to dynatrace api: %s", err.Error())
	}

	resData, err := dtc.getServerResponseData(res)
	if err != nil {
		return "", err
	}

	fmt.Printf("got response from creating setting  %s \n %s", resData, dtEndpoint)
	var response []postObjectsResponse
	err = json.Unmarshal(resData, &response)
	if err != nil {
		return "", err
	}

	if len(response) != 1 {
		return "", fmt.Errorf("response is not containing exactly one entry %s", resData)
	}

	return response[0].ObjectId, nil
}
