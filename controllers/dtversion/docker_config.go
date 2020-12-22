package dtversion

import (
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
)

type DockerConfig struct {
	Auths         map[string]DockerAuth
	SkipCertCheck bool
}

type DockerAuth struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func ParseDockerAuthsFromSecret(secret *corev1.Secret) (map[string]DockerAuth, error) {
	if secret == nil {
		return nil, fmt.Errorf("given secret is nil")
	}

	config, hasConfig := secret.Data[".dockerconfigjson"]
	if !hasConfig {
		return nil, fmt.Errorf("could not find any docker config in image pull secret")
	}

	var dockerConf struct {
		Auths map[string]DockerAuth `json:"auths"`
	}
	err := json.Unmarshal(config, &dockerConf)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return dockerConf.Auths, nil
}
