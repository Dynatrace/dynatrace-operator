package standalone

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testApiUrl    = "test.com"
	testApiToken  = "testy"
	testPaasToken = "testz"

	testProxy       = "proxy"
	testNetworkZone = "zone"
	testTrustedCA   = "secret"

	testTenantUUID = "test"
	testNodeName   = "node1"
	testNodeIP     = "123"
	testTlsCert    = "tls"
	testHostGroup  = "group"
	testClusterID  = "id"
)

var testSecretConfig = SecretConfig{
	ApiUrl:        testApiUrl,
	ApiToken:      testApiToken,
	PaasToken:     testPaasToken,
	Proxy:         testProxy,
	NetworkZone:   testNetworkZone,
	TrustedCAs:    testTrustedCA,
	SkipCertCheck: true,
	TenantUUID:    testTenantUUID,
	HasHost:       true,
	MonitoringNodes: map[string]string{
		testNodeName: testNodeIP,
	},
	TlsCert:   testTlsCert,
	HostGroup: testHostGroup,
	ClusterID: testClusterID,
}

func TestNewSecretConfigViaFs(t *testing.T) {
	t.Run(`read correct json`, func(t *testing.T) {
		fs := prepTestFs(t)

		config, err := newSecretConfigViaFs(fs)

		require.NoError(t, err)
		require.NotNil(t, config)

		assert.Equal(t, testApiUrl, config.ApiUrl)
		assert.Equal(t, testApiToken, config.ApiToken)
		assert.Equal(t, testPaasToken, config.PaasToken)
		assert.Equal(t, testProxy, config.Proxy)
		assert.Equal(t, testNetworkZone, config.NetworkZone)
		assert.Equal(t, testTrustedCA, config.TrustedCAs)
		assert.True(t, config.SkipCertCheck)
		assert.Equal(t, testTenantUUID, config.TenantUUID)
		assert.True(t, config.HasHost)
		assert.Equal(t, testTlsCert, config.TlsCert)
		assert.Equal(t, testHostGroup, config.HostGroup)
		assert.Equal(t, testClusterID, config.ClusterID)
		assert.Len(t, config.MonitoringNodes, 1)
	})
}

func prepTestFs(t *testing.T) afero.Fs {
	fs := afero.NewMemMapFs()
	require.NoError(t, fs.MkdirAll(SecretConfigMount, 0770))

	file, err := fs.OpenFile(filepath.Join(SecretConfigMount, SecretConfigFieldName), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0770)
	require.NoError(t, err)
	require.NotNil(t, file)

	rawJson, err := json.Marshal(testSecretConfig)
	require.NoError(t, err)

	_, err = file.Write(rawJson)
	require.NoError(t, err)
	file.Close()

	return fs
}
