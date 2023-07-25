package standalone

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/config"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	testApiUrl      = "test.com"
	testApiToken    = "testy"
	testPaasToken   = "testz"
	testTenantToken = "token"

	testProxy       = "proxy"
	testNoProxy     = "no-proxy"
	testNetworkZone = "zone"
	testTrustedCA   = "secret"

	testNodeName  = "node1"
	testTlsCert   = "tls"
	testHostGroup = "group"
	testClusterID = "id"

	testTenantUUID = "test"
	testEndpoints  = "beep.com;bop.com"

	testInitialConnectRetry = 30
)

func getTestSecretConfig() *SecretConfig {
	return &SecretConfig{
		ApiUrl:         testApiUrl,
		ApiToken:       testApiToken,
		PaasToken:      testPaasToken,
		TenantToken:    testTenantToken,
		ConnectionInfo: getTestConnectionInfo(),
		Proxy:          testProxy,
		NoProxy:        testNoProxy,
		NetworkZone:    testNetworkZone,
		TrustedCAs:     testTrustedCA,
		SkipCertCheck:  true,
		HasHost:        true,
		MonitoringNodes: map[string]string{
			testNodeName: testTenantUUID,
		},
		TlsCert:             testTlsCert,
		HostGroup:           testHostGroup,
		ClusterID:           testClusterID,
		InitialConnectRetry: testInitialConnectRetry,
	}
}

func getTestConnectionInfo() dynatracev1beta1.OneAgentConnectionInfoStatus {
	return dynatracev1beta1.OneAgentConnectionInfoStatus{
		ConnectionInfoStatus: dynatracev1beta1.ConnectionInfoStatus{
			TenantUUID:  testTenantUUID,
			Endpoints:   testEndpoints,
			LastRequest: metav1.Time{},
		},
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
	assert.Equal(t, testTenantToken, config.TenantToken)
	assert.Equal(t, getTestConnectionInfo(), config.ConnectionInfo)
	assert.Equal(t, testProxy, config.Proxy)
	assert.Equal(t, testNetworkZone, config.NetworkZone)
	assert.Equal(t, testTrustedCA, config.TrustedCAs)
	assert.True(t, config.SkipCertCheck)
	assert.True(t, config.HasHost)
	assert.Equal(t, testTlsCert, config.TlsCert)
	assert.Equal(t, testHostGroup, config.HostGroup)
	assert.Equal(t, testClusterID, config.ClusterID)
	assert.Len(t, config.MonitoringNodes, 1)
	assert.Equal(t, testInitialConnectRetry, config.InitialConnectRetry)
}

func prepTestFs(t *testing.T) afero.Fs {
	fs := afero.NewMemMapFs()
	require.NoError(t, fs.MkdirAll(config.AgentConfigDirMount, 0770))

	file, err := fs.OpenFile(filepath.Join(config.AgentConfigDirMount, config.AgentInitSecretConfigField), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0770)
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

	require.NoError(t, fs.MkdirAll(config.AgentConfInitDirMount, 0770))
	return fs
}
