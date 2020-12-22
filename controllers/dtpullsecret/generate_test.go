package dtpullsecret

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetImageRegistryFromAPIURL(t *testing.T) {
	for _, url := range []string{
		"https://host.com/api",
		"https://host.com/e/abc1234/api",
		"http://host.com/api",
		"http://host.com/e/abc1234/api",
	} {
		host, err := getImageRegistryFromAPIURL(url)
		if assert.NoError(t, err) {
			assert.Equal(t, "host.com", host)
		}
	}
}
