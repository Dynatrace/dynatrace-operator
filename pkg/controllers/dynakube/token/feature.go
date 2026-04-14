package token

import (
	"slices"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	tokenclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/token"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/envvars"
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
		optionalScopes[scope] = slices.Contains(availableScopes, scope)
	}

	return optionalScopes
}

func getFeaturesForAPIToken(paasTokenExists bool) []Feature {
	return []Feature{
		{
			Name:           "Access problem and event feed, metrics, and topology",
			RequiredScopes: []string{tokenclient.ScopeDataExport},
			IsEnabled: func(dk dynakube.DynaKube) bool {
				return envvars.GetBool(consts.HostAvailabilityDetectionEnvVar, true)
			},
		},
		{
			Name: "Kubernetes API Monitoring",
			OptionalScopes: []string{
				tokenclient.ScopeSettingsRead,
				tokenclient.ScopeSettingsWrite},
			IsEnabled: func(dk dynakube.DynaKube) bool {
				return dk.ActiveGate().IsKubernetesMonitoringEnabled() &&
					dk.FF().IsAutomaticK8sAPIMonitoring()
			},
		},
		{
			Name: "LogMonitoring",
			OptionalScopes: []string{
				tokenclient.ScopeSettingsRead,
				tokenclient.ScopeSettingsWrite},
			IsEnabled: func(dk dynakube.DynaKube) bool {
				return dk.LogMonitoring().IsEnabled()
			},
		},
		{
			Name: "CodeModule Injection",
			OptionalScopes: []string{
				tokenclient.ScopeSettingsRead,
			},
			IsEnabled: func(dk dynakube.DynaKube) bool {
				return dk.OneAgent().IsAppInjectionNeeded() // also covers node-image pull
			},
		},
		{
			Name: "TelemetryIngest",
			OptionalScopes: []string{
				tokenclient.ScopeSettingsRead,
			},
			IsEnabled: func(dk dynakube.DynaKube) bool {
				return dk.TelemetryIngest().IsEnabled()
			},
		},
		{
			Name: "PrometheusExtensions",
			OptionalScopes: []string{
				tokenclient.ScopeSettingsRead,
			},
			IsEnabled: func(dk dynakube.DynaKube) bool {
				return dk.Extensions().IsPrometheusEnabled()
			},
		},
		{
			Name:           "Automatic ActiveGate Token Creation",
			RequiredScopes: []string{tokenclient.ScopeActiveGateTokenCreate},
			IsEnabled: func(dk dynakube.DynaKube) bool {
				return dk.ActiveGate().IsEnabled()
			},
		},
		{
			Name:           "Download Installer",
			RequiredScopes: []string{tokenclient.ScopeInstallerDownload},
			IsEnabled: func(_ dynakube.DynaKube) bool {
				return !paasTokenExists
			},
		},
		{
			Name:           "MetadataEnrichment Rules",
			OptionalScopes: []string{tokenclient.ScopeSettingsRead},
			IsEnabled: func(dk dynakube.DynaKube) bool {
				return dk.MetadataEnrichment().IsEnabled()
			},
		},
		{
			Name:           "OTLP Auto-configuration",
			OptionalScopes: []string{tokenclient.ScopeSettingsRead},
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
			RequiredScopes: []string{tokenclient.ScopeInstallerDownload},
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
			RequiredScopes: []string{tokenclient.ScopeMetricsIngest},
			IsEnabled: func(dk dynakube.DynaKube) bool {
				return dk.ActiveGate().IsMetricsIngestEnabled()
			},
		},
		{
			Name: "Telemetry Ingest OTLP",
			RequiredScopes: []string{
				tokenclient.ScopeMetricsIngest,
				tokenclient.ScopeOpenTelemetryTraceIngest,
				tokenclient.ScopeLogsIngest,
			},
			IsEnabled: func(dk dynakube.DynaKube) bool {
				return dk.TelemetryIngest().IsOTLPEnabled()
			},
		},
		{
			Name: "Telemetry Ingest Zipkin",
			RequiredScopes: []string{
				tokenclient.ScopeOpenTelemetryTraceIngest,
			},
			IsEnabled: func(dk dynakube.DynaKube) bool {
				return dk.TelemetryIngest().IsZipkinEnabled()
			},
		},
		{
			Name: "Telemetry Ingest Jaeger",
			RequiredScopes: []string{
				tokenclient.ScopeOpenTelemetryTraceIngest,
			},
			IsEnabled: func(dk dynakube.DynaKube) bool {
				return dk.TelemetryIngest().IsZipkinEnabled()
			},
		},
		{
			Name: "Telemetry Ingest StatsD",
			RequiredScopes: []string{
				tokenclient.ScopeMetricsIngest,
			},
			IsEnabled: func(dk dynakube.DynaKube) bool {
				return dk.TelemetryIngest().IsZipkinEnabled()
			},
		},
		{
			Name:           "OTLP trace exporter configuration",
			RequiredScopes: []string{tokenclient.ScopeOpenTelemetryTraceIngest},
			IsEnabled: func(dk dynakube.DynaKube) bool {
				return dk.OTLPExporterConfiguration().IsTracesEnabled()
			},
		},
		{
			Name:           "OTLP logs exporter configuration",
			RequiredScopes: []string{tokenclient.ScopeLogsIngest},
			IsEnabled: func(dk dynakube.DynaKube) bool {
				return dk.OTLPExporterConfiguration().IsLogsEnabled()
			},
		},
		{
			Name:           "OTLP metrics exporter configuration",
			RequiredScopes: []string{tokenclient.ScopeMetricsIngest},
			IsEnabled: func(dk dynakube.DynaKube) bool {
				return dk.OTLPExporterConfiguration().IsMetricsEnabled()
			},
		},
	}
}
