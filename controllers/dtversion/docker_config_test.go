package dtversion

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

const (
	testName  = "test-name"
	testKey   = "testKey"
	testValue = "testValue"
)

func TestNewDockerConfig(t *testing.T) {
	t.Run(`NewDockerConfig handles nil secret`, func(t *testing.T) {
		config, err := NewDockerConfig(nil)
		assert.Nil(t, config)
		assert.Error(t, err)
	})
	t.Run(`NewDockerConfig handles missing secret data`, func(t *testing.T) {
		config, err := NewDockerConfig(&corev1.Secret{})
		assert.Nil(t, config)
		assert.Error(t, err)
	})
	t.Run(`NewDockerConfig handles invalid json`, func(t *testing.T) {
		config, err := NewDockerConfig(&corev1.Secret{
			Data: map[string][]byte{
				".dockerconfigjson": []byte(`invalid json`),
			},
		})

		assert.Nil(t, config)
		assert.Error(t, err)
	})
	t.Run(`NewDockerConfig returns docker config from valid secret`, func(t *testing.T) {
		config, err := NewDockerConfig(&corev1.Secret{
			Data: map[string][]byte{
				".dockerconfigjson": []byte(
					fmt.Sprintf(`{ "auths": { "%s": { "username": "%s", "password": "%s" } } }`, testKey, testName, testValue)),
			},
		})

		assert.NoError(t, err)
		assert.NotNil(t, config)
		assert.NotEmpty(t, config.Auths)
		assert.Contains(t, config.Auths, testKey)
		assert.Equal(t, testName, config.Auths[testKey].Username)
		assert.Equal(t, testValue, config.Auths[testKey].Password)
	})
}
