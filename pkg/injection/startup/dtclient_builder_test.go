package startup

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateClient(t *testing.T) {
	t.Run(`no options`, func(t *testing.T) {
		config := basicTestSecretConfigForClient()
		builder := newDTClientBuilder(config, "")

		client, err := builder.createClient()

		require.NoError(t, err)
		require.NotNil(t, client)

		assert.Empty(t, builder.options)
	})

	t.Run(`multiple options`, func(t *testing.T) {
		config := complexTestSecretConfigForClient()
		builder := newDTClientBuilder(config, "")

		client, err := builder.createClient()

		require.NoError(t, err)
		require.NotNil(t, client)

		assert.Len(t, builder.options, 3)
	})
}

func basicTestSecretConfigForClient() *SecretConfig {
	return &SecretConfig{
		ApiUrl:    testApiUrl,
		ApiToken:  testApiToken,
		PaasToken: testPaasToken,
	}
}

func complexTestSecretConfigForClient() *SecretConfig {
	return &SecretConfig{
		ApiUrl:      testApiUrl,
		ApiToken:    testApiToken,
		PaasToken:   testPaasToken,
		Proxy:       testProxy,
		NoProxy:     testNoProxy,
		NetworkZone: testNetworkZone,
		//TrustedCAs:  testTrustedCA,
	}
}
