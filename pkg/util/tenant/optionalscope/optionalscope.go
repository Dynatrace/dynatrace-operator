package optionalscope

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/token"
	"k8s.io/utils/ptr"
)

// IsAvailable returns true if the DynaKube status has the matching optional scope.
// Always returns true if the used apiToken is a platform token.
func IsAvailable(dk *dynakube.DynaKube, scope string) bool {
	if ptr.Deref(dk.Status.APIToken.Platform, false) {
		return true
	}

	switch scope {
	case token.ScopeSettingsRead:
		return ptr.Deref(dk.Status.APIToken.AvailableOptionalScopes.SettingsRead, false)
	case token.ScopeSettingsWrite:
		return ptr.Deref(dk.Status.APIToken.AvailableOptionalScopes.SettingsWrite, false)
	}

	return false
}

func SetMissing(dk *dynakube.DynaKube, scope string) {
	switch scope {
	case token.ScopeSettingsRead:
		dk.Status.APIToken.AvailableOptionalScopes.SettingsRead = new(false)
	case token.ScopeSettingsWrite:
		dk.Status.APIToken.AvailableOptionalScopes.SettingsWrite = new(false)
	}
}

func SetAvailable(dk *dynakube.DynaKube, scope string) {
	switch scope {
	case token.ScopeSettingsRead:
		dk.Status.APIToken.AvailableOptionalScopes.SettingsRead = new(true)
	case token.ScopeSettingsWrite:
		dk.Status.APIToken.AvailableOptionalScopes.SettingsWrite = new(true)
	}
}
