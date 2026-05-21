package dtpullsecret

import (
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/communication"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testTenant     = "test-tenant"
	testAPIURLHost = "test-api-url"
	testAPIURL     = "https://" + testAPIURLHost + "/e/" + testTenant + "/api"
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
	dk := &dynakube.DynaKube{
		Spec: dynakube.DynaKubeSpec{
			APIURL: testAPIURL,
		},
		Status: dynakube.DynaKubeStatus{
			OneAgent: oneagent.Status{
				ConnectionInfo: communication.ConnectionInfo{
					TenantUUID: testTenant,
				},
			},
		},
	}
	r := &Reconciler{}

	tests := []struct {
		name        string
		tokens      token.Tokens
		expectToken string
	}{
		{
			"use paas token",
			token.Tokens{
				token.PaaSKey: &token.Token{Value: testPaasToken},
			},
			testPaasToken,
		},
		{
			"use api token",
			token.Tokens{
				token.APIKey: &token.Token{Value: testAPIToken},
			},
			testAPIToken,
		},
		{
			"prefer platform token",
			token.Tokens{
				token.PaaSKey: &token.Token{Value: testPaasToken},
				token.APIKey:  &token.Token{Value: testPlatformToken},
			},
			testPlatformToken,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			data, err := r.generateData(dk, test.tokens)

			require.NoError(t, err)
			assert.NotEmpty(t, data)

			auth := fmt.Sprintf("%s:%s", testTenant, test.expectToken)
			expected := dockerConfig{
				Auths: map[string]dockerAuthentication{
					testAPIURLHost: {
						Username: testTenant,
						Password: test.expectToken,
						Auth:     b64.StdEncoding.EncodeToString([]byte(auth)),
					},
				},
			}

			var actual dockerConfig
			err = json.Unmarshal(data[DockerConfigJSON], &actual)

			require.NoError(t, err)
			assert.Equal(t, expected, actual)
		})
	}
}
