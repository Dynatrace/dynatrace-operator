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

	dockerConfig := newDockerConfigWithAuth(connectionInfo.TenantUUID,
		string(r.token.Data[dtclient.DynatracePaasToken]),
		registry,
		r.buildAuthString(connectionInfo))

	return pullSecretDataFromDockerConfig(dockerConfig)
}

func (r *Reconciler) buildAuthString(connectionInfo dtclient.ConnectionInfo) string {
	auth := fmt.Sprintf("%s:%s", connectionInfo.TenantUUID, string(r.token.Data[dtclient.DynatracePaasToken]))
	return b64.StdEncoding.EncodeToString([]byte(auth))
}

func getImageRegistryFromAPIURL(apiURL string) (string, error) {
	r := strings.TrimPrefix(apiURL, "https://")
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
