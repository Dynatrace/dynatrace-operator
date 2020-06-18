package docker

import (
	"fmt"
	"net/http"
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

	switch registry.Server {
	case DockerHubApiServer:
		token, err := registry.getDockerHubToken()
		if err != nil {
			return nil, err
		}

		request.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
	case GcrApiServer:
		token, err := registry.getGcrToken()
		if err != nil {
			return nil, err
		}

		request.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
	default:
		request.SetBasicAuth(registry.Username, registry.Password)
	}
	return request, nil
}

const (
	Get  = "GET"
	Post = "POST"

	Latest      = "latest"
	UrlTemplate = "https://%s/v2/%s/manifests/%s"

	DockerHubApiServer = "registry-1.docker.io"
	DockerHubTokenUrl  = "https://auth.docker.io/token?service=registry.docker.io&scope=repository:%s:pull"

	GcrApiServer   = "gcr.io"
	GcrApiTokenUrl = "https://gcr.io/v2/token?scope=repository:%s:pull"
)
