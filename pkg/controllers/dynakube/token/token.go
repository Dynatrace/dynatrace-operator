package token

import (
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
)

type Token struct {
	Value          string
	RequiredScopes []string
}

func (token Token) setApiTokenScopes(dynakube dynatracev1beta1.DynaKube, hasPaasToken bool) Token {
	token.RequiredScopes = make([]string, 0)

	if !hasPaasToken {
		token.RequiredScopes = append(token.RequiredScopes, dtclient.TokenScopeInstallerDownload)
	}

	token.RequiredScopes = append(token.RequiredScopes, dtclient.TokenScopeDataExport)

	if dynakube.IsKubernetesMonitoringActiveGateEnabled() &&
		dynakube.FeatureAutomaticKubernetesApiMonitoring() {
		token.RequiredScopes = append(token.RequiredScopes,
			dtclient.TokenScopeEntitiesRead,
			dtclient.TokenScopeSettingsRead,
			dtclient.TokenScopeSettingsWrite)
	}

	if dynakube.UseActiveGateAuthToken() {
		token.RequiredScopes = append(token.RequiredScopes,
			dtclient.TokenScopeActiveGateTokenCreate)
	}

	return token
}

func (token Token) setPaasTokenScopes() Token {
	token.RequiredScopes = []string{dtclient.TokenScopeInstallerDownload}
	return token
}

func (token Token) setDataIngestScopes() Token {
	token.RequiredScopes = []string{dtclient.TokenScopeMetricsIngest}
	return token
}

func (token Token) getMissingScopes(scopes dtclient.TokenScopes) []string {
	missingScopes := make([]string, 0)

	for _, requiredScope := range token.RequiredScopes {
		if !scopes.Contains(requiredScope) {
			missingScopes = append(missingScopes, requiredScope)
		}
	}

	return missingScopes
}
