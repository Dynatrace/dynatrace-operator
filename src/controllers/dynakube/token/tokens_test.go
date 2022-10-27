package token

import (
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func TestTokens(t *testing.T) {
	t.Run("set api token scopes", testSetApiTokenScopes)
	t.Run("set paas token scopes", testPaasTokenScopes)
	t.Run("set data ingest token scopes", testDataIngestTokenScopes)
	t.Run("verify token scopes", testVerifyTokenScopes)
}

func testSetApiTokenScopes(t *testing.T) {
	t.Run("empty dynakube", func(t *testing.T) {
		tokens := Tokens{
			dtclient.DynatraceApiToken: {},
		}
		tokens = tokens.setScopes(dynatracev1beta1.DynaKube{})

		assert.Equal(t,
			[]string{
				dtclient.TokenScopeInstallerDownload,
				dtclient.TokenScopeDataExport,
			},
			tokens.ApiToken().RequiredScopes)
	})
	t.Run("disabled host requests", func(t *testing.T) {
		tokens := Tokens{
			dtclient.DynatraceApiToken: {},
		}
		tokens = tokens.setScopes(dynatracev1beta1.DynaKube{
			ObjectMeta: v1.ObjectMeta{
				Annotations: map[string]string{
					dynatracev1beta1.AnnotationFeatureHostsRequests: "false",
				},
			},
		})

		assert.Equal(t,
			[]string{dtclient.TokenScopeInstallerDownload},
			tokens.ApiToken().RequiredScopes)
	})
	t.Run("kubernetes monitoring with auth token", func(t *testing.T) {
		tokens := Tokens{
			dtclient.DynatraceApiToken: {},
		}
		tokens = tokens.setScopes(dynatracev1beta1.DynaKube{
			ObjectMeta: v1.ObjectMeta{
				Annotations: map[string]string{
					dynatracev1beta1.AnnotationFeatureAutomaticK8sApiMonitoring: "true",
				},
			},
			Spec: dynatracev1beta1.DynaKubeSpec{
				ActiveGate: dynatracev1beta1.ActiveGateSpec{
					Capabilities: []dynatracev1beta1.CapabilityDisplayName{
						dynatracev1beta1.KubeMonCapability.DisplayName,
					},
				},
			},
		})

		assert.Equal(t,
			[]string{
				dtclient.TokenScopeInstallerDownload,
				dtclient.TokenScopeDataExport,
				dtclient.TokenScopeEntitiesRead,
				dtclient.TokenScopeSettingsRead,
				dtclient.TokenScopeSettingsWrite,
				dtclient.TokenScopeActiveGateTokenCreate,
			},
			tokens.ApiToken().RequiredScopes)
	})
}

func testPaasTokenScopes(t *testing.T) {
	tokens := Tokens{
		dtclient.DynatracePaasToken: {},
	}
	tokens = tokens.setScopes(dynatracev1beta1.DynaKube{})

	assert.Equal(t,
		[]string{dtclient.TokenScopeInstallerDownload},
		tokens.PaasToken().RequiredScopes)
}

func testDataIngestTokenScopes(t *testing.T) {
	tokens := Tokens{
		dtclient.DynatraceDataIngestToken: {},
	}
	tokens = tokens.setScopes(dynatracev1beta1.DynaKube{})

	assert.Equal(t,
		[]string{dtclient.TokenScopeMetricsIngest},
		tokens.DataIngestToken().RequiredScopes)
}

func testVerifyTokenScopes(t *testing.T) {
	validTokens := Tokens{
		"empty-scopes": Token{
			Value:          "empty-scopes",
			RequiredScopes: []string{},
		},
		"valid-scopes": Token{
			Value:          "valid-scopes",
			RequiredScopes: []string{"a", "c"},
		},
	}
	invalidTokens := Tokens{
		"invalid-scopes": Token{
			Value:          "invalid-scopes",
			RequiredScopes: []string{"a", "b", "c", "d"},
		},
	}
	apiError := Tokens{
		"api-error": Token{
			Value:          "api-error",
			RequiredScopes: []string{"a", "c"},
		},
	}
	fakeDynatraceClient := &dtclient.MockDynatraceClient{}

	fakeDynatraceClient.
		On("GetTokenScopes", "empty-scopes").
		Return(dtclient.TokenScopes{"a", "c"}, nil)
	fakeDynatraceClient.
		On("GetTokenScopes", "valid-scopes").
		Return(dtclient.TokenScopes{"a", "c"}, nil)
	fakeDynatraceClient.
		On("GetTokenScopes", "invalid-scopes").
		Return(dtclient.TokenScopes{"a", "c"}, nil)
	fakeDynatraceClient.
		On("GetTokenScopes", "api-error").
		Return(dtclient.TokenScopes{}, errors.New("test api-error"))

	fakeDynatraceClient.AssertNotCalled(t, "GetTokenScopes", "empty-scopes")
	assert.NoError(t, validTokens.verifyScopes(fakeDynatraceClient))
	assert.EqualError(t,
		invalidTokens.verifyScopes(fakeDynatraceClient),
		"token 'invalid-scopes' is missing the following scopes: [ b, d ]")
	assert.EqualError(t,
		apiError.verifyScopes(fakeDynatraceClient),
		"test api-error")

}
