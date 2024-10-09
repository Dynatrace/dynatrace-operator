package token

import (
	"context"
	"net/http"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube/activegate"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	dtclientmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func getAllScopesForAPIToken() dtclient.TokenScopes {
	return []string{
		dtclient.TokenScopeDataExport,
		dtclient.TokenScopeEntitiesRead,
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
		apiToken := newToken(dtclient.ApiToken, fakeTokenAllAPITokenPermissions)
		tokens := Tokens{
			dtclient.ApiToken: &apiToken,
		}
		tokens = tokens.AddFeatureScopesToTokens()
		err := tokens.VerifyScopes(context.Background(), createFakeClient(t), dynakube.DynaKube{})

		assert.Len(t, tokens.ApiToken().Features, 4)
		assert.Empty(t, tokens.PaasToken().Features)
		assert.Empty(t, tokens.DataIngestToken().Features)
		assert.EqualError(t, err, "token 'apiToken' has scope errors: [feature 'Download Installer' is missing scope 'InstallerDownload']")
	})
	t.Run("empty dynakube, all permissions in api token, but paas + paas token => should work", func(t *testing.T) {
		apiToken := newToken(dtclient.ApiToken, fakeTokenAllAPITokenPermissions)
		paasToken := newToken(dtclient.PaasToken, fakeTokenPaas)
		tokens := Tokens{
			dtclient.ApiToken:  &apiToken,
			dtclient.PaasToken: &paasToken,
		}
		tokens = tokens.AddFeatureScopesToTokens()
		err := tokens.VerifyScopes(context.Background(), createFakeClient(t), dynakube.DynaKube{})

		assert.Len(t, tokens.ApiToken().Features, 4)
		assert.Len(t, tokens.PaasToken().Features, 1)
		assert.Empty(t, tokens.DataIngestToken().Features)
		assert.NoError(t, err)
	})
	t.Run("empty dynakube, all permissions in api token => should work", func(t *testing.T) {
		apiToken := newToken(dtclient.ApiToken, fakeTokenAllAPITokenPermissionsIncludingPaaS)
		tokens := Tokens{
			dtclient.ApiToken: &apiToken,
		}
		tokens = tokens.AddFeatureScopesToTokens()
		err := tokens.VerifyScopes(context.Background(), createFakeClient(t), dynakube.DynaKube{})

		assert.Len(t, tokens.ApiToken().Features, 4)
		assert.Empty(t, tokens.PaasToken().Features)
		assert.Empty(t, tokens.DataIngestToken().Features)
		assert.NoError(t, err, "")
	})
	t.Run("activegate enabled dynakube, no permissions in api token => fail", func(t *testing.T) {
		dk := dynakube.DynaKube{}
		dk.Spec.ActiveGate.Capabilities = []activegate.CapabilityDisplayName{
			activegate.KubeMonCapability.DisplayName,
		}

		apiToken := newToken(dtclient.ApiToken, fakeTokenNoPermissions)
		tokens := Tokens{
			dtclient.ApiToken: &apiToken,
		}
		tokens = tokens.AddFeatureScopesToTokens()
		err := tokens.VerifyScopes(context.Background(), createFakeClient(t), dk)

		assert.Len(t, tokens.ApiToken().Features, 4)
		assert.Empty(t, tokens.PaasToken().Features)
		assert.Empty(t, tokens.DataIngestToken().Features)
		assert.EqualError(t, err, "token 'apiToken' has scope errors: [feature 'Access problem and event feed, metrics, and topology' is missing scope 'DataExport' feature 'Kubernetes API Monitoring' is missing scope 'entities.read, settings.read, settings.write' feature 'Automatic ActiveGate Token Creation' is missing scope 'activeGateTokenManagement.create' feature 'Download Installer' is missing scope 'InstallerDownload']")
	})
	t.Run("data ingest enabled => dataingest token missing rights => fail", func(t *testing.T) {
		dk := dynakube.DynaKube{}
		enableKubernetesMonitoringAndMetricsIngest(&dk)

		apiToken := newToken(dtclient.ApiToken, fakeTokenAllAPITokenPermissionsIncludingPaaS)
		dataingestToken := newToken(dtclient.DataIngestToken, fakeTokenNoPermissions)
		tokens := Tokens{
			dtclient.ApiToken:        &apiToken,
			dtclient.DataIngestToken: &dataingestToken,
		}
		tokens = tokens.AddFeatureScopesToTokens()
		err := tokens.VerifyScopes(context.Background(), createFakeClient(t), dk)

		assert.Len(t, tokens.ApiToken().Features, 4)
		assert.Empty(t, tokens.PaasToken().Features)
		assert.Len(t, tokens.DataIngestToken().Features, 1)
		assert.EqualError(t, err, "token 'dataIngestToken' has scope errors: [feature 'Data Ingest' is missing scope 'metrics.ingest']")
	})
	t.Run("data ingest enabled => dataingest token has rights => success", func(t *testing.T) {
		apiToken := newToken(dtclient.ApiToken, fakeTokenAllAPITokenPermissionsIncludingPaaS)
		dataingestToken := newToken(dtclient.DataIngestToken, fakeTokenAllDataIngestPermissions)
		tokens := Tokens{
			dtclient.ApiToken:        &apiToken,
			dtclient.DataIngestToken: &dataingestToken,
		}
		tokens = tokens.AddFeatureScopesToTokens()
		err := tokens.VerifyScopes(context.Background(), createFakeClient(t), dynakube.DynaKube{})

		assert.Len(t, tokens.ApiToken().Features, 4)
		assert.Empty(t, tokens.PaasToken().Features)
		assert.Len(t, tokens.DataIngestToken().Features, 1)
		assert.NoError(t, err)
	})
}

func enableKubernetesMonitoringAndMetricsIngest(dk *dynakube.DynaKube) *dynakube.DynaKube {
	dk.Spec.ActiveGate.Capabilities = []activegate.CapabilityDisplayName{
		activegate.KubeMonCapability.DisplayName,
		activegate.MetricsIngestCapability.DisplayName,
	}

	return dk
}

func TestTokens_VerifyValues(t *testing.T) {
	validToken := newToken(dtclient.ApiToken, "valid-value")
	invalidToken := newToken(dtclient.ApiToken, " invalid-value ")

	validTokens := Tokens{
		dtclient.ApiToken: &validToken,
	}
	invalidTokens := Tokens{
		dtclient.ApiToken: &invalidToken,
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
