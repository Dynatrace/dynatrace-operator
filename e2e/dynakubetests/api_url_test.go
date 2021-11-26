//go:build e2e
// +build e2e

package dynakubetests

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/e2e"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

const (
	keySkipCertCheck = "ONEAGENT_INSTALLER_SKIP_CERT_CHECK"
	keyEnvironmentId = "DYNATRACE_ENVIRONMENT_ID"
)

func TestApiURL(t *testing.T) {
	apiURL := os.Getenv(keyApiURL)
	assert.NotEmpty(t, apiURL, fmt.Sprintf("variable %s must be set", keyApiURL))

	environmentId := os.Getenv(keyEnvironmentId)
	assert.NotEmpty(t, apiURL, fmt.Sprintf("variable %s must be set", keyEnvironmentId))

	clt := e2e.CreateClient(t)
	assert.NotNil(t, clt)

	err := e2e.PrepareEnvironment(clt, namespace)
	require.NoError(t, err)

	instance := dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      testName,
		},
		Spec: dynatracev1beta1.DynaKubeSpec{
			APIURL: apiURL,
			Tokens: e2e.TokenSecretName,
			OneAgent: dynatracev1beta1.OneAgentSpec{
				ClassicFullStack: &dynatracev1beta1.ClassicFullStackSpec{},
			},
		},
	}

	err = clt.Create(context.TODO(), &instance)
	assert.NoError(t, err)

	phaseWait := e2e.NewOneAgentWaitConfiguration(t, clt, maxWaitCycles, namespace, testName)
	err = phaseWait.WaitForPhase(dynatracev1beta1.Deploying)
	assert.NoError(t, err)

	err = phaseWait.WaitForPhase(dynatracev1beta1.Running)
	assert.NoError(t, err)

	apiToken, paasToken := e2e.GetTokensFromEnv()
	dtc, err := dtclient.NewClient(apiURL, apiToken, paasToken)
	assert.NoError(t, err)

	connectionInfo, err := dtc.GetConnectionInfo()
	assert.NoError(t, err)
	assert.NotNil(t, connectionInfo)
	assert.Equal(t, environmentId, connectionInfo.TenantUUID)
	assert.True(t, containsAPIConnectionHost(connectionInfo, apiURL))

	apiScopes, err := dtc.GetTokenScopes(apiToken)
	assert.NoError(t, err)
	assert.NotEmpty(t, apiScopes)

	paasScopes, err := dtc.GetTokenScopes(paasToken)
	assert.NoError(t, err)
	assert.NotEmpty(t, paasScopes)
}

func containsAPIConnectionHost(connectionInfo dtclient.ConnectionInfo, apiURL string) bool {
	apiUrl, err := url.Parse(apiURL)
	if err != nil {
		return false
	}

	for _, connectionHost := range connectionInfo.CommunicationHosts {
		if connectionHost.Host == apiUrl.Host {
			return true
		}
	}
	return false
}
