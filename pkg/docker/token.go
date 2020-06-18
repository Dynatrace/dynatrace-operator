package docker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

func (registry *Registry) getDockerHubToken() ([]byte, error) {
	credentials, err := registry.buildJsonCredentials()
	if err != nil {
		return nil, err
	}

	image := registry.Image
	if !strings.Contains(image, "/") {
		image = "library/" + image
	}

	request, err := http.NewRequest(
		Get, fmt.Sprintf(DockerHubTokenUrl, image), bytes.NewReader(credentials))
	if err != nil {
		return nil, err
	}

	return requestToken(request)
}

func (registry *Registry) getGcrToken() ([]byte, error) {
	request, err := http.NewRequest(
		Get, fmt.Sprintf(GcrApiTokenUrl, registry.Image), nil)
	if err != nil {
		return nil, err
	}

	request.SetBasicAuth(registry.Username, registry.Password)
	return requestToken(request)
}

func requestToken(request *http.Request) ([]byte, error) {
	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer func() {
		//Ignore error because there is nothing one could do here
		_ = response.Body.Close()
	}()

	responseData, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	tokenResponse := struct {
		Token string
	}{}
	err = json.Unmarshal(responseData, &tokenResponse)
	if err != nil {
		return nil, err
	}

	return []byte(tokenResponse.Token), nil
}

func (registry *Registry) buildJsonCredentials() ([]byte, error) {
	credentials := struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}{
		Username: registry.Username,
		Password: registry.Password,
	}

	data, err := json.Marshal(&credentials)
	if err != nil {
		return nil, err
	}

	return data, nil
}
