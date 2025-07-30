package token

import (
	"context"
	"net/http"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	dtclientmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func getAllScopesForAPIToken() dtclient.TokenScopes {
	return []string{
		dtclient.TokenScopeSettingsRead,
		dtclient.TokenScopeSettingsWrite,
		dtclient.TokenScopeActiveGateTokenCreate,
	}
}

func getAllScopesForPaaSToken() dtclient.TokenScopes {
	return []string{
		dtclient.TokenScopeInstallerDownload,
	}
}

func getAllScopesForDataIngest() dtclient.TokenScopes {
	return []string{
		dtclient.TokenScopeMetricsIngest,
	}
}

const (
	fakeTokenNoPermissions                       = "no-permissions"
	fakeTokenAllAPITokenPermissions              = "all-permissions"
	fakeTokenAllAPITokenPermissionsIncludingPaaS = "all-permissions-including-paas"
	fakeTokenPaas                                = "paas-token"
	fakeTokenAllDataIngestPermissions            = "all-data-ingest-permissions"
)

func createFakeClient(t *testing.T) *dtclientmock.Client {
	fakeClient := dtclientmock.NewClient(t)

	tokenScopes := []struct {
		token  string
		scopes dtclient.TokenScopes
	}{
		{fakeTokenNoPermissions, dtclient.TokenScopes{}},
		{fakeTokenAllAPITokenPermissions, getAllScopesForAPIToken()},
		{fakeTokenAllAPITokenPermissionsIncludingPaaS, append(getAllScopesForAPIToken(), getAllScopesForPaaSToken()...)},
		{fakeTokenPaas, getAllScopesForPaaSToken()},
		{fakeTokenAllDataIngestPermissions, getAllScopesForDataIngest()},
	}

	for _, tokenScope := range tokenScopes {
		fakeClient.On("GetTokenScopes", mock.Anything, tokenScope.token).
			Return(tokenScope.scopes, nil).Maybe()
	}

	return fakeClient
}

func TestTokens(t *testing.T) {
	t.Run("empty dynakube, all permissions in api token, but paas => should fail", func(t *testing.T) {
		apiToken := newToken(dtclient.APIToken, fakeTokenAllAPITokenPermissions)
		tokens := Tokens{
			dtclient.APIToken: &apiToken,
		}
		tokens = tokens.AddFeatureScopesToTokens()
		_, err := tokens.VerifyScopes(context.Background(), createFakeClient(t), dynakube.DynaKube{})

		assert.Len(t, tokens.APIToken().Features, 4)
		assert.Empty(t, tokens.PaasToken().Features)
		assert.Empty(t, tokens.DataIngestToken().Features)
		assert.EqualError(t, err, "token 'apiToken' has scope errors: [feature 'Download Installer' is missing scope 'InstallerDownload']")
	})
	t.Run("empty dynakube, all permissions in api token, but paas + paas token => should work", func(t *testing.T) {
		apiToken := newToken(dtclient.APIToken, fakeTokenAllAPITokenPermissions)
		paasToken := newToken(dtclient.PaasToken, fakeTokenPaas)
		tokens := Tokens{
			dtclient.APIToken:  &apiToken,
			dtclient.PaasToken: &paasToken,
		}
		tokens = tokens.AddFeatureScopesToTokens()
		_, err := tokens.VerifyScopes(context.Background(), createFakeClient(t), dynakube.DynaKube{})

		assert.Len(t, tokens.APIToken().Features, 4)
		assert.Len(t, tokens.PaasToken().Features, 1)
		assert.Empty(t, tokens.DataIngestToken().Features)
		assert.NoError(t, err)
	})
	t.Run("empty dynakube, all permissions in api token => should work", func(t *testing.T) {
		apiToken := newToken(dtclient.APIToken, fakeTokenAllAPITokenPermissionsIncludingPaaS)
		tokens := Tokens{
			dtclient.APIToken: &apiToken,
		}
		tokens = tokens.AddFeatureScopesToTokens()
		_, err := tokens.VerifyScopes(context.Background(), createFakeClient(t), dynakube.DynaKube{})

		assert.Len(t, tokens.APIToken().Features, 4)
		assert.Empty(t, tokens.PaasToken().Features)
		assert.Empty(t, tokens.DataIngestToken().Features)
		assert.NoError(t, err)
	})
	t.Run("activegate enabled dynakube, no permissions in api token => fail", func(t *testing.T) {
		dk := dynakube.DynaKube{}
		dk.Spec.ActiveGate.Capabilities = []activegate.CapabilityDisplayName{
			activegate.KubeMonCapability.DisplayName,
		}

		apiToken := newToken(dtclient.APIToken, fakeTokenNoPermissions)
		tokens := Tokens{
			dtclient.APIToken: &apiToken,
		}
		tokens = tokens.AddFeatureScopesToTokens()
		_, err := tokens.VerifyScopes(context.Background(), createFakeClient(t), dk)

		assert.Len(t, tokens.APIToken().Features, 4)
		assert.Empty(t, tokens.PaasToken().Features)
		assert.Empty(t, tokens.DataIngestToken().Features)
		assert.EqualError(t, err, "token 'apiToken' has scope errors: [feature 'Automatic ActiveGate Token Creation' is missing scope 'activeGateTokenManagement.create' feature 'Download Installer' is missing scope 'InstallerDownload']")
	})
	t.Run("data ingest enabled => dataingest token missing rights => fail", func(t *testing.T) {
		dk := dynakube.DynaKube{}
		enableKubernetesMonitoringAndMetricsIngest(&dk)

		apiToken := newToken(dtclient.APIToken, fakeTokenAllAPITokenPermissionsIncludingPaaS)
		dataingestToken := newToken(dtclient.DataIngestToken, fakeTokenNoPermissions)
		tokens := Tokens{
			dtclient.APIToken:        &apiToken,
			dtclient.DataIngestToken: &dataingestToken,
		}
		tokens = tokens.AddFeatureScopesToTokens()
		_, err := tokens.VerifyScopes(context.Background(), createFakeClient(t), dk)

		assert.Len(t, tokens.APIToken().Features, 4)
		assert.Empty(t, tokens.PaasToken().Features)
		assert.Len(t, tokens.DataIngestToken().Features, 1)
		assert.EqualError(t, err, "token 'dataIngestToken' has scope errors: [feature 'Data Ingest' is missing scope 'metrics.ingest']")
	})
	t.Run("data ingest enabled => dataingest token has rights => success", func(t *testing.T) {
		apiToken := newToken(dtclient.APIToken, fakeTokenAllAPITokenPermissionsIncludingPaaS)
		dataingestToken := newToken(dtclient.DataIngestToken, fakeTokenAllDataIngestPermissions)
		tokens := Tokens{
			dtclient.APIToken:        &apiToken,
			dtclient.DataIngestToken: &dataingestToken,
		}
		tokens = tokens.AddFeatureScopesToTokens()
		_, err := tokens.VerifyScopes(context.Background(), createFakeClient(t), dynakube.DynaKube{})

		assert.Len(t, tokens.APIToken().Features, 4)
		assert.Empty(t, tokens.PaasToken().Features)
		assert.Len(t, tokens.DataIngestToken().Features, 1)
		assert.NoError(t, err)
	})
}

func TestOptionalTokens(t *testing.T) {
	t.Run("optional scope is missing - kubernetes-monitoring enabled", func(t *testing.T) {
		dk := dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					"feature.dynatrace.com/automatic-kubernetes-api-monitoring": "true",
				},
			},
			Spec: dynakube.DynaKubeSpec{
				ActiveGate: activegate.Spec{
					Capabilities: []activegate.CapabilityDisplayName{
						activegate.KubeMonCapability.DisplayName,
					},
				},
			},
		}

		apiTokenNoMissingScopes := "api-token-value1"
		apiTokenMissingSettingsRead := "api-token-value2"
		apiTokenMissingSettingsWrite := "api-token-value3"
		apiTokenMissingSettingsReadWrite := "api-token-value4"

		fakeClient := dtclientmock.NewClient(t)
		fakeClient.On("GetTokenScopes", mock.Anything, apiTokenNoMissingScopes).Return(dtclient.TokenScopes{
			dtclient.TokenScopeSettingsRead,
			dtclient.TokenScopeSettingsWrite,
			dtclient.TokenScopeInstallerDownload,
			dtclient.TokenScopeActiveGateTokenCreate,
		}, nil).Maybe()
		fakeClient.On("GetTokenScopes", mock.Anything, apiTokenMissingSettingsRead).Return(dtclient.TokenScopes{
			dtclient.TokenScopeSettingsWrite,
			dtclient.TokenScopeInstallerDownload,
			dtclient.TokenScopeActiveGateTokenCreate,
		}, nil).Maybe()
		fakeClient.On("GetTokenScopes", mock.Anything, apiTokenMissingSettingsWrite).Return(dtclient.TokenScopes{
			dtclient.TokenScopeSettingsRead,
			dtclient.TokenScopeInstallerDownload,
			dtclient.TokenScopeActiveGateTokenCreate,
		}, nil).Maybe()
		fakeClient.On("GetTokenScopes", mock.Anything, apiTokenMissingSettingsReadWrite).Return(dtclient.TokenScopes{
			dtclient.TokenScopeInstallerDownload,
			dtclient.TokenScopeActiveGateTokenCreate,
		}, nil).Maybe()

		missingScopes := map[string][]string{
			apiTokenNoMissingScopes: {},
			apiTokenMissingSettingsRead: {
				dtclient.TokenScopeSettingsRead,
			},
			apiTokenMissingSettingsWrite: {
				dtclient.TokenScopeSettingsWrite,
			},
			apiTokenMissingSettingsReadWrite: {
				dtclient.TokenScopeSettingsRead,
				dtclient.TokenScopeSettingsWrite,
			},
		}
		assertOptionalScopes(t, fakeClient, dk, missingScopes)
	})
	t.Run("optional scope is missing - metadataEnrichment enabled", func(t *testing.T) {
		dk := dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				MetadataEnrichment: dynakube.MetadataEnrichment{
					Enabled: ptr.To(true),
				},
			},
		}

		apiTokenNoMissingScopes := "api-token-value1"
		apiTokenMissingSettingsRead := "api-token-value2"

		fakeClient := dtclientmock.NewClient(t)
		fakeClient.On("GetTokenScopes", mock.Anything, apiTokenNoMissingScopes).Return(dtclient.TokenScopes{
			dtclient.TokenScopeSettingsRead,
			dtclient.TokenScopeInstallerDownload,
			dtclient.TokenScopeActiveGateTokenCreate,
		}, nil).Maybe()
		fakeClient.On("GetTokenScopes", mock.Anything, apiTokenMissingSettingsRead).Return(dtclient.TokenScopes{
			dtclient.TokenScopeInstallerDownload,
			dtclient.TokenScopeActiveGateTokenCreate,
		}, nil).Maybe()

		missingScopes := map[string][]string{
			apiTokenNoMissingScopes: {},
			apiTokenMissingSettingsRead: {
				dtclient.TokenScopeSettingsRead,
			},
		}
		assertOptionalScopes(t, fakeClient, dk, missingScopes)
	})
	t.Run("optional scope is missing - kubernetes-monitoring and metadataEnrichment enabled", func(t *testing.T) {
		dk := dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					"feature.dynatrace.com/automatic-kubernetes-api-monitoring": "true",
				},
			},
			Spec: dynakube.DynaKubeSpec{
				ActiveGate: activegate.Spec{
					Capabilities: []activegate.CapabilityDisplayName{
						activegate.KubeMonCapability.DisplayName,
					},
				},
				MetadataEnrichment: dynakube.MetadataEnrichment{
					Enabled: ptr.To(true),
				},
			},
		}

		apiTokenNoMissingScopes := "api-token-value1"
		apiTokenMissingSettingsRead := "api-token-value2"
		apiTokenMissingSettingsWrite := "api-token-value3"
		apiTokenMissingSettingsReadWrite := "api-token-value4"

		fakeClient := dtclientmock.NewClient(t)
		fakeClient.On("GetTokenScopes", mock.Anything, apiTokenNoMissingScopes).Return(dtclient.TokenScopes{
			dtclient.TokenScopeSettingsRead,
			dtclient.TokenScopeSettingsWrite,
			dtclient.TokenScopeInstallerDownload,
			dtclient.TokenScopeActiveGateTokenCreate,
		}, nil).Maybe()
		fakeClient.On("GetTokenScopes", mock.Anything, apiTokenMissingSettingsRead).Return(dtclient.TokenScopes{
			dtclient.TokenScopeSettingsWrite,
			dtclient.TokenScopeInstallerDownload,
			dtclient.TokenScopeActiveGateTokenCreate,
		}, nil).Maybe()
		fakeClient.On("GetTokenScopes", mock.Anything, apiTokenMissingSettingsWrite).Return(dtclient.TokenScopes{
			dtclient.TokenScopeSettingsRead,
			dtclient.TokenScopeInstallerDownload,
			dtclient.TokenScopeActiveGateTokenCreate,
		}, nil).Maybe()
		fakeClient.On("GetTokenScopes", mock.Anything, apiTokenMissingSettingsReadWrite).Return(dtclient.TokenScopes{
			dtclient.TokenScopeInstallerDownload,
			dtclient.TokenScopeActiveGateTokenCreate,
		}, nil).Maybe()

		missingScopes := map[string][]string{
			apiTokenNoMissingScopes: {},
			apiTokenMissingSettingsRead: {
				dtclient.TokenScopeSettingsRead,
			},
			apiTokenMissingSettingsWrite: {
				dtclient.TokenScopeSettingsWrite,
			},
			apiTokenMissingSettingsReadWrite: {
				dtclient.TokenScopeSettingsRead,
				dtclient.TokenScopeSettingsWrite,
			},
		}

		assertOptionalScopes(t, fakeClient, dk, missingScopes)
	})
}

