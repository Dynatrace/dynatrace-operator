// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package dtpullsecret

import (
	"encoding/base64"
	"encoding/json"
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

func TestGenerateData(t *testing.T) {
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

	tests := []struct {
		name        string
		tokens      token.Tokens
		expectToken string
	}{
		{"use paas token", token.Tokens{token.PaaSKey: &token.Token{Value: testPaasToken}}, testPaasToken},
		{"use api token", token.Tokens{token.APIKey: &token.Token{Value: testAPIToken}}, testAPIToken},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := generateData(dk, tt.tokens)

			require.NoError(t, err)
			assert.NotEmpty(t, data)

			auth := testTenant + ":" + tt.expectToken
			expected := dockerConfig{
				Auths: map[string]dockerAuthentication{
					testAPIURLHost: {
						Username: testTenant,
						Password: tt.expectToken,
						Auth:     base64.StdEncoding.EncodeToString([]byte(auth)),
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
