package version

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/controller/parser"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestGetDockerLabel(t *testing.T) {
	auths := make(map[string]parser.DockerConfigAuth)
	auths["https://index.docker.io/v1/"] = parser.DockerConfigAuth{
		Username: os.Getenv("DOCKER_USERNAME"),
		Password: os.Getenv("DOCKER_PASSWORD")}

	versionChecker := NewDockerLabelsChecker(
		"meik99/test:2.0.0",
		map[string]string{
			"version": "1.0.0",
		},
		&parser.DockerConfig{Auths: auths})

	isLatest, err := versionChecker.IsLatest()

	assert.NoError(t, err)
	assert.False(t, isLatest)

	versionChecker = NewDockerLabelsChecker(
		"meik99/test:2.0.0",
		map[string]string{
			"version": "2.0.0",
		},
		&parser.DockerConfig{Auths: auths})

	isLatest, err = versionChecker.IsLatest()

	assert.NoError(t, err)
	assert.True(t, isLatest)

	versionChecker = NewDockerLabelsChecker(
		"meik99/test:2.0.0",
		map[string]string{
			"version": "3.0.0",
		},
		&parser.DockerConfig{Auths: auths})

	isLatest, err = versionChecker.IsLatest()

	assert.NoError(t, err)
	assert.True(t, isLatest)
}
