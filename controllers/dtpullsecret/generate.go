package dtpullsecret

import (
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/dtclient"
)

type dockerAuthentication struct {
	Username string
	Password string
	Auth     string
}

type dockerConfig struct {
	Auths map[string]dockerAuthentication
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
	connectionInfo, err := r.dtc.GetConnectionInfo()
	if err != nil {
		return nil, err
	}

	registry, err := getImageRegistryFromAPIURL(r.instance.Spec.APIURL)
	if err != nil {
		return nil, err
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
	r := strings.TrimPrefix(apiURL, "https://")
	r = strings.TrimPrefix(r, "http://")
	r = strings.TrimSuffix(r, "/api")
	return r, nil
}

func pullSecretDataFromDockerConfig(dockerConf *dockerConfig) (map[string][]byte, error) {
	dockerConfJson, err := json.Marshal(dockerConf)
	if err != nil {
		return nil, err
	}
	return map[string][]byte{".dockerconfigjson": dockerConfJson}, nil
}
