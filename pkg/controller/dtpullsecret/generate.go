package dtpullsecret

import (
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/dtclient"
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

	paasToken := ""
	if r.token != nil {
		paasToken = string(r.token.Data[dtclient.DynatracePaasToken])
	}

	dockerConfig := newDockerConfigWithAuth(connectionInfo.TenantUUID,
		paasToken,
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
	r = strings.TrimPrefix(apiURL, "http://")
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
