package token

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
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

func (feature *Feature) CollectMissingOptionalScopes(availableScopes []string) []string {
	missingScopes := make([]string, 0)

	for _, optionalScope := range feature.OptionalScopes {
		if !slices.Contains(availableScopes, optionalScope) {
			missingScopes = append(missingScopes, optionalScope)
		}
	}

	return missingScopes
}

func getFeaturesForAPIToken(paasTokenExists bool) []Feature {
	return []Feature{
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
				return dk.MetadataEnrichmentEnabled()
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
	}
}
