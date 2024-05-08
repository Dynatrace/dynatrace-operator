package token

import (
	dynatracev1beta2 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta2/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"golang.org/x/exp/slices"
)

type Feature struct {
	IsEnabled      func(dynakube dynatracev1beta2.DynaKube) bool
	Name           string
	RequiredScopes []string
}

func (feature *Feature) IsScopeMissing(scopes []string) (bool, []string) {
	missingScopes := make([]string, 0)

	for _, requiredScope := range feature.RequiredScopes {
		if !slices.Contains(scopes, requiredScope) {
			missingScopes = append(missingScopes, requiredScope)
		}
	}

	return len(missingScopes) > 0, missingScopes
}

func getFeaturesForAPIToken(paasTokenExists bool) []Feature {
	return []Feature{
		{
			Name:           "Access problem and event feed, metrics, and topology",
			RequiredScopes: []string{dtclient.TokenScopeDataExport},
			IsEnabled: func(dynakube dynatracev1beta2.DynaKube) bool {
				return true
			},
		},
		{
			Name: "Kubernetes API Monitoring",
			RequiredScopes: []string{
				dtclient.TokenScopeEntitiesRead,
				dtclient.TokenScopeSettingsRead,
				dtclient.TokenScopeSettingsWrite},
			IsEnabled: func(dynakube dynatracev1beta2.DynaKube) bool {
				return dynakube.IsKubernetesMonitoringActiveGateEnabled() &&
					dynakube.FeatureAutomaticKubernetesApiMonitoring()
			},
		},
		{
			Name:           "Automatic ActiveGate Token Creation",
			RequiredScopes: []string{dtclient.TokenScopeActiveGateTokenCreate},
			IsEnabled: func(dynakube dynatracev1beta2.DynaKube) bool {
				return dynakube.NeedsActiveGate()
			},
		},
		{
			Name:           "Download Installer",
			RequiredScopes: []string{dtclient.TokenScopeInstallerDownload},
			IsEnabled: func(dynakube dynatracev1beta2.DynaKube) bool {
				return !paasTokenExists
			},
		},
	}
}

func getFeaturesForPaaSToken() []Feature {
	return []Feature{
		{
			Name:           "PaaS Token",
			RequiredScopes: []string{dtclient.TokenScopeInstallerDownload},
			IsEnabled: func(dynakube dynatracev1beta2.DynaKube) bool {
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
			IsEnabled: func(dynakube dynatracev1beta2.DynaKube) bool {
				return dynakube.IsMetricsIngestActiveGateEnabled()
			},
		},
	}
}
