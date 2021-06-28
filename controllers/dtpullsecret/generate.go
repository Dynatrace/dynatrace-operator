package dtpullsecret

import (
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/Dynatrace/dynatrace-operator/dtclient"
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

	if r.token == nil {
		return nil, fmt.Errorf("token secret is nil, cannot generate docker config")
	}

	paasToken, hasToken := r.token.Data[dtclient.DynatracePaasToken]
	if !hasToken {
		return nil, fmt.Errorf("token secret does not contain a paas token, cannot generate docker config")
	}

	dockerConfig := newDockerConfigWithAuth(connectionInfo.TenantUUID,
		string(paasToken),
		registry,
		r.buildAuthString(connectionInfo))

	return pullSecretDataFromDockerConfig(dockerConfig)
}

func (r *Reconciler) buildAuthString(connectionInfo dtclient.ConnectionInfo) string {
	paasToken := ""
	if r.token != nil {
		paasToken = string(r.token.Data[dtclient.DynatracePaasToken])
	}

	auth := fmt.Sprintf("%s:%s", connectionInfo.TenantUUID, paasToken)
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
