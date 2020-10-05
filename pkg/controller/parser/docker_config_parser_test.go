package parser

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestNewDockerConfig(t *testing.T) {
	t.Run("NewDockerConfig", func(t *testing.T) {
		auths := make(map[string]struct {
			Username string
			Password string
		})
		auths["localhost"] = struct {
			Username string
			Password string
		}{
			Username: "username",
			Password: "password",
		}
		templateDockerConf := DockerConfig{Auths: auths}
		templateDockerConfJson, _ := json.Marshal(templateDockerConf)
		data := make(map[string][]byte)
		data[".dockerconfigjson"] = templateDockerConfJson
		secret := corev1.Secret{
			Data: data,
		}
		dockerConfig, err := NewDockerConfig(&secret)
		assert.NoError(t, err)
		assert.NotNil(t, dockerConfig)
		assert.Equal(t, templateDockerConf, *dockerConfig)
	})
	t.Run("NewDockerConfig handle nil secret", func(t *testing.T) {
		dockerConfig, err := NewDockerConfig(nil)
		assert.Error(t, err)
		assert.Nil(t, dockerConfig)
	})
	t.Run("NewDockerConfig handle empty data", func(t *testing.T) {
		secret := corev1.Secret{}
		dockerConfig, err := NewDockerConfig(&secret)
		assert.Error(t, err)
		assert.Nil(t, dockerConfig)
	})
	t.Run("NewDockerConfig handle malformed data", func(t *testing.T) {
		data := make(map[string][]byte)
		data[".dockerconfigjson"] = make([]byte, 0)
		secret := corev1.Secret{
			Data: data,
		}
		dockerConfig, err := NewDockerConfig(&secret)
		assert.Error(t, err)
		assert.Nil(t, dockerConfig)
	})
}
