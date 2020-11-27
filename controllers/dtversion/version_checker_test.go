package dtversion

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	testString = "string"
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
	t.Run(`MakeSystemContext sets credentials from docker config`, func(t *testing.T) {
		systemContext := MakeSystemContext(&mockDockerReference{}, &DockerConfig{
			Auths: map[string]DockerConfigAuth{
				testName: {
					Username: testName,
					Password: testString,
				},
			},
		})
		assert.NotNil(t, systemContext)
		assert.NotNil(t, systemContext.DockerAuthConfig)
		assert.Equal(t, testName, systemContext.DockerAuthConfig.Username)
		assert.Equal(t, testString, systemContext.DockerAuthConfig.Password)
	})
}

type mockDockerReference struct{}

func (m mockDockerReference) String() string {
	return testString
}

func (m mockDockerReference) Name() string {
	return testName
}