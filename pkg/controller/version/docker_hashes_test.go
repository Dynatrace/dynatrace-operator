package version

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/controller/parser"
	"github.com/containers/image/v5/transports/alltransports"
	"github.com/containers/image/v5/types"
	"github.com/stretchr/testify/assert"
)

func TestMakeSystemContext(t *testing.T) {
	versionChecker := NewDockerHashesChecker(
		"localhost.com/image:1234",
		"localhost.com/image:1234",
		nil)

	assert.NotNil(t, versionChecker)

	reference, err := alltransports.ParseImageName("docker://localhost/image:1234")
	assert.NoError(t, err)
	assert.NotNil(t, reference)

	noAuth := makeSystemContext(reference.DockerReference(), versionChecker.dockerConfig)
	assert.Equal(t, types.SystemContext{}, *noAuth)

	auths := make(map[string]parser.DockerConfigAuth)
	auths["localhost.com"] = parser.DockerConfigAuth{Username: "username", Password: "password"}
	versionChecker.dockerConfig = &parser.DockerConfig{Auths: auths}
	missingAuth := makeSystemContext(reference.DockerReference(), versionChecker.dockerConfig)
	assert.Equal(t, types.SystemContext{}, *missingAuth)

	auths["localhost"] = parser.DockerConfigAuth{Username: "username", Password: "password"}
	versionChecker.dockerConfig = &parser.DockerConfig{Auths: auths}
	withAuth := makeSystemContext(reference.DockerReference(), versionChecker.dockerConfig)
	assert.Equal(t, withAuth.DockerAuthConfig.Username, "username")
	assert.Equal(t, withAuth.DockerAuthConfig.Password, "password")
}
