package dockerconfig

import (
	"github.com/containers/image/v5/docker/reference"
	"github.com/containers/image/v5/types"
)

// MakeSystemContext returns a SystemConfig for the given image and Dockerconfig.
func MakeSystemContext(dockerReference reference.Named, dockerConfig *DockerConfig) *types.SystemContext {
	if dockerReference == nil || dockerConfig == nil {
		return &types.SystemContext{}
	}

	var systemContext types.SystemContext

	if dockerConfig.SkipCertCheck() {
		systemContext.DockerInsecureSkipTLSVerify = types.OptionalBoolTrue
	}
	if dockerConfig.TrustedCertsPath != "" {
		systemContext.DockerCertPath = dockerConfig.TrustedCertsPath
	}
	if dockerConfig.RegistryAuthPath != "" {
		systemContext.AuthFilePath = dockerConfig.RegistryAuthPath
	}

	return &systemContext
}
