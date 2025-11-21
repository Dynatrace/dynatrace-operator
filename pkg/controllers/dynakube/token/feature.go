package token

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/envvars"
	"golang.org/x/exp/slices"
)

type Feature struct {
	IsEnabled      func(dk dynakube.DynaKube) bool
	Name           string
	RequiredScopes []string
	OptionalScopes []string
}

func (feature *Feature) CollectMissingRequiredScopes(availableScopes []string) []string {
	missingScopes := make([]string, 0)

	for _, requiredScope := range feature.RequiredScopes {
		if !slices.Contains(availableScopes, requiredScope) {
			missingScopes = append(missingScopes, requiredScope)
		}
	}

	return missingScopes
}

func (feature *Feature) CollectOptionalScopes(availableScopes []string) map[string]bool {
	optionalScopes := map[string]bool{}

	for _, scope := range feature.OptionalScopes {
		if !slices.Contains(availableScopes, scope) {
			optionalScopes[scope] = false
		} else {
			optionalScopes[scope] = true
		}
	}

	return optionalScopes
}

func getFeaturesForAPIToken(paasTokenExists bool) []Feature {
	return []Feature{
		{
			Name:           "Access problem and event feed, metrics, and topology",
			RequiredScopes: []string{dtclient.TokenScopeDataExport},
			IsEnabled: func(dk dynakube.DynaKube) bool {
				return envvars.GetBool(consts.HostAvailabilityDetectionEnvVar, true)
			},
		},
		{
			Name: "Kubernetes API Monitoring",
			OptionalScopes: []string{
				dtclient.TokenScopeSettingsRead,
				dtclient.TokenScopeSettingsWrite},
			IsEnabled: func(dk dynakube.DynaKube) bool {
				return dk.ActiveGate().IsKubernetesMonitoringEnabled() &&
					dk.FF().IsAutomaticK8sAPIMonitoring()
			},
		},
		{
			Name: "LogMonitoring",
			OptionalScopes: []string{
				dtclient.TokenScopeSettingsRead,
				dtclient.TokenScopeSettingsWrite},
			IsEnabled: func(dk dynakube.DynaKube) bool {
				return dk.LogMonitoring().IsEnabled()
			},
		},
		{
			Name: "CodeModule Injection",
			OptionalScopes: []string{
				dtclient.TokenScopeSettingsRead,
			},
			IsEnabled: func(dk dynakube.DynaKube) bool {
				return dk.OneAgent().IsAppInjectionNeeded() // also covers node-image pull
			},
		},
		{
			Name: "TelemetryIngest",
			OptionalScopes: []string{
				dtclient.TokenScopeSettingsRead,
			},
			IsEnabled: func(dk dynakube.DynaKube) bool {
				return dk.TelemetryIngest().IsEnabled()
			},
		},
		{
			Name: "PrometheusExtensions",
			OptionalScopes: []string{
				dtclient.TokenScopeSettingsRead,
			},
			IsEnabled: func(dk dynakube.DynaKube) bool {
				return dk.Extensions().IsPrometheusEnabled()
			},
		},
		{
			Name:           "Automatic ActiveGate Token Creation",
			RequiredScopes: []string{dtclient.TokenScopeActiveGateTokenCreate},
			IsEnabled: func(dk dynakube.DynaKube) bool {
				return dk.ActiveGate().IsEnabled()
			},
		},
		{
			Name:           "Download Installer",
			RequiredScopes: []string{dtclient.TokenScopeInstallerDownload},
			IsEnabled: func(_ dynakube.DynaKube) bool {
				return !paasTokenExists
			},
		},
		{
			Name:           "MetadataEnrichment Rules",
			OptionalScopes: []string{dtclient.TokenScopeSettingsRead},
			IsEnabled: func(dk dynakube.DynaKube) bool {
				return dk.MetadataEnrichment().IsEnabled()
			},
		},
		{
			Name:           "OTLP Auto-configuration",
			OptionalScopes: []string{dtclient.TokenScopeSettingsRead},
			IsEnabled: func(dk dynakube.DynaKube) bool {
				return dk.OTLPExporterConfiguration().IsEnabled()
			},
		},
	}
}

func getFeaturesForPaaSToken() []Feature {
	return []Feature{
		{
			Name:           "PaaS Token",
			RequiredScopes: []string{dtclient.TokenScopeInstallerDownload},
			IsEnabled: func(_ dynakube.DynaKube) bool {
				return true
			},
		},
	}
}

func getFeaturesForDataIngest() []Feature {
	return []Feature{
		{
			Name:           "Data Ingest",
			RequiredScopes: []string{dtclient.TokenScopeMetricsIngest},
			IsEnabled: func(dk dynakube.DynaKube) bool {
				return dk.ActiveGate().IsMetricsIngestEnabled()
			},
		},
		{
			Name: "Telemetry Ingest OTLP",
			RequiredScopes: []string{
				dtclient.TokenScopeMetricsIngest,
				dtclient.TokenScopeOpenTelemetryTraceIngest,
				dtclient.TokenScopeLogsIngest,
			},
			IsEnabled: func(dk dynakube.DynaKube) bool {
				return dk.TelemetryIngest().IsOtlpEnabled()
			},
		},
		{
			Name: "Telemetry Ingest Zipkin",
			RequiredScopes: []string{
				dtclient.TokenScopeOpenTelemetryTraceIngest,
			},
			IsEnabled: func(dk dynakube.DynaKube) bool {
				return dk.TelemetryIngest().IsZipkinEnabled()
			},
		},
		{
			Name: "Telemetry Ingest Jaeger",
			RequiredScopes: []string{
				dtclient.TokenScopeOpenTelemetryTraceIngest,
			},
			IsEnabled: func(dk dynakube.DynaKube) bool {
				return dk.TelemetryIngest().IsZipkinEnabled()
			},
		},
		{
			Name: "Telemetry Ingest StatsD",
			RequiredScopes: []string{
				dtclient.TokenScopeMetricsIngest,
			},
			IsEnabled: func(dk dynakube.DynaKube) bool {
				return dk.TelemetryIngest().IsZipkinEnabled()
			},
		},
		{
			Name:           "OTLP trace exporter configuration",
			RequiredScopes: []string{dtclient.TokenScopeOpenTelemetryTraceIngest},
			IsEnabled: func(dk dynakube.DynaKube) bool {
				return dk.OTLPExporterConfiguration().IsTracesEnabled()
			},
		},
		{
			Name:           "OTLP logs exporter configuration",
			RequiredScopes: []string{dtclient.TokenScopeLogsIngest},
			IsEnabled: func(dk dynakube.DynaKube) bool {
				return dk.OTLPExporterConfiguration().IsLogsEnabled()
			},
		},
		{
			Name:           "OTLP metrics exporter configuration",
			RequiredScopes: []string{dtclient.TokenScopeMetricsIngest},
			IsEnabled: func(dk dynakube.DynaKube) bool {
				return dk.OTLPExporterConfiguration().IsMetricsEnabled()
			},
		},
	}
}