func assertOptionalScopes(t *testing.T, fakeClient dtclient.Client, dk dynakube.DynaKube, missingScopes map[string][]string) {
	for tokenValue, scopes := range missingScopes {
		apiToken := newToken(dtclient.APIToken, tokenValue)
		tokens := Tokens{
			dtclient.APIToken: &apiToken,
		}
		tokens = tokens.AddFeatureScopesToTokens()
		missingOptionalScopes, err := tokens.VerifyScopes(context.Background(), fakeClient, dk)

		assert.Len(t, tokens.APIToken().Features, 4, tokenValue)
		assert.Equal(t, scopes, missingOptionalScopes, tokenValue)
		assert.NoError(t, err, tokenValue)
	}
}

func enableKubernetesMonitoringAndMetricsIngest(dk *dynakube.DynaKube) *dynakube.DynaKube {
	dk.Spec.ActiveGate.Capabilities = []activegate.CapabilityDisplayName{
		activegate.KubeMonCapability.DisplayName,
		activegate.MetricsIngestCapability.DisplayName,
	}

	return dk
}

func TestTokens_VerifyValues(t *testing.T) {
	validToken := newToken(dtclient.APIToken, "valid-value")
	invalidToken := newToken(dtclient.APIToken, " invalid-value ")

	validTokens := Tokens{
		dtclient.APIToken: &validToken,
	}
	invalidTokens := Tokens{
		dtclient.APIToken: &invalidToken,
	}

	require.NoError(t, validTokens.VerifyValues())
	require.EqualError(t, invalidTokens.VerifyValues(), "token 'apiToken' contains leading or trailing whitespaces")
}

