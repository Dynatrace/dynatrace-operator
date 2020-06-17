package docker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

type Registry struct {
	Server   string
	Image    string
	Username string
	Password string
}

func (registry *Registry) prepareRequest(url string) (*http.Request, error) {
	request, err := http.NewRequest(Get, url, nil)
	if err != nil {
		return nil, err
	}

	if registry.Server == DockerApi {
		token, err := registry.getDockerHubToken()
		if err != nil {
			return nil, err
		}

		request.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
	} else {
		request.SetBasicAuth(registry.Username, registry.Password)
	}
	return request, nil
}

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

const (
	Get  = "GET"
	Post = "POST"

	Latest      = "latest"
	UrlTemplate = "https://%s/v2/%s/manifests/%s"
	DockerHub   = "hub.docker.com"
	DockerApi   = "registry-1.docker.io"

	DockerHubScopes   = "realm=\"https://auth.docker.io\",service=\"registry-1.docker.io\",scope=\"repository:manifests/library/alpine:pull\""
	DockerHubTokenUrl = "https://auth.docker.io/token?service=registry.docker.io&scope=repository:%s:pull"
)
