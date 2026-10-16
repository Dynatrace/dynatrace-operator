// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package dtpullsecret

import (
	b64 "encoding/base64"
	"encoding/json"

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
	registryToken := tokens.APIToken().Value
	if registryToken == "" {
		return nil, errors.New("token secret does not contain an api token, cannot generate docker config")
	}

	registry := dk.APIURLHost()

	tenantUUID, err := dk.TenantUUID()
	if err != nil {
		return nil, errors.WithMessage(err, "cannot generate docker config")
	}

	dockerCfg := newDockerConfigWithAuth(tenantUUID,
		registryToken,
		registry,
		basicAuth(tenantUUID, registryToken))

	return pullSecretDataFromDockerConfig(dockerCfg)
}

func basicAuth(username string, password string) string {
	auth := username + ":" + password

	return b64.StdEncoding.EncodeToString([]byte(auth))
}

func pullSecretDataFromDockerConfig(dockerConf *dockerConfig) (map[string][]byte, error) {
	dockerConfJSON, err := json.Marshal(dockerConf)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return map[string][]byte{DockerConfigJSON: dockerConfJSON}, nil
}