type concatErrorsTestCase struct {
	name              string
	encounteredErrors []error
	message           string
}

func TestConcatErrors(t *testing.T) {
	stringError1 := errors.New("error 1")
	stringError2 := errors.New("error 2")
	serviceUnavailableError := dtclient.ServerError{
		Code:    http.StatusServiceUnavailable,
		Message: "ServiceUnavailable",
	}
	tooManyRequestsError := dtclient.ServerError{
		Code:    http.StatusTooManyRequests,
		Message: "TooManyRequests",
	}

	testCases := []concatErrorsTestCase{
		{
			name: "string errors",
			encounteredErrors: []error{
				stringError1,
				stringError2,
			},
			message: "error 1\n\terror 2",
		},
		{
			name: "string + ServiceUnavailable errors",
			encounteredErrors: []error{
				stringError1,
				serviceUnavailableError,
			},
			message: "dynatrace server error 503: error 1\n\tdynatrace server error 503: ServiceUnavailable",
		},
		{
			name: "string + TooManyRequests errors",
			encounteredErrors: []error{
				stringError1,
				tooManyRequestsError,
			},
			message: "dynatrace server error 429: error 1\n\tdynatrace server error 429: TooManyRequests",
		},
		{
			name: "string + ServiceUnavailable + TooManyRequests errors",
			encounteredErrors: []error{
				stringError1,
				serviceUnavailableError,
				tooManyRequestsError,
			},
			message: "dynatrace server error 503: error 1\n\tdynatrace server error 503: ServiceUnavailable\n\tdynatrace server error 429: TooManyRequests",
		},
		{
			name: "string + TooManyRequests + ServiceUnavailable errors",
			encounteredErrors: []error{
				stringError1,
				tooManyRequestsError,
				serviceUnavailableError,
			},
			message: "dynatrace server error 429: error 1\n\tdynatrace server error 429: TooManyRequests\n\tdynatrace server error 503: ServiceUnavailable",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := concatErrors(testCase.encounteredErrors)
			require.EqualError(t, err, testCase.message)
		})
	}
}

func TestCheckForDataIngestToken(t *testing.T) {
	t.Run("data ingest token is present, but empty", func(t *testing.T) {
		tokens := Tokens{
			dtclient.DataIngestToken: &Token{},
		}

		assert.False(t, CheckForDataIngestToken(tokens))
	})

	t.Run("data ingest token is present and not empty", func(t *testing.T) {
		tokens := Tokens{
			dtclient.DataIngestToken: &Token{
				Value: "token",
			},
		}

		assert.True(t, CheckForDataIngestToken(tokens))
	})

	t.Run("data ingest token is not present", func(t *testing.T) {
		tokens := Tokens{}

		assert.False(t, CheckForDataIngestToken(tokens))
	})
}
