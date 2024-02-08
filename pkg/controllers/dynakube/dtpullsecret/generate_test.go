package dtpullsecret

import (
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testTenant     = "test-tenant"
	testApiUrlHost = "test-api-url"
	testApiUrl     = "https://" + testApiUrlHost + "/e/" + testTenant + "/api"
)

func TestGetImageRegistryFromAPIURL(t *testing.T) {
	for _, url := range []string{
		"https://host.com/api",
		"https://host.com/e/abc1234/api",
		"http://host.com/api",
		"http://host.com/e/abc1234/api",
	} {
		host, err := getImageRegistryFromAPIURL(url)
		require.NoError(t, err)
		assert.Equal(t, "host.com", host)
	}
}

func TestReconciler_GenerateData(t *testing.T) {
	dynakube := &dynatracev1beta1.DynaKube{
		Spec: dynatracev1beta1.DynaKubeSpec{
			APIURL: testApiUrl,
		},
	}
	r := &Reconciler{
		dynakube: dynakube,
		tokens: token.Tokens{
			dtclient.PaasToken: token.Token{Value: testPaasToken},
		},
	}

	data, err := r.GenerateData()

	require.NoError(t, err)
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
