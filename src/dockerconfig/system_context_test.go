package dockerconfig

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	testString           = "string"
	testRegistryAuthPath = "registryAuthPath"
	testTrustedCertsPath = "trustedCertsPath"
)

func TestMakeSystemContext(t *testing.T) {
	t.Run(`MakeSystemContext returns default value for nil values`, func(t *testing.T) {
		systemContext := MakeSystemContext(nil, nil)
		assert.NotNil(t, systemContext)
	})
	t.Run(`MakeSystemContext returns default value for docker config without credentials`, func(t *testing.T) {
		systemContext := MakeSystemContext(&mockDockerReference{}, &DockerConfig{})
		assert.NotNil(t, systemContext)
	})
	t.Run(`MakeSystemContext sets authfile and trusted certs path`, func(t *testing.T) {
		systemContext := MakeSystemContext(&mockDockerReference{}, &DockerConfig{
			RegistryAuthPath: testRegistryAuthPath,
			TrustedCertsPath: testTrustedCertsPath,
		})
		assert.NotNil(t, systemContext)
		assert.Equal(t, testRegistryAuthPath, systemContext.AuthFilePath)
		assert.Equal(t, testTrustedCertsPath, systemContext.DockerCertPath)
	})
}

type mockDockerReference struct{}

func (m mockDockerReference) String() string {
	return testString
}

func (m mockDockerReference) Name() string {
	return testName
}
