package version

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/controller/parser"
	"github.com/containers/image/v5/docker/reference"
	"github.com/containers/image/v5/types"
	"strings"
)

type ReleaseValidator interface {
	IsLatest() (bool, error)
}

func MakeSystemContext(dockerReference reference.Named, dockerConfig *parser.DockerConfig) *types.SystemContext {
	if dockerReference == nil || dockerConfig == nil {
		return &types.SystemContext{}
	}

	registryName := strings.Split(dockerReference.Name(), "/")[0]
	credentials, hasCredentials := dockerConfig.Auths[registryName]

	if !hasCredentials {
		registryURL := "https://" + registryName
		credentials, hasCredentials = dockerConfig.Auths[registryURL]
		if !hasCredentials {
			return &types.SystemContext{}
		}
	}

	return &types.SystemContext{
		DockerAuthConfig: &types.DockerAuthConfig{
			Username: credentials.Username,
			Password: credentials.Password,
		}}

}
