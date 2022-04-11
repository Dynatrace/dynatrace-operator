package dockerconfig

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
		auths, err := parseDockerAuthsFromSecret(nil)
		assert.Nil(t, auths)
		assert.Error(t, err)
	})
	t.Run(`NewDockerConfig handles missing secret data`, func(t *testing.T) {
		auths, err := parseDockerAuthsFromSecret(&corev1.Secret{})
		assert.Nil(t, auths)
		assert.Error(t, err)
	})
	t.Run(`NewDockerConfig handles invalid json`, func(t *testing.T) {
		auths, err := parseDockerAuthsFromSecret(&corev1.Secret{
			Data: map[string][]byte{
				".dockerconfigjson": []byte(`invalid json`),
			},
		})

		assert.Nil(t, auths)
		assert.Error(t, err)
	})
	t.Run(`NewDockerConfig returns docker config from valid secret`, func(t *testing.T) {
		auths, err := parseDockerAuthsFromSecret(&corev1.Secret{
			Data: map[string][]byte{
				".dockerconfigjson": []byte(
					fmt.Sprintf(`{ "auths": { "%s": { "username": "%s", "password": "%s" } } }`, testKey, testName, testValue)),
			},
		})

		assert.NoError(t, err)
		assert.NotEmpty(t, auths)
		assert.Contains(t, auths, testKey)
		assert.Equal(t, testName, auths[testKey].Username)
		assert.Equal(t, testValue, auths[testKey].Password)
	})
}
