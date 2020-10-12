package parser

import (
	"encoding/json"
	"fmt"

	corev1 "k8s.io/api/core/v1"
)

type DockerConfigAuth struct {
	Username string
	Password string
}

type DockerConfig struct {
	Auths map[string]DockerConfigAuth
}

func NewDockerConfig(secret *corev1.Secret) (*DockerConfig, error) {
	if secret == nil {
		return nil, fmt.Errorf("given secret is nil")
	}

	config, hasConfig := secret.Data[".dockerconfigjson"]
	if !hasConfig {
		return nil, fmt.Errorf("could not find any docker config in image pull secret")
	}

	var dockerConf DockerConfig
	err := json.Unmarshal(config, &dockerConf)
	if err != nil {
		return nil, err
	}

	return &dockerConf, nil
}
