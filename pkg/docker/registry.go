package docker

import (
	"fmt"
	"net/http"
	"strings"
)

type Registry struct {
	Server   string
	Image    string
	Username string
	Password string
}

func RegistryFromImage(image string) *Registry {
	var registry Registry

	image = trimTag(image)

	for _, supportedRegistry := range GetSupportedRegistries() {
		if strings.Contains(image, supportedRegistry) {
			registry.Server = strings.Split(image, "/")[0]
			registry.Image = strings.TrimPrefix(image, registry.Server+"/")
			return &registry
		}
	}

	registry.Server = DockerHubApiServer
	registry.Image = image
	return &registry
}

func trimTag(image string) string {
	urlParts := strings.Split(image, "/")
	if len(urlParts) > 0 {
		name := urlParts[len(urlParts)-1]
		nameParts := strings.Split(name, ":")
		if len(nameParts) > 1 {
			return strings.TrimSuffix(image, ":"+nameParts[len(nameParts)-1])
		}
	}

	return image
}

func (registry *Registry) prepareRequest(url string) (*http.Request, error) {
	request, err := http.NewRequest(Get, url, nil)
	if err != nil {
		return nil, err
	}

	if strings.Contains(registry.Server, DockerHubApiServer) {
		token, err := registry.getDockerHubToken()
		if err != nil {
			return nil, err
		}

		request.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
	} else if strings.Contains(registry.Server, GcrApiServer) {
		token, err := registry.getGcrToken()
		if err != nil {
			return nil, err
		}

		request.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
	} else if strings.Contains(registry.Server, RhccApiServer) {
		token, err := registry.getRhccToken()
		if err != nil {
			return nil, err
		}

		request.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
	} else if strings.Contains(registry.Server, QuayApiServer) {
		token, err := registry.getQuayToken()
		if err != nil {
			return nil, err
		}

		request.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
	} else if strings.Contains(registry.Server, AmazonAws) {
		request.SetBasicAuth(registry.Username, registry.Password)
	} else {
		return nil, fmt.Errorf("unsupported registry")
	}

	return request, nil
}

//GetSupportedRegistries
//Returns supported registries. Omits DockerHub, because registry defaults to it
func GetSupportedRegistries() []string {
	return []string{GcrApiServer, RhccApiServer, QuayApiServer, AmazonAws}
}

const (
	Get  = "GET"
	Post = "POST"

	Latest      = "latest"
	UrlTemplate = "https://%s/v2/%s/manifests/%s"

	DockerHubTokenUrl  = "https://auth.docker.io/token?service=registry.docker.io&scope=repository:%s:pull"
	DockerHubApiServer = "registry-1.docker.io"

	GcrTokenUrl  = "https://gcr.io/v2/token?scope=repository:%s:pull"
	GcrApiServer = "gcr.io"

	RhccTokenUrl  = "https://registry.connect.redhat.com/auth/realms/rhc4tp/protocol/redhat-docker-v2/auth?service=docker-registry"
	RhccApiServer = "registry.connect.redhat.com"

	QuayTokenUrl  = "https://quay.io/v2/auth?service=quay.io"
	QuayApiServer = "quay.io"

	AmazonAws = "amazonaws.com"
)
