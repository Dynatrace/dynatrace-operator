package dtpullsecret

import (
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"

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

func (r *Reconciler) GenerateData() (map[string][]byte, error) {
	var registryToken string

	registry, err := getImageRegistryFromAPIURL(r.dk.Spec.APIURL)
	if err != nil {
		return nil, err
	}

	switch {
	case r.tokens.PaasToken().Value != "":
		registryToken = r.tokens.PaasToken().Value
	case r.tokens.ApiToken().Value != "":
		registryToken = r.tokens.ApiToken().Value
	default:
		return nil, errors.New("token secret does not contain a paas or api token, cannot generate docker config")
	}

	tenantUUID, err := r.dk.TenantUUID()
	if err != nil {
		return nil, errors.WithMessage(err, "cannot generate docker config")
	}

	dockerCfg := newDockerConfigWithAuth(tenantUUID,
		registryToken,
		registry,
		r.buildAuthString(tenantUUID, registryToken))

	return pullSecretDataFromDockerConfig(dockerCfg)
}

func (r *Reconciler) buildAuthString(tenantUUID string, registryToken string) string {
	auth := fmt.Sprintf("%s:%s", tenantUUID, registryToken)

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
