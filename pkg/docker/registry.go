package docker

import "net/http"

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
	request.SetBasicAuth(registry.Username, registry.Password)
	return request, nil
}
