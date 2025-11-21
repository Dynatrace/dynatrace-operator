package token

import (
	"net/http"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/logmonitoring"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/metadataenrichment"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/otlp"
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
		dtclient.TokenScopeDataExport,
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
func getAllScopesForTelemetryIngest() dtclient.TokenScopes {
	return []string{
		dtclient.TokenScopeMetricsIngest,
		dtclient.TokenScopeOpenTelemetryTraceIngest,
		dtclient.TokenScopeLogsIngest,
	}
}

func getAllScopesForOTLPExporter() dtclient.TokenScopes {
	return []string{
		dtclient.TokenScopeMetricsIngest,
		dtclient.TokenScopeOpenTelemetryTraceIngest,
		dtclient.TokenScopeLogsIngest,
	}
}

func TestTokens(t *testing.T) {
	const (
		fakeTokenNoPermissions                       = "no-permissions"
		fakeTokenAllAPITokenPermissions              = "all-permissions"
		fakeTokenAllAPITokenPermissionsIncludingPaaS = "all-permissions-including-paas"
		fakeTokenPaas                                = "paas-token"
		fakeTokenAllDataIngestPermissions            = "all-data-ingest-permissions"
		fakeTokenAllOTLPExporterPermissions          = "all-otlp-exporter-permissions"
		fakeTokenAllTelemetryIngestPermissions       = "all-telemetry-ingest-permissions"
	)

	createFakeClient := func(t *testing.T) *dtclientmock.Client {
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
			{fakeTokenAllOTLPExporterPermissions, getAllScopesForOTLPExporter()},
			{fakeTokenAllTelemetryIngestPermissions, getAllScopesForTelemetryIngest()},
		}

		for _, tokenScope := range tokenScopes {
			fakeClient.On("GetTokenScopes", mock.Anything, tokenScope.token).
				Return(tokenScope.scopes, nil).Maybe()
		}

		return fakeClient
	}

	enableKubernetesMonitoringAndMetricsIngest := func(dk *dynakube.DynaKube) *dynakube.DynaKube {
		dk.Spec.ActiveGate.Capabilities = []activegate.CapabilityDisplayName{
			activegate.KubeMonCapability.DisplayName,
			activegate.MetricsIngestCapability.DisplayName,
		}

		return dk
	}

	t.Run("empty dynakube, all permissions in api token, but paas => should fail", func(t *testing.T) {
		apiToken := newToken(dtclient.APIToken, fakeTokenAllAPITokenPermissions)
		tokens := Tokens{
			dtclient.APIToken: &apiToken,
		}
		tokens = tokens.AddFeatureScopesToTokens()
		_, err := tokens.VerifyScopes(t.Context(), createFakeClient(t), dynakube.DynaKube{})

		assert.Len(t, tokens.APIToken().Features, 10)
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
		_, err := tokens.VerifyScopes(t.Context(), createFakeClient(t), dynakube.DynaKube{})

		assert.Len(t, tokens.APIToken().Features, 10)
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
		_, err := tokens.VerifyScopes(t.Context(), createFakeClient(t), dynakube.DynaKube{})

		assert.Len(t, tokens.APIToken().Features, 10)
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
		_, err := tokens.VerifyScopes(t.Context(), createFakeClient(t), dk)

		assert.Len(t, tokens.APIToken().Features, 10)
		assert.Empty(t, tokens.PaasToken().Features)
		assert.Empty(t, tokens.DataIngestToken().Features)
		assert.EqualError(t, err, "token 'apiToken' has scope errors: [feature 'Access problem and event feed, metrics, and topology' is missing scope 'DataExport' feature 'Automatic ActiveGate Token Creation' is missing scope 'activeGateTokenManagement.create' feature 'Download Installer' is missing scope 'InstallerDownload']")
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
		_, err := tokens.VerifyScopes(t.Context(), createFakeClient(t), dk)

		assert.Len(t, tokens.APIToken().Features, 10)
		assert.Empty(t, tokens.PaasToken().Features)
		assert.Len(t, tokens.DataIngestToken().Features, 8)
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
		_, err := tokens.VerifyScopes(t.Context(), createFakeClient(t), dynakube.DynaKube{})

		assert.Len(t, tokens.APIToken().Features, 10)
		assert.Empty(t, tokens.PaasToken().Features)
		assert.Len(t, tokens.DataIngestToken().Features, 8)
		assert.NoError(t, err)
	})
	t.Run("otlp exporter configuration enabled => dataingest token missing rights => fail", func(t *testing.T) {
		dk := dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				OTLPExporterConfiguration: &otlp.ExporterConfigurationSpec{
					Signals: otlp.SignalConfiguration{
						Traces:  &otlp.TracesSignal{},
						Metrics: &otlp.MetricsSignal{},
						Logs:    &otlp.LogsSignal{},
					},
				},
			},
		}

		apiToken := newToken(dtclient.APIToken, fakeTokenAllAPITokenPermissionsIncludingPaaS)
		dataingestToken := newToken(dtclient.DataIngestToken, fakeTokenNoPermissions)
		tokens := Tokens{
			dtclient.APIToken:        &apiToken,
			dtclient.DataIngestToken: &dataingestToken,
		}
		tokens = tokens.AddFeatureScopesToTokens()
		_, err := tokens.VerifyScopes(t.Context(), createFakeClient(t), dk)

		assert.Len(t, tokens.APIToken().Features, 10)
		assert.Empty(t, tokens.PaasToken().Features)
		assert.Len(t, tokens.DataIngestToken().Features, 8)
		assert.EqualError(t, err, "token 'dataIngestToken' has scope errors: [feature 'OTLP trace exporter configuration' is missing scope 'openTelemetryTrace.ingest' feature 'OTLP logs exporter configuration' is missing scope 'logs.ingest' feature 'OTLP metrics exporter configuration' is missing scope 'metrics.ingest']")
	})
	t.Run("otlp exporter configuration enabled => dataingest token has rights => success", func(t *testing.T) {
		dk := dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				OTLPExporterConfiguration: &otlp.ExporterConfigurationSpec{
					Signals: otlp.SignalConfiguration{
						Traces:  &otlp.TracesSignal{},
						Metrics: &otlp.MetricsSignal{},
						Logs:    &otlp.LogsSignal{},
					},
				},
			},
		}

		apiToken := newToken(dtclient.APIToken, fakeTokenAllAPITokenPermissionsIncludingPaaS)
		dataingestToken := newToken(dtclient.DataIngestToken, fakeTokenAllOTLPExporterPermissions)
		tokens := Tokens{
			dtclient.APIToken:        &apiToken,
			dtclient.DataIngestToken: &dataingestToken,
		}
		tokens = tokens.AddFeatureScopesToTokens()
		_, err := tokens.VerifyScopes(t.Context(), createFakeClient(t), dk)

		assert.Len(t, tokens.APIToken().Features, 10)
		assert.Empty(t, tokens.PaasToken().Features)
		assert.Len(t, tokens.DataIngestToken().Features, 8)
		assert.NoError(t, err)
	})
}

