package dtpullsecret

import (
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/token"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/pkg/errors"
)

const (
	DockerConfigJson = ".dockerconfigjson"
)

type dockerAuthentication struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Auth     string `json:"auth"`
}

type dockerConfig struct {
	Auths map[string]dockerAuthentication `json:"auths"`
}

func newDockerConfigWithAuth(username string, password string, registry string, auth string) *dockerConfig {
	return &dockerConfig{
		Auths: map[string]dockerAuthentication{
			registry: {
				Username: username,
				Password: password,
				Auth:     auth,
			},
		},
	}
}

func (r *Reconciler) GenerateData(tokens token.Tokens) (map[string][]byte, error) {
	var registryToken string

	connectionInfo := r.dynakube.ConnectionInfo()
	registry, err := getImageRegistryFromAPIURL(r.dynakube.Spec.APIURL)
	if err != nil {
		return nil, err
	}

	if tokens.PaasToken().Value != "" {
		registryToken = tokens.PaasToken().Value
	} else if tokens.ApiToken().Value != "" {
		registryToken = tokens.ApiToken().Value
	} else {
		return nil, errors.New("token secret does not contain a paas or api token, cannot generate docker config")
	}

	dockerCfg := newDockerConfigWithAuth(connectionInfo.TenantUUID,
		registryToken,
		registry,
		r.buildAuthString(connectionInfo, registryToken))

	return pullSecretDataFromDockerConfig(dockerCfg)
}

func (r *Reconciler) buildAuthString(connectionInfo dtclient.OneAgentConnectionInfo, registryToken string) string {
	auth := fmt.Sprintf("%s:%s", connectionInfo.TenantUUID, registryToken)
	return b64.StdEncoding.EncodeToString([]byte(auth))
}

func getImageRegistryFromAPIURL(apiURL string) (string, error) {
	u, err := url.Parse(apiURL)
	if err != nil {
		return "", errors.WithStack(err)
	}
	return u.Host, nil
}

func pullSecretDataFromDockerConfig(dockerConf *dockerConfig) (map[string][]byte, error) {
	dockerConfJson, err := json.Marshal(dockerConf)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return map[string][]byte{DockerConfigJson: dockerConfJson}, nil
}
