package dtclient

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewClient(t *testing.T) {
	{
		c, err := NewClient("https://aabb.live.dynatrace.com/api", "foo", "bar")
		if assert.NoError(t, err) {
			assert.NotNil(t, c)
		}
	}
	{
		c, err := NewClient("https://aabb.live.dynatrace.com/api", "foo", "bar", SkipCertificateValidation(false))
		if assert.NoError(t, err) {
			assert.NotNil(t, c)
		}
	}
	{
		c, err := NewClient("https://aabb.live.dynatrace.com/api", "foo", "bar", SkipCertificateValidation(true))
		if assert.NoError(t, err) {
			assert.NotNil(t, c)
		}
	}

	{
		_, err := NewClient("https://aabb.live.dynatrace.com/api", "", "")
		assert.Error(t, err, "tokens are empty")
	}
	{
		_, err := NewClient("", "foo", "bar")
		assert.Error(t, err, "empty URL")
	}
}
