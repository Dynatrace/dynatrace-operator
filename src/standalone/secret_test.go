package standalone

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSecretJson = `{
"apiUrl": "test.com",
"apiToken": "test.token",
"paasToken": "test.old.token",
"proxy": "test.proxy",
"networkZone": "testZone",
"trustedCAs": "trust",
"skipCertCheck": true,
"tenantUUID": "test",
"hasHost": true,
"monitoringNodes": {
	"node1": "123",
	"node2": "223"
},
"tlsCert": "test-cert",
"hostGroup": "test-group",
"clusterID": "test-id"
}`

func TestNewSecretConfigViaFs(t *testing.T) {
	t.Run(`read correct json`, func(t *testing.T) {
		fs := prepTestFs(t)

		config, err := newSecretConfigViaFs(fs)

		require.NoError(t, err)
		require.NotNil(t, config)

		assert.Equal(t, "test.com", config.ApiUrl)
		assert.Equal(t, "test.token", config.ApiToken)
		assert.Equal(t, "test.old.token", config.PaasToken)
		assert.Equal(t, "test.proxy", config.Proxy)
		assert.Equal(t, "testZone", config.NetworkZone)
		assert.Equal(t, "trust", config.TrustedCAs)
		assert.True(t, config.SkipCertCheck)
		assert.Equal(t, "test", config.TenantUUID)
		assert.True(t, config.HasHost)
		assert.Equal(t, "test-cert", config.TlsCert)
		assert.Equal(t, "test-group", config.HostGroup)
		assert.Equal(t, "test-id", config.ClusterID)
		assert.Len(t, config.MonitoringNodes, 2)
	})
}

func prepTestFs(t *testing.T) afero.Fs {
	fs := afero.NewMemMapFs()
	require.NoError(t, fs.MkdirAll(SecretConfigMount, 0770))

	file, err := fs.OpenFile(filepath.Join(SecretConfigMount, SecretConfigFieldName), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0770)
	require.NoError(t, err)
	require.NotNil(t, file)

	_, err = file.WriteString(testSecretJson)
	require.NoError(t, err)
	file.Close()

	return fs
}
