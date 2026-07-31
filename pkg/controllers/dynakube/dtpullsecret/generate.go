// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package dtpullsecret

import (
	b64 "encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/token"
	"github.com/pkg/errors"
)

const (
	DockerConfigJSON = ".dockerconfigjson"
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

func generateData(dk *dynakube.DynaKube, tokens token.Tokens) (map[string][]byte, error) {
	var registryToken string

	registry := dk.APIURLHost()

	switch {
	case tokens.PaasToken().Value != "":
		registryToken = tokens.PaasToken().Value
	case tokens.APIToken().Value != "":
		registryToken = tokens.APIToken().Value
	default:
		return nil, errors.New("token secret does not contain a paas or api token, cannot generate docker config")
	}

	tenantUUID, err := dk.TenantUUID()
	if err != nil {
		return nil, errors.WithMessage(err, "cannot generate docker config")
	}

	dockerCfg := newDockerConfigWithAuth(tenantUUID,
		registryToken,
		registry,
		buildAuthString(tenantUUID, registryToken))

	return pullSecretDataFromDockerConfig(dockerCfg)
}

func buildAuthString(tenantUUID string, registryToken string) string {
	auth := fmt.Sprintf("%s:%s", tenantUUID, registryToken)

	return b64.StdEncoding.EncodeToString([]byte(auth))
}

func pullSecretDataFromDockerConfig(dockerConf *dockerConfig) (map[string][]byte, error) {
	dockerConfJSON, err := json.Marshal(dockerConf)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return map[string][]byte{DockerConfigJSON: dockerConfJSON}, nil
}
