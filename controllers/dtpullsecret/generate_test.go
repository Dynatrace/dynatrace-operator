package dtpullsecret

import (
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"testing"

	dynatracev1 "github.com/Dynatrace/dynatrace-operator/api/v1"
	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/dtclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
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
	instance := &dynatracev1.DynaKube{
		Spec: dynatracev1.DynaKubeSpec{
			APIURL: testApiUrl,
		},
		Status: dynatracev1alpha1.DynaKubeStatus{
			ConnectionInfo: dynatracev1alpha1.ConnectionInfoStatus{
				CommunicationHosts: []dynatracev1alpha1.CommunicationHostStatus{
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
		instance: instance,
		token: &corev1.Secret{
			Data: map[string][]byte{
				dtclient.DynatracePaasToken: []byte(testPaasToken),
			},
		},
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
	err = json.Unmarshal(data[dockerConfigJson], &actual)

	require.NoError(t, err)
	assert.NotNil(t, actual)
	assert.Equal(t, expected, actual)
}
