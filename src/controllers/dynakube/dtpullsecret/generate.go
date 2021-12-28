package dtpullsecret

import (
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/pkg/errors"
)

const (
	dockerConfigJson = ".dockerconfigjson"
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

func (r *Reconciler) GenerateData() (map[string][]byte, error) {
	connectionInfo := r.instance.ConnectionInfo()
	registry, err := getImageRegistryFromAPIURL(r.instance.Spec.APIURL)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	if r.paasToken == "" {
		if r.apiToken != "" {
			r.paasToken = r.apiToken
		} else {
			return nil, fmt.Errorf("token secret does not contain a paas or api token, cannot generate docker config")
		}
	}

	dockerConfig := newDockerConfigWithAuth(connectionInfo.TenantUUID,
		string(r.paasToken),
		registry,
		r.buildAuthString(connectionInfo))

	return pullSecretDataFromDockerConfig(dockerConfig)
}

func (r *Reconciler) buildAuthString(connectionInfo dtclient.ConnectionInfo) string {
	auth := fmt.Sprintf("%s:%s", connectionInfo.TenantUUID, r.paasToken)
	return b64.StdEncoding.EncodeToString([]byte(auth))
}

func getImageRegistryFromAPIURL(apiURL string) (string, error) {
	u, err := url.Parse(apiURL)
	if err != nil {
		return "", err
	}
	return u.Host, nil
}

func pullSecretDataFromDockerConfig(dockerConf *dockerConfig) (map[string][]byte, error) {
	dockerConfJson, err := json.Marshal(dockerConf)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return map[string][]byte{dockerConfigJson: dockerConfJson}, nil
}
