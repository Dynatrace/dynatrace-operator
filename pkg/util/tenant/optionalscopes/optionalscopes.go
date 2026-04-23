package optionalscopes

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/token"
)

func IsAvailable(tokenStatus *dynakube.OptionalScopes, scope string) bool {
	switch scope {
	case token.ScopeSettingsRead:
		return tokenStatus.APITokenSettingsReadAvailable
	case token.ScopeSettingsWrite:
		return tokenStatus.APITokenSettingsWriteAvailable
	}

	return false
}

func Missing(tokenStatus *dynakube.OptionalScopes, scope string) {
	switch scope {
	case token.ScopeSettingsRead:
		tokenStatus.APITokenSettingsReadAvailable = false
	case token.ScopeSettingsWrite:
		tokenStatus.APITokenSettingsWriteAvailable = false
	}
}

func Available(tokenStatus *dynakube.OptionalScopes, scope string) {
	switch scope {
	case token.ScopeSettingsRead:
		tokenStatus.APITokenSettingsReadAvailable = true
	case token.ScopeSettingsWrite:
		tokenStatus.APITokenSettingsWriteAvailable = true
	}
}