func TestTokens_VerifyScopes(t *testing.T) {
	type testCase struct {
		title            string
		dk               dynakube.DynaKube
		availableScopes  dtclient.TokenScopes
		expectedOptional map[string]bool
		shouldError      bool
	}

	cases := []testCase{
		{
			title: "kubernetes-monitoring enabled - all scopes present",
			dk: dynakube.DynaKube{
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
			},
			availableScopes: dtclient.TokenScopes{
				dtclient.TokenScopeDataExport,
				dtclient.TokenScopeSettingsRead,
				dtclient.TokenScopeSettingsWrite,
				dtclient.TokenScopeInstallerDownload, // TODO: is this really necessary? I think this is only needed in case of appmon (when we download the zip)
				dtclient.TokenScopeActiveGateTokenCreate,
			},
			expectedOptional: map[string]bool{
				dtclient.TokenScopeSettingsRead:  true,
				dtclient.TokenScopeSettingsWrite: true,
			},
			shouldError: false,
		},
		{
			title: "kubernetes-monitoring enabled - required scopes missing",
			dk: dynakube.DynaKube{
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
			},
			availableScopes: dtclient.TokenScopes{
				dtclient.TokenScopeDataExport,
				dtclient.TokenScopeSettingsRead,
				dtclient.TokenScopeSettingsWrite,
				dtclient.TokenScopeInstallerDownload, // TODO: is this really necessary? I think this is only needed in case of appmon (when we download the zip)
			},
			expectedOptional: map[string]bool{
				dtclient.TokenScopeSettingsRead:  true,
				dtclient.TokenScopeSettingsWrite: true,
			},
			shouldError: true,
		},
		{
			title: "kubernetes-monitoring enabled - optional scopes missing",
			dk: dynakube.DynaKube{
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
			},
			availableScopes: dtclient.TokenScopes{
				dtclient.TokenScopeDataExport,
				dtclient.TokenScopeActiveGateTokenCreate,
				dtclient.TokenScopeInstallerDownload, // TODO: is this really necessary? I think this is only needed in case of appmon (when we download the zip)
			},
			expectedOptional: map[string]bool{
				dtclient.TokenScopeSettingsRead:  false,
				dtclient.TokenScopeSettingsWrite: false,
			},
			shouldError: false,
		},
		{
			title: "metadataEnrichment - all scopes present",
			dk: dynakube.DynaKube{
				Spec: dynakube.DynaKubeSpec{
					MetadataEnrichment: metadataenrichment.Spec{
						Enabled: ptr.To(true),
					},
				},
			},
			availableScopes: dtclient.TokenScopes{
				dtclient.TokenScopeDataExport,
				dtclient.TokenScopeSettingsRead,
				dtclient.TokenScopeInstallerDownload, // TODO: is this really necessary? I think this is only needed in case of appmon (when we download the zip)
			},
			expectedOptional: map[string]bool{
				dtclient.TokenScopeSettingsRead: true,
			},
			shouldError: false,
		},
		{
			title: "metadataEnrichment - required scopes missing", // TODO: related to the other TODOS, this test is a bit "incorrect", as metadataEnrichment doesn't really have required scopes
			dk: dynakube.DynaKube{
				Spec: dynakube.DynaKubeSpec{
					MetadataEnrichment: metadataenrichment.Spec{
						Enabled: ptr.To(true),
					},
				},
			},
			availableScopes: dtclient.TokenScopes{
				dtclient.TokenScopeDataExport,
				dtclient.TokenScopeSettingsRead,
			},
			expectedOptional: map[string]bool{
				dtclient.TokenScopeSettingsRead: true,
			},
			shouldError: true,
		},
		{
			title: "metadataEnrichment - optional scopes missing",
			dk: dynakube.DynaKube{
				Spec: dynakube.DynaKubeSpec{
					MetadataEnrichment: metadataenrichment.Spec{
						Enabled: ptr.To(true),
					},
				},
			},
			availableScopes: dtclient.TokenScopes{
				dtclient.TokenScopeDataExport,
				dtclient.TokenScopeInstallerDownload, // TODO: is this really necessary? I think this is only needed in case of appmon (when we download the zip)
			},
			expectedOptional: map[string]bool{
				dtclient.TokenScopeSettingsRead: false,
			},
			shouldError: false,
		},
		{
			title: "logMonitoring enabled - optional scopes present",
			dk: dynakube.DynaKube{
				Spec: dynakube.DynaKubeSpec{
					LogMonitoring: &logmonitoring.Spec{},
				},
			},
			availableScopes: dtclient.TokenScopes{
				dtclient.TokenScopeDataExport,
				dtclient.TokenScopeSettingsRead,
				dtclient.TokenScopeSettingsWrite,
				dtclient.TokenScopeInstallerDownload, // TODO: is this really necessary? I think this is only needed in case of appmon (when we download the zip)
			},
			expectedOptional: map[string]bool{
				dtclient.TokenScopeSettingsRead:  true,
				dtclient.TokenScopeSettingsWrite: true,
			},
			shouldError: false,
		},
		{
			title: "logMonitoring enabled - optional scopes missing",
			dk: dynakube.DynaKube{
				Spec: dynakube.DynaKubeSpec{
					LogMonitoring: &logmonitoring.Spec{},
				},
			},
			availableScopes: dtclient.TokenScopes{
				dtclient.TokenScopeDataExport,
				dtclient.TokenScopeInstallerDownload, // TODO: is this really necessary? I think this is only needed in case of appmon (when we download the zip)
			},
			expectedOptional: map[string]bool{
				dtclient.TokenScopeSettingsRead:  false,
				dtclient.TokenScopeSettingsWrite: false,
			},
			shouldError: false,
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			tokenValue := "test-token"
			fakeClient := dtclientmock.NewClient(t)
			fakeClient.On("GetTokenScopes", mock.Anything, tokenValue).Return(c.availableScopes, nil)

			apiToken := newToken(dtclient.APIToken, tokenValue)
			tokens := Tokens{
				dtclient.APIToken: &apiToken,
			}
			tokens = tokens.AddFeatureScopesToTokens()
			optionalScopes, err := tokens.VerifyScopes(t.Context(), fakeClient, c.dk)

			assert.Equal(t, c.expectedOptional, optionalScopes)
			assert.Equal(t, c.shouldError, err != nil)
		})
	}
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
			name:              "no errors -> no error",
			encounteredErrors: []error{},
		},
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

			if len(testCase.encounteredErrors) == 0 {
				require.NoError(t, err)
			} else {
				require.EqualError(t, err, testCase.message)
			}
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
