package token

import (
	"net/http"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestTokens(t *testing.T) {
	t.Run("set api token scopes", testSetApiTokenScopes)
	t.Run("set paas token scopes", testPaasTokenScopes)
	t.Run("set data ingest token scopes", testDataIngestTokenScopes)
	t.Run("verify token scopes", testVerifyTokenScopes)
	t.Run("verify token values", testVerifyTokenValues)
}

func testSetApiTokenScopes(t *testing.T) {
	t.Run("empty dynakube", func(t *testing.T) {
		tokens := Tokens{
			dtclient.DynatraceApiToken: {},
		}
		tokens = tokens.SetScopesForDynakube(dynatracev1beta1.DynaKube{})

		assert.Equal(t,
			[]string{
				dtclient.TokenScopeInstallerDownload,
				dtclient.TokenScopeDataExport,
			},
			tokens.ApiToken().RequiredScopes)
	})
	t.Run("kubernetes monitoring with auth token", func(t *testing.T) {
		tokens := Tokens{
			dtclient.DynatraceApiToken: {},
		}
		tokens = tokens.SetScopesForDynakube(dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
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
	tokens = tokens.SetScopesForDynakube(dynatracev1beta1.DynaKube{})

	assert.Equal(t,
		[]string{dtclient.TokenScopeInstallerDownload},
		tokens.PaasToken().RequiredScopes)
}

func testDataIngestTokenScopes(t *testing.T) {
	tokens := Tokens{
		dtclient.DynatraceDataIngestToken: {},
	}
	tokens = tokens.SetScopesForDynakube(dynatracev1beta1.DynaKube{})

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
	fakeDynatraceClient := mocks.NewClient(t)

	fakeDynatraceClient.
		On("GetTokenScopes", "empty-scopes").
		Return(dtclient.TokenScopes{"a", "c"}, nil).Maybe().Times(0)
	fakeDynatraceClient.
		On("GetTokenScopes", "valid-scopes").
		Return(dtclient.TokenScopes{"a", "c"}, nil)
	fakeDynatraceClient.
		On("GetTokenScopes", "invalid-scopes").
		Return(dtclient.TokenScopes{"a", "c"}, nil)
	fakeDynatraceClient.
		On("GetTokenScopes", "api-error").
		Return(dtclient.TokenScopes{}, errors.New("test api-error"))

	require.NoError(t, validTokens.VerifyScopes(fakeDynatraceClient))
	require.EqualError(t,
		invalidTokens.VerifyScopes(fakeDynatraceClient),
		"token 'invalid-scopes' is missing the following scopes: [ b, d ]")
	require.EqualError(t,
		apiError.VerifyScopes(fakeDynatraceClient),
		"test api-error")
}

func testVerifyTokenValues(t *testing.T) {
	validTokens := Tokens{
		"valid-value": Token{Value: "valid-value"},
	}
	invalidTokens := Tokens{
		"whitespaces": Token{Value: " whitespaces "},
	}

	require.NoError(t, validTokens.VerifyValues())
	require.EqualError(t, invalidTokens.VerifyValues(), "value of token 'whitespaces' contains whitespaces at the beginning or end of the value")
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
