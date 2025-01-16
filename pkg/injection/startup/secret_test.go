package startup

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testApiUrl    = "test.com"
	testApiToken  = "testy"
	testPaasToken = "testz"

	testProxy           = "proxy"
	testNoProxy         = "no-proxy"
	testOneAgentNoProxy = "oa-no-proxy"
	testNetworkZone     = "zone"
	testTrustedCA       = "secret"

	testNodeName  = "node1"
	testTlsCert   = "tls"
	testHostGroup = "group"

	testTenantUUID = "test"

	testInitialConnectRetry = 30
)

func getTestSecretConfig() *SecretConfig {
	return &SecretConfig{
		ApiUrl:          testApiUrl,
		ApiToken:        testApiToken,
		PaasToken:       testPaasToken,
		Proxy:           testProxy,
		NoProxy:         testNoProxy,
		OneAgentNoProxy: testOneAgentNoProxy,
		TenantUUID:      testTenantUUID,
		NetworkZone:     testNetworkZone,
		SkipCertCheck:   true,
		HasHost:         true,
		MonitoringNodes: map[string]string{
			testNodeName: testTenantUUID,
		},
		HostGroup:           testHostGroup,
		InitialConnectRetry: testInitialConnectRetry,
	}
}

func TestNewSecretConfigViaFs(t *testing.T) {
	fs := prepTestFs(t)

	config, err := newSecretConfigViaFs(fs)

	require.NoError(t, err)
	require.NotNil(t, config)

	assert.Equal(t, testApiUrl, config.ApiUrl)
	assert.Equal(t, testApiToken, config.ApiToken)
	assert.Equal(t, testPaasToken, config.PaasToken)
	assert.Equal(t, testTenantUUID, config.TenantUUID)
	assert.Equal(t, testProxy, config.Proxy)
	assert.Equal(t, testOneAgentNoProxy, config.OneAgentNoProxy)
	assert.Equal(t, testNetworkZone, config.NetworkZone)
	assert.True(t, config.SkipCertCheck)
	assert.True(t, config.HasHost)
	assert.Equal(t, testHostGroup, config.HostGroup)
	assert.Len(t, config.MonitoringNodes, 1)
	assert.Equal(t, testInitialConnectRetry, config.InitialConnectRetry)
}

func TestNewCertificatesViaFs(t *testing.T) {
	fs := prepTestFs(t)

	certificates, err := newCertificatesViaFs(fs, consts.CustomCertsFileName)

	require.NoError(t, err)
	assert.Equal(t, testTrustedCA, certificates)
}

func prepTestFs(t *testing.T) afero.Fs {
	fs := afero.NewMemMapFs()
	require.NoError(t, fs.MkdirAll(consts.SharedConfigConfigDirMount, 0770))

	file, err := fs.OpenFile(filepath.Join(consts.SharedConfigConfigDirMount, consts.AgentInitSecretConfigField), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0770)
	require.NoError(t, err)
	require.NotNil(t, file)

	rawJson, err := json.Marshal(getTestSecretConfig())
	require.NoError(t, err)

	_, err = file.Write(rawJson)
	require.NoError(t, err)

	err = file.Close()
	require.NoError(t, err)

	file, err = fs.OpenFile(filepath.Join(consts.SharedConfigConfigDirMount, consts.CustomCertsFileName), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0770)
	require.NoError(t, err)
	require.NotNil(t, file)

	_, err = file.Write([]byte(testTrustedCA))
	require.NoError(t, err)

	err = file.Close()
	require.NoError(t, err)

	return fs
}

func prepReadOnlyCSIFilesystem(t *testing.T, fs afero.Fs) afero.Fs {
	require.NoError(t, fs.MkdirAll(getReadOnlyAgentConfMountPath(), 0770))

	for i := range 10 {
		file, err := fs.OpenFile(filepath.Join(getReadOnlyAgentConfMountPath(), fmt.Sprintf("%d.conf", i)), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0444)
		require.NoError(t, err)
		err = file.Close()
		require.NoError(t, err)
	}

	fs.Chmod(getReadOnlyAgentConfMountPath(), 0444)

	require.NoError(t, fs.MkdirAll(consts.AgentConfInitDirMount, 0770))

	return fs
}
