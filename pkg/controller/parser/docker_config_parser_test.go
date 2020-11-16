package parser

import (
	"encoding/json"
	"github.com/Dynatrace/dynatrace-operator/pkg/controller/dtversion"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestNewDockerConfig(t *testing.T) {
	t.Run("NewDockerConfig", func(t *testing.T) {
		auths := make(map[string]dtversion.DockerConfigAuth)
		auths["localhost"] = dtversion.DockerConfigAuth{Username: "username", Password: "password"}
		templateDockerConf := dtversion.DockerConfig{Auths: auths}
		templateDockerConfJson, _ := json.Marshal(templateDockerConf)
		data := make(map[string][]byte)
		data[".dockerconfigjson"] = templateDockerConfJson
		secret := corev1.Secret{
			Data: data,
		}
		dockerConfig, err := dtversion.NewDockerConfig(&secret)
		assert.NoError(t, err)
		assert.NotNil(t, dockerConfig)
		assert.Equal(t, templateDockerConf, *dockerConfig)
	})
	t.Run("NewDockerConfig handle nil secret", func(t *testing.T) {
		dockerConfig, err := dtversion.NewDockerConfig(nil)
		assert.Error(t, err)
		assert.Nil(t, dockerConfig)
	})
	t.Run("NewDockerConfig handle empty data", func(t *testing.T) {
		secret := corev1.Secret{}
		dockerConfig, err := dtversion.NewDockerConfig(&secret)
		assert.Error(t, err)
		assert.Nil(t, dockerConfig)
	})
	t.Run("NewDockerConfig handle malformed data", func(t *testing.T) {
		data := make(map[string][]byte)
		data[".dockerconfigjson"] = make([]byte, 0)
		secret := corev1.Secret{
			Data: data,
		}
		dockerConfig, err := dtversion.NewDockerConfig(&secret)
		assert.Error(t, err)
		assert.Nil(t, dockerConfig)
	})
}
