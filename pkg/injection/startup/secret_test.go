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
		TrustedCAs:      testTrustedCA,
		SkipCertCheck:   true,
		HasHost:         true,
		MonitoringNodes: map[string]string{
			testNodeName: testTenantUUID,
		},
		TlsCert:             testTlsCert,
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
	assert.Equal(t, testTrustedCA, config.TrustedCAs)
	assert.True(t, config.SkipCertCheck)
	assert.True(t, config.HasHost)
	assert.Equal(t, testTlsCert, config.TlsCert)
	assert.Equal(t, testHostGroup, config.HostGroup)
	assert.Len(t, config.MonitoringNodes, 1)
	assert.Equal(t, testInitialConnectRetry, config.InitialConnectRetry)
}

func prepTestFs(t *testing.T) afero.Fs {
	fs := afero.NewMemMapFs()
	require.NoError(t, fs.MkdirAll(consts.AgentConfigDirMount, 0770))

	file, err := fs.OpenFile(filepath.Join(consts.AgentConfigDirMount, consts.AgentInitSecretConfigField), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0770)
	require.NoError(t, err)
	require.NotNil(t, file)

	rawJson, err := json.Marshal(getTestSecretConfig())
	require.NoError(t, err)

	_, err = file.Write(rawJson)
	require.NoError(t, err)

	err = file.Close()
	require.NoError(t, err)

	return fs
}

func prepReadOnlyCSIFilesystem(t *testing.T, fs afero.Fs) afero.Fs {
	require.NoError(t, fs.MkdirAll(getReadOnlyAgentConfMountPath(), 0770))

	for i := 0; i < 10; i++ {
		file, err := fs.OpenFile(filepath.Join(getReadOnlyAgentConfMountPath(), fmt.Sprintf("%d.conf", i)), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0444)
		require.NoError(t, err)
		err = file.Close()
		require.NoError(t, err)
	}
	fs.Chmod(getReadOnlyAgentConfMountPath(), 0444)

	require.NoError(t, fs.MkdirAll(consts.AgentConfInitDirMount, 0770))

	return fs
}
