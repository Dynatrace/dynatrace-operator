package docker

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

func (registry *Registry) getToken(url string) ([]byte, error) {
	image := registry.Image
	if url == DockerHubTokenUrl {
		if !strings.Contains(image, "/") {
			image = "library/" + image
		}
	}
	url = fmt.Sprintf(url, image)
	if strings.Contains(url, FormatError) {
		//Remove format error if url does not need image name
		url = url[:strings.LastIndex(url, FormatError)]
	}

	request, err := http.NewRequest(
		Get, url, nil)
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

	if response.StatusCode != 200 {
		return nil, fmt.Errorf("invalid credentials")
	}

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
	FormatError = "%!(EXTRA"
)
