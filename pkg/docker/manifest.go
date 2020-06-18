package docker

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

type Config struct {
	Digest string
}

type Manifest struct {
	Config Config
}

func (registry *Registry) GetManifest(digest string) (*Manifest, error) {
	request, err := registry.prepareRequest(registry.buildUrl(digest))
	if err != nil {
		return nil, err
	}
	return getManifest(request)
}

func (registry *Registry) GetLatestManifest() (*Manifest, error) {
	return registry.GetManifest(Latest)
}

func getManifest(request *http.Request) (*Manifest, error) {
	request.Header.Add("Accept", "application/vnd.docker.distribution.manifest.v2+json")
	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer func() {
		//Ignore error because there is nothing one could do here
		_ = response.Body.Close()
	}()

	switch response.StatusCode {
	case 200:
		return parseManifest(response)
	case 201:
		return parseManifest(response)
	default:
		return nil, fmt.Errorf("unexpected response: code: %d, status: %s", response.StatusCode, response.Status)
	}
}

func parseManifest(response *http.Response) (*Manifest, error) {
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	var manifest Manifest
	err = json.Unmarshal(body, &manifest)
	if err != nil {
		return nil, err
	}
	return &manifest, err
}

func (registry *Registry) buildUrl(digest string) string {
	image := registry.Image
	if registry.Server == "" {
		registry.Server = DockerHubApiServer
		image = "library/" + image
	}
	return fmt.Sprintf(UrlTemplate, registry.Server, image, digest)
}
