package token

import (
	"errors"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/logmonitoring"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/metadataenrichment"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/otlp"
	tokenclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/token"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/dttoken"
	tokenclientmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func getAllScopesForAPIToken() []string {
	return []string{
		tokenclient.ScopeDataExport,
		tokenclient.ScopeSettingsRead,
		tokenclient.ScopeSettingsWrite,
		tokenclient.ScopeActiveGateTokenCreate,
	}
}

func getAllScopesForPaaSToken() []string {
	return []string{
		tokenclient.ScopeInstallerDownload,
	}
}

func getAllScopesForDataIngest() []string {
	return []string{
		tokenclient.ScopeMetricsIngest,
	}
}

func getAllScopesForTelemetryIngest() []string {
	return []string{
		tokenclient.ScopeMetricsIngest,
		tokenclient.ScopeOpenTelemetryTraceIngest,
		tokenclient.ScopeLogsIngest,
	}
}

func getAllScopesForOTLPExporter() []string {
	return []string{
		tokenclient.ScopeMetricsIngest,
		tokenclient.ScopeOpenTelemetryTraceIngest,
		tokenclient.ScopeLogsIngest,
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

	createFakeClient := func(t *testing.T) *tokenclientmock.Client {
		mockedTokenClient := tokenclientmock.NewClient(t)

		tokenScopes := []struct {
			token  string
			scopes []string
		}{
			{fakeTokenNoPermissions, []string{}},
			{fakeTokenAllAPITokenPermissions, getAllScopesForAPIToken()},
			{fakeTokenAllAPITokenPermissionsIncludingPaaS, append(getAllScopesForAPIToken(), getAllScopesForPaaSToken()...)},
			{fakeTokenPaas, getAllScopesForPaaSToken()},
			{fakeTokenAllDataIngestPermissions, getAllScopesForDataIngest()},
			{fakeTokenAllOTLPExporterPermissions, getAllScopesForOTLPExporter()},
			{fakeTokenAllTelemetryIngestPermissions, getAllScopesForTelemetryIngest()},
		}

		for _, tokenScope := range tokenScopes {
			mockedTokenClient.EXPECT().GetScopes(t.Context(), tokenScope.token).Return(tokenScope.scopes, nil).Maybe()
		}

		return mockedTokenClient
	}

	enableKubernetesMonitoringAndMetricsIngest := func(dk *dynakube.DynaKube) *dynakube.DynaKube {
		dk.Spec.ActiveGate.Capabilities = []activegate.CapabilityDisplayName{
			activegate.KubeMonCapability.DisplayName,
			activegate.MetricsIngestCapability.DisplayName,
		}

		return dk
	}

	t.Run("empty dynakube, all permissions in api token, but paas => should fail", func(t *testing.T) {
		apiToken := newToken(APIKey, fakeTokenAllAPITokenPermissions)
		tokens := Tokens{
			APIKey: &apiToken,
		}
		tokens = tokens.AddFeatureScopesToTokens()
		_, err := tokens.VerifyScopes(t.Context(), createFakeClient(t), dynakube.DynaKube{})

		assert.Len(t, tokens.APIToken().Features, 10)
		assert.Empty(t, tokens.PaasToken().Features)
		assert.Empty(t, tokens.DataIngestToken().Features)

		assert.Equal(t, []string{"InstallerDownload"}, GetMissingScopes(err))
		assert.EqualError(t, err, "token 'apiToken' has scope errors: [feature 'Download Installer' is missing scope 'InstallerDownload']")
	})
	t.Run("empty dynakube, all permissions in api token, but paas + paas token => should work", func(t *testing.T) {
		apiToken := newToken(APIKey, fakeTokenAllAPITokenPermissions)
		paasToken := newToken(PaaSKey, fakeTokenPaas)
		tokens := Tokens{
			APIKey:  &apiToken,
			PaaSKey: &paasToken,
		}
		tokens = tokens.AddFeatureScopesToTokens()
		_, err := tokens.VerifyScopes(t.Context(), createFakeClient(t), dynakube.DynaKube{})

		assert.Len(t, tokens.APIToken().Features, 10)
		assert.Len(t, tokens.PaasToken().Features, 1)
		assert.Empty(t, tokens.DataIngestToken().Features)
		assert.NoError(t, err)
	})
	t.Run("empty dynakube, all permissions in api token => should work", func(t *testing.T) {
		apiToken := newToken(APIKey, fakeTokenAllAPITokenPermissionsIncludingPaaS)
		tokens := Tokens{
			APIKey: &apiToken,
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

		apiToken := newToken(APIKey, fakeTokenNoPermissions)
		tokens := Tokens{
			APIKey: &apiToken,
		}
		tokens = tokens.AddFeatureScopesToTokens()
		_, err := tokens.VerifyScopes(t.Context(), createFakeClient(t), dk)

		assert.Len(t, tokens.APIToken().Features, 10)
		assert.Empty(t, tokens.PaasToken().Features)
		assert.Empty(t, tokens.DataIngestToken().Features)
		assert.Equal(t, []string{"DataExport", "activeGateTokenManagement.create", "InstallerDownload"}, GetMissingScopes(err))
		assert.EqualError(t, err, "token 'apiToken' has scope errors: [feature 'Access problem and event feed, metrics, and topology' is missing scope 'DataExport' feature 'Automatic ActiveGate Token Creation' is missing scope 'activeGateTokenManagement.create' feature 'Download Installer' is missing scope 'InstallerDownload']")
	})
	t.Run("data ingest enabled => dataingest token missing rights => fail", func(t *testing.T) {
		dk := dynakube.DynaKube{}
		enableKubernetesMonitoringAndMetricsIngest(&dk)

		apiToken := newToken(APIKey, fakeTokenAllAPITokenPermissionsIncludingPaaS)
		dataingestToken := newToken(DataIngestKey, fakeTokenNoPermissions)
		tokens := Tokens{
			APIKey:        &apiToken,
			DataIngestKey: &dataingestToken,
		}
		tokens = tokens.AddFeatureScopesToTokens()
		_, err := tokens.VerifyScopes(t.Context(), createFakeClient(t), dk)

		assert.Len(t, tokens.APIToken().Features, 10)
		assert.Empty(t, tokens.PaasToken().Features)
		assert.Len(t, tokens.DataIngestToken().Features, 8)
		assert.Equal(t, []string{"metrics.ingest"}, GetMissingScopes(err))
		assert.EqualError(t, err, "token 'dataIngestToken' has scope errors: [feature 'Data Ingest' is missing scope 'metrics.ingest']")
	})
	t.Run("data ingest enabled => dataingest token has rights => success", func(t *testing.T) {
		apiToken := newToken(APIKey, fakeTokenAllAPITokenPermissionsIncludingPaaS)
		dataingestToken := newToken(DataIngestKey, fakeTokenAllDataIngestPermissions)
		tokens := Tokens{
			APIKey:        &apiToken,
			DataIngestKey: &dataingestToken,
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

		apiToken := newToken(APIKey, fakeTokenAllAPITokenPermissionsIncludingPaaS)
		dataingestToken := newToken(DataIngestKey, fakeTokenNoPermissions)
		tokens := Tokens{
			APIKey:        &apiToken,
			DataIngestKey: &dataingestToken,
		}
		tokens = tokens.AddFeatureScopesToTokens()
		_, err := tokens.VerifyScopes(t.Context(), createFakeClient(t), dk)

		assert.Len(t, tokens.APIToken().Features, 10)
		assert.Empty(t, tokens.PaasToken().Features)
		assert.Len(t, tokens.DataIngestToken().Features, 8)
		assert.Equal(t, []string{"openTelemetryTrace.ingest", "logs.ingest", "metrics.ingest"}, GetMissingScopes(err))
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

		apiToken := newToken(APIKey, fakeTokenAllAPITokenPermissionsIncludingPaaS)
		dataingestToken := newToken(DataIngestKey, fakeTokenAllOTLPExporterPermissions)
		tokens := Tokens{
			APIKey:        &apiToken,
			DataIngestKey: &dataingestToken,
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
		availableScopes  []string
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
			availableScopes: []string{
				tokenclient.ScopeDataExport,
				tokenclient.ScopeSettingsRead,
				tokenclient.ScopeSettingsWrite,
				tokenclient.ScopeInstallerDownload, // TODO: is this really necessary? I think this is only needed in case of appmon (when we download the zip)
				tokenclient.ScopeActiveGateTokenCreate,
			},
			expectedOptional: map[string]bool{
				tokenclient.ScopeSettingsRead:  true,
				tokenclient.ScopeSettingsWrite: true,
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
			availableScopes: []string{
				tokenclient.ScopeDataExport,
				tokenclient.ScopeSettingsRead,
				tokenclient.ScopeSettingsWrite,
				tokenclient.ScopeInstallerDownload, // TODO: is this really necessary? I think this is only needed in case of appmon (when we download the zip)
			},
			expectedOptional: map[string]bool{
				tokenclient.ScopeSettingsRead:  true,
				tokenclient.ScopeSettingsWrite: true,
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
			availableScopes: []string{
				tokenclient.ScopeDataExport,
				tokenclient.ScopeActiveGateTokenCreate,
				tokenclient.ScopeInstallerDownload, // TODO: is this really necessary? I think this is only needed in case of appmon (when we download the zip)
			},
			expectedOptional: map[string]bool{
				tokenclient.ScopeSettingsRead:  false,
				tokenclient.ScopeSettingsWrite: false,
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
			availableScopes: []string{
				tokenclient.ScopeDataExport,
				tokenclient.ScopeSettingsRead,
				tokenclient.ScopeInstallerDownload, // TODO: is this really necessary? I think this is only needed in case of appmon (when we download the zip)
			},
			expectedOptional: map[string]bool{
				tokenclient.ScopeSettingsRead: true,
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
			availableScopes: []string{
				tokenclient.ScopeDataExport,
				tokenclient.ScopeSettingsRead,
			},
			expectedOptional: map[string]bool{
				tokenclient.ScopeSettingsRead: true,
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
			availableScopes: []string{
				tokenclient.ScopeDataExport,
				tokenclient.ScopeInstallerDownload, // TODO: is this really necessary? I think this is only needed in case of appmon (when we download the zip)
			},
			expectedOptional: map[string]bool{
				tokenclient.ScopeSettingsRead: false,
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
			availableScopes: []string{
				tokenclient.ScopeDataExport,
				tokenclient.ScopeSettingsRead,
				tokenclient.ScopeSettingsWrite,
				tokenclient.ScopeInstallerDownload, // TODO: is this really necessary? I think this is only needed in case of appmon (when we download the zip)
			},
			expectedOptional: map[string]bool{
				tokenclient.ScopeSettingsRead:  true,
				tokenclient.ScopeSettingsWrite: true,
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
			availableScopes: []string{
				tokenclient.ScopeDataExport,
				tokenclient.ScopeInstallerDownload, // TODO: is this really necessary? I think this is only needed in case of appmon (when we download the zip)
			},
			expectedOptional: map[string]bool{
				tokenclient.ScopeSettingsRead:  false,
				tokenclient.ScopeSettingsWrite: false,
			},
			shouldError: false,
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			tokenValue := "test-token"
			mockedTokenClient := tokenclientmock.NewClient(t)
			mockedTokenClient.EXPECT().GetScopes(t.Context(), tokenValue).Return(c.availableScopes, nil).Once()

			apiToken := newToken(APIKey, tokenValue)
			tokens := Tokens{
				APIKey: &apiToken,
			}
			tokens = tokens.AddFeatureScopesToTokens()
			optionalScopes, err := tokens.VerifyScopes(t.Context(), mockedTokenClient, c.dk)

			assert.Equal(t, c.expectedOptional, optionalScopes)
			assert.Equal(t, c.shouldError, err != nil)
		})
	}
}

func TestTokens_VerifyValues(t *testing.T) {
	validToken := newToken(APIKey, "valid-value")
	invalidToken := newToken(APIKey, " invalid-value ")

	validTokens := Tokens{
		APIKey: &validToken,
	}
	invalidTokens := Tokens{
		APIKey: &invalidToken,
	}

	require.NoError(t, validTokens.VerifyValues())
	require.EqualError(t, invalidTokens.VerifyValues(), "token 'apiToken' contains leading or trailing whitespaces")
}

func TestCheckForDataIngestToken(t *testing.T) {
	t.Run("data ingest token is present, but empty", func(t *testing.T) {
		tokens := Tokens{
			DataIngestKey: &Token{},
		}

		assert.False(t, CheckForDataIngestToken(tokens))
	})

	t.Run("data ingest token is present and not empty", func(t *testing.T) {
		tokens := Tokens{
			DataIngestKey: &Token{
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

func TestDisableLookupForPlatformToken(t *testing.T) {
	tokens := Tokens{APIKey: &Token{Value: dttoken.PlatformPrefix + "test", Features: []Feature{{Name: "ignoreme"}}}}
	scopes, err := tokens.VerifyScopes(t.Context(), nil, dynakube.DynaKube{})
	require.NoError(t, err)
	assert.Empty(t, scopes)
}

func TestGetMissingScopes(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want []string
	}{
		{
			name: "nil error returns nil",
			err:  nil,
			want: nil,
		},
		{
			name: "single ScopeError returns its missing scopes",
			err: ScopeError{
				Token:         APIKey,
				MissingScopes: []string{"DataExport", "InstallerDownload"},
			},
			want: []string{"DataExport", "InstallerDownload"},
		},
		{
			name: "single ScopeError with no missing scopes returns nil",
			err: ScopeError{
				Token:         APIKey,
				MissingScopes: nil,
			},
			want: nil,
		},
		{
			name: "non-ScopeError returns nil",
			err:  errors.New("some error"),
			want: nil,
		},
		{
			name: "joined ScopeErrors returns all missing scopes",
			err: errors.Join(
				ScopeError{
					Token:         APIKey,
					MissingScopes: []string{"DataExport"},
				},
				ScopeError{
					Token:         PaaSKey,
					MissingScopes: []string{"InstallerDownload"},
				},
			),
			want: []string{"DataExport", "InstallerDownload"},
		},
		{
			name: "joined non-ScopeErrors returns nil",
			err: errors.Join(
				errors.New("some error"),
				errors.New("another error"),
			),
			want: nil,
		},
		{
			name: "joined mixed errors returns only ScopeError missing scopes",
			err: errors.Join(
				ScopeError{
					Token:         APIKey,
					MissingScopes: []string{"DataExport"},
				},
				errors.New("some non-scope error"),
			),
			want: []string{"DataExport"},
		},
	}

	for _, c := range tests {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, GetMissingScopes(c.err))
		})
	}
}
