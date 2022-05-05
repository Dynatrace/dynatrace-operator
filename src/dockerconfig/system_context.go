package dockerconfig

import (
	"strings"

	"github.com/containers/image/v5/docker/reference"
	"github.com/containers/image/v5/types"
)

// MakeSystemContext returns a SystemConfig for the given image and Dockerconfig.
func MakeSystemContext(dockerReference reference.Named, dockerConfig *DockerConfig) *types.SystemContext {
	if dockerReference == nil || dockerConfig == nil {
		return &types.SystemContext{}
	}

	var systemContext types.SystemContext

	if dockerConfig.SkipCertCheck {
		systemContext.DockerInsecureSkipTLSVerify = types.OptionalBoolTrue
	}
	if dockerConfig.TrustedCertsPath != "" {
		systemContext.DockerCertPath = dockerConfig.TrustedCertsPath
	}

	registry := strings.Split(dockerReference.Name(), "/")[0]

	for _, r := range []string{registry, "https://" + registry} {
		if creds, ok := dockerConfig.Auths[r]; ok {
			systemContext.DockerAuthConfig = &types.DockerAuthConfig{Username: creds.Username, Password: creds.Password}
		}
	}

	return &systemContext
}
