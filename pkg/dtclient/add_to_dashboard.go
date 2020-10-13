package dtclient

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

type kubernetesCredentialPayload struct {
	Label                      string `json:"label"`
	EndpointUrl                string `json:"endpointUrl"`
	WorkloadIntegrationEnabled bool   `json:"workloadIntegrationEnabled"`
	EventsIntegrationEnabled   bool   `json:"eventsIntegrationEnabled"`
	AuthToken                  string `json:"authToken"`
	Active                     bool   `json:"active"`
	CertificateCheckEnabled    bool   `json:"certificateCheckEnabled"`
}

type kubernetesCredentialResponse struct {
	Id string `json:"id"`
}

func (dtc *dynatraceClient) AddToDashboard(label string, kubernetesApiEndpoint string, bearerToken string) (string, error) {
	if label == "" {
		return "", errors.New("no label given")
	}
	if kubernetesApiEndpoint == "" {
		return "", errors.New("no Kubernetes api endpoint given")
	}
	if bearerToken == "" {
		return "", errors.New("no bearer token given")
	}

	payload := kubernetesCredentialPayload{
		Label:                      label,
		EndpointUrl:                kubernetesApiEndpoint,
		AuthToken:                  bearerToken,
		WorkloadIntegrationEnabled: false,
		EventsIntegrationEnabled:   true,
		CertificateCheckEnabled:    true,
	}
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	dtEndpoint := fmt.Sprintf("%s/config/v1/kubernetes/credentials", dtc.url)
	req, err := http.NewRequest("POST", dtEndpoint, bytes.NewReader(jsonPayload))
	if err != nil {
		return "", fmt.Errorf("error initializing http request: %s", err.Error())
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", fmt.Sprintf("Api-Token %s", dtc.apiToken))

	res, err := dtc.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("error making post request to dynatrace api: %s", err.Error())
	}

	resData, err := dtc.getServerResponseData(res)
	if err != nil {
		return "", err
	}

	var resObject kubernetesCredentialResponse
	err = json.Unmarshal(resData, &resObject)
	if err != nil {
		return "", err
	}

	return resObject.Id, nil
}
