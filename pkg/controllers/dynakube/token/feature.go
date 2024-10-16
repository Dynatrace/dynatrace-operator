package token

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"golang.org/x/exp/slices"
)

type Feature struct {
	IsEnabled      func(dk dynakube.DynaKube) bool
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
			IsEnabled: func(dk dynakube.DynaKube) bool {
				return true
			},
		},
		{
			Name: "Kubernetes API Monitoring",
			RequiredScopes: []string{
				dtclient.TokenScopeEntitiesRead,
				dtclient.TokenScopeSettingsRead,
				dtclient.TokenScopeSettingsWrite},
			IsEnabled: func(dk dynakube.DynaKube) bool {
				return dk.ActiveGate().IsKubernetesMonitoringEnabled() &&
					dk.FeatureAutomaticKubernetesApiMonitoring()
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
