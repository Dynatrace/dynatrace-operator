package dtpullsecret

import (
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testTenant     = "test-tenant"
	testProtocol   = "http"
	testHost       = "test-host"
	testPort       = 1234
	testApiUrl     = "https://test-api-url/api"
	testApiUrlHost = "test-api-url"
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

func TestReconciler_GenerateData(t *testing.T) {
	instance := &dynatracev1beta1.DynaKube{
		Spec: dynatracev1beta1.DynaKubeSpec{
			APIURL: testApiUrl,
		},
		Status: dynatracev1beta1.DynaKubeStatus{
			ConnectionInfo: dynatracev1beta1.ConnectionInfoStatus{
				CommunicationHosts: []dynatracev1beta1.CommunicationHostStatus{
					{
						Protocol: testProtocol,
						Host:     testHost,
						Port:     testPort,
					},
				},
				TenantUUID: testTenant,
			},
		},
	}
	r := &Reconciler{
		instance:  instance,
		paasToken: testPaasToken,
	}

	data, err := r.GenerateData()

	assert.NoError(t, err)
	assert.NotNil(t, data)
	assert.NotEmpty(t, data)

	auth := fmt.Sprintf("%s:%s", testTenant, testPaasToken)
	expected := dockerConfig{
		Auths: map[string]dockerAuthentication{
			testApiUrlHost: {
				Username: testTenant,
				Password: testPaasToken,
				Auth:     b64.StdEncoding.EncodeToString([]byte(auth)),
			},
		},
	}

	var actual dockerConfig
	err = json.Unmarshal(data[DockerConfigJson], &actual)

	require.NoError(t, err)
	assert.NotNil(t, actual)
	assert.Equal(t, expected, actual)
}
